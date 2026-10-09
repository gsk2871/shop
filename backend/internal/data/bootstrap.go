package data

import (
	"context"
	"embed"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"io/fs"
	"sort"
)

//go:embed migration/*.up.sql
var schema embed.FS

// Migrate applies embedded migrations once, serializing concurrent runs.
func Migrate(d *Data) error {
	names, err := fs.Glob(schema, "migration/*.up.sql")
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
		for _, name := range names {
			var count int64
			if err := tx.Table("schema_migration").Where("name = ?", name).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			sql, err := schema.ReadFile(name)
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

// InitAdmin creates the one administrator without overwriting an existing account.
func InitAdmin(ctx context.Context, d *Data, user, password string) error {
	if len(user) == 0 || len(user) > 100 || len(password) < 12 || len(password) > 72 {
		return fmt.Errorf("username required and password must be 12 to 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result := d.DB.WithContext(ctx).Exec("INSERT INTO admin_account(id,username,password_hash) VALUES(1,?,?) ON CONFLICT(id) DO NOTHING", user, string(hash))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("administrator already exists")
	}
	return nil
}
