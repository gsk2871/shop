package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RuntimeMetadata verifies the schema before publishing a candidate pool. A
// path hash is stable across processes; the registry survives directory changes.
func (d *Data) RuntimeMetadata(ctx context.Context, dir string) (identity, schema, root string, commit func() error, err error) {
	pool, e := d.DB.DB()
	if e != nil {
		err = e
		return
	}
	if e = pool.PingContext(ctx); e != nil {
		err = errors.New("database ping failed")
		return
	}
	var site struct{ SiteID string }
	if e = d.DB.WithContext(ctx).Table("site_identity").Where("id = ?", 1).Take(&site).Error; e != nil {
		err = errors.New("runtime schema is not initialized; run migrate")
		return
	}
	identity = site.SiteID
	var migrations []string
	if e = d.DB.WithContext(ctx).Table("schema_migration").Order("name").Pluck("name", &migrations).Error; e != nil {
		err = errors.New("schema check failed")
		return
	}
	expected, e := embeddedMigrations.ReadDir("migration")
	if e != nil {
		err = e
		return
	}
	applied := map[string]bool{}
	for _, name := range migrations {
		applied[name] = true
	}
	for _, entry := range expected {
		if !applied["migration/"+entry.Name()] {
			err = errors.New("database migrations are incomplete")
			return
		}
	}
	schema = strings.Join(migrations, ";")
	var shape string
	if e = d.DB.WithContext(ctx).Raw("SELECT COALESCE(string_agg(table_name || '.' || column_name || ':' || data_type || ':' || is_nullable || ':' || COALESCE(column_default,''), ';' ORDER BY table_name,ordinal_position),'') FROM information_schema.columns WHERE table_schema=current_schema()").Scan(&shape).Error; e != nil {
		err = errors.New("database structure check failed")
		return
	}
	var constraints, indices string
	if e = d.DB.WithContext(ctx).Raw("SELECT COALESCE(string_agg(c.relname||':'||pg_get_constraintdef(k.oid),';' ORDER BY c.relname,k.conname),'') FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema()").Scan(&constraints).Error; e != nil {
		err = e
		return
	}
	if e = d.DB.WithContext(ctx).Raw("SELECT COALESCE(string_agg(indexdef,';' ORDER BY tablename,indexname),'') FROM pg_indexes WHERE schemaname=current_schema()").Scan(&indices).Error; e != nil {
		err = e
		return
	}
	schema += "|" + shape + "|" + constraints + "|" + indices
	absolute, e := filepath.Abs(dir)
	if e != nil {
		err = e
		return
	}
	if e = os.MkdirAll(absolute, 0750); e != nil {
		err = errors.New("storage directory unavailable")
		return
	}
	absolute, e = filepath.EvalSymlinks(absolute)
	if e != nil {
		err = e
		return
	}
	probe, e := os.CreateTemp(absolute, ".write-check-")
	if e != nil {
		err = errors.New("storage directory is not writable")
		return
	}
	name := probe.Name()
	e = probe.Close()
	removeErr := os.Remove(name)
	if e != nil || removeErr != nil {
		err = errors.New("storage directory probe failed")
		return
	}
	sum := sha256.Sum256([]byte(absolute))
	root = hex.EncodeToString(sum[:16])
	commit = func() error {
		commitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return d.DB.WithContext(commitCtx).Exec("INSERT INTO storage_root(id,path) VALUES(?,?) ON CONFLICT(id) DO NOTHING", root, absolute).Error
	}
	return
}
func (d *Data) StoragePath(ctx context.Context, root string) (string, error) {
	var row struct{ Path string }
	err := d.DB.WithContext(ctx).Table("storage_root").Where("id = ?", root).Take(&row).Error
	return row.Path, err
}
