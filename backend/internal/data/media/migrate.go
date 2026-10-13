package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"os"
	"path/filepath"
	biz "peptide-site/internal/biz/media"
	"peptide-site/internal/conf"
	"peptide-site/internal/pkg/cosstore"
	"strconv"
	"strings"
	"time"
)

type MigrationStore interface {
	cosstore.Store
	PutImage(context.Context, cosstore.Location, string, string, []byte) error
}

// MigrateLocal preserves IDs, references, the original local file and its stable
// URL. PublicFiles redirects that URL to COS after the database commit.
func (r *Repo) MigrateLocal(ctx context.Context, c conf.Storage, id int64, store MigrationStore) error {
	if !c.COS.Enabled {
		return errors.New("COS must be enabled for migration")
	}
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row biz.Media
		if err := tx.Table("media").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND available AND deleted_at IS NULL AND media_type='image'", id).Take(&row).Error; err != nil {
			return err
		}
		if row.StorageProvider == "cos" {
			return nil
		}
		var root struct{ Path string }
		if err := tx.Table("storage_root").Where("id=?", row.StorageRootID).Take(&root).Error; err != nil {
			return err
		}
		if filepath.Base(row.StorageKey) != row.StorageKey || strings.HasPrefix(row.StorageKey, ".") {
			return errors.New("invalid local storage key")
		}
		name := filepath.Join(root.Path, row.StorageKey)
		info, err := os.Lstat(name)
		if err != nil {
			return errors.New("local image is missing")
		}
		if !info.Mode().IsRegular() || info.Size() != row.Size || info.Size() > c.MaxImageBytes {
			return errors.New("local image size or type is invalid")
		}
		f, err := os.Open(name)
		if err != nil {
			return errors.New("local image cannot be opened")
		}
		body, err := io.ReadAll(io.LimitReader(f, c.MaxImageBytes+1))
		f.Close()
		if err != nil {
			return errors.New("local image cannot be read")
		}
		if _, _, err := biz.InspectImage(body, row.MIME, c.MaxImageBytes); err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		key := c.COS.KeyPrefix + "/images/migrated-" + strconv.FormatInt(id, 10) + "-" + hex.EncodeToString(digest[:]) + "." + biz.ImageExtension(row.MIME)
		loc := cosstore.Location{Bucket: c.COS.Bucket, Region: c.COS.Region}
		if err := store.PutImage(ctx, loc, key, row.MIME, body); err != nil {
			return err
		}
		received, _, err := store.Read(ctx, loc, key, c.MaxImageBytes)
		if err != nil || !bytes.Equal(received, body) {
			return errors.New("migrated image verification failed")
		}
		batchID, err := randomID()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		b := Batch{ID: batchID, UploaderID: -1, Bucket: loc.Bucket, Region: loc.Region, PublicDomain: strings.TrimRight(c.COS.PublicDomain, "/"), KeyPrefix: c.COS.KeyPrefix, MaxBytes: c.MaxImageBytes, ExpiresAt: now, CleanupAfter: now, CleanedAt: &now, CreatedAt: now}
		if err := tx.Table("media_upload_batch").Create(&b).Error; err != nil {
			return err
		}
		if err := tx.Table("media").Where("id=?", id).Updates(map[string]any{"storage_provider": "cos", "storage_key": key, "storage_root_id": "cos:" + loc.Region + ":" + loc.Bucket, "batch_id": batchID, "temporary_cleaned": true}).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "media.migrate", id)
	})
}
