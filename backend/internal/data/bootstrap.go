package data

import (
	"embed"
	"gorm.io/gorm"
	"io/fs"
	"sort"
)

//go:embed migration/*.up.sql
var embeddedMigrations embed.FS

// Migrate applies embedded migrations once, serializing concurrent runs.
func Migrate(d *Data) error {
	names, err := fs.Glob(embeddedMigrations, "migration/*.up.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(784296315)").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE TABLE IF NOT EXISTS schema_migration (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())").Error; err != nil {
			return err
		}
		if err := tx.Exec(`COMMENT ON TABLE schema_migration IS '数据库结构执行记录';
COMMENT ON COLUMN schema_migration.name IS '结构文件名称';
COMMENT ON COLUMN schema_migration.applied_at IS '执行时间'`).Error; err != nil {
			return err
		}
		for _, name := range names {
			var count int64
			if err := tx.Table("schema_migration").Where("name = ?", name).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			sql, err := embeddedMigrations.ReadFile(name)
			if err != nil {
				return err
			}
			if err := tx.Exec(string(sql)).Error; err != nil {
				return err
			}
			if err := tx.Exec("INSERT INTO schema_migration(name) VALUES(?)", name).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
