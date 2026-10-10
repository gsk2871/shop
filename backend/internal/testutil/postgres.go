// Package testutil creates disposable real PostgreSQL fixtures only on an explicit test DSN.
package testutil

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	admin "peptide-site/internal/data/admin"
	"testing"
	"time"
)

func Postgres(t *testing.T) (*data.Data, conf.Config) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_DSN")
	if base == "" {
		t.Skip("TEST_DATABASE_DSN required")
	}
	for _, name := range []string{"DATABASE_DSN", "HTTP_ADDRESS", "UPLOAD_DIR", "SEED_ADMIN_PASSWORD"} {
		t.Setenv(name, "")
	}
	t.Setenv("COOKIE_SECURE", "false")
	loader := conf.NewLoader()
	c, err := loader.Decode([]byte("data:\n  database:\n    dsn: " + base + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	c.Data.Database.DSN = base
	c.Data.Storage.UploadDir = t.TempDir()
	c.Auth.TOTP.ActiveKeyID = "test"
	c.Auth.TOTP.Keys = map[string]string{"test": base64.StdEncoding.EncodeToString(make([]byte, 32))}
	root, closeRoot, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("shop_fixture_%d", time.Now().UnixNano())
	if err := root.DB.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatal(err)
	}
	c.Data.Database.DSN = base + " dbname=" + name
	d, closeDB, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeDB()
		if err := root.DB.Exec("DROP DATABASE " + name + " WITH (FORCE)").Error; err != nil {
			t.Error(err)
		}
		closeRoot()
	})
	if err := data.Migrate(d); err != nil {
		t.Fatal(err)
	}
	if err := admin.Seed(context.Background(), d, admin.SeedOptions{Username: "admin", Password: "test-initial-password"}); err != nil {
		t.Fatal(err)
	}
	return d, c
}
