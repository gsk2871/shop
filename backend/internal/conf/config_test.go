package conf

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"HTTP_ADDRESS", "DATABASE_DSN", "UPLOAD_DIR", "COOKIE_SECURE"} {
		old, exists := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if exists {
				_ = os.Setenv(name, old)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	clearEnv(t)
	t.Chdir(t.TempDir())
	if _, err := Load(); err == nil {
		t.Fatal("missing DSN must fail")
	}
	t.Setenv("DATABASE_DSN", "test-dsn")
	c, err := Load()
	if err != nil || c.Server.HTTP.Addr != "127.0.0.1:8080" || !c.Auth.CookieSecure {
		t.Fatalf("unexpected defaults: %v", err)
	}
	t.Setenv("HTTP_ADDRESS", ":9090")
	t.Setenv("UPLOAD_DIR", "/tmp/uploads")
	t.Setenv("COOKIE_SECURE", "false")
	c, err = Load()
	if err != nil || c.Server.HTTP.Addr != ":9090" || c.Data.Storage.UploadDir != "/tmp/uploads" || c.Auth.CookieSecure {
		t.Fatal("environment settings not applied")
	}
}

func TestLoadFile(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := "data:\n  database:\n    dsn: file-dsn\nserver:\n  http:\n    addr: :9090\n    timeout: 15s\nauth:\n  cookie_secure: false\n"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadFile(path)
	if err != nil || c.Data.Database.DSN != "file-dsn" || c.Server.HTTP.Addr != ":9090" || c.Server.HTTP.Timeout != 15*time.Second || c.Auth.CookieSecure {
		t.Fatalf("file settings failed: %v", err)
	}
	t.Setenv("DATABASE_DSN", "override")
	t.Setenv("COOKIE_SECURE", "true")
	c, err = LoadFile(path)
	if err != nil || c.Data.Database.DSN != "override" || !c.Auth.CookieSecure {
		t.Fatal("environment override failed")
	}
	t.Setenv("COOKIE_SECURE", "invalid")
	if _, err := LoadFile(path); err == nil {
		t.Fatal("invalid boolean accepted")
	}
	t.Setenv("COOKIE_SECURE", "true")
	for _, body := range []string{"unknown: true", "data:\n  database:\n    dsn: [broken", "data:\n  database:\n    dsn: secret\n---\nother: true", "server:\n  http:\n    timeout: invalid"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadFile(path); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("invalid YAML accepted or leaked")
		}
	}
	if _, err := LoadFile(path + ".missing"); err == nil {
		t.Fatal("explicit missing file accepted")
	}
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Config)
	}{
		{"name", func(c *Config) { c.App.Name = "" }},
		{"dsn", func(c *Config) { c.Data.Database.DSN = "" }},
		{"uploads", func(c *Config) { c.Data.Storage.UploadDir = "" }},
		{"address", func(c *Config) { c.Server.HTTP.Addr = "invalid" }},
		{"port", func(c *Config) { c.Server.HTTP.Addr = ":70000" }},
		{"timeout", func(c *Config) { c.Server.HTTP.Timeout = -time.Second }},
		{"write budget", func(c *Config) { c.Server.HTTP.WriteTimeout = c.Server.HTTP.ReadTimeout }},
		{"pool", func(c *Config) { c.Data.Database.MaxIdleConns = 11 }},
		{"pool lifetime", func(c *Config) { c.Data.Database.ConnMaxLifetime = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := defaults()
			c.Data.Database.DSN = "password=secret"
			tc.change(&c)
			if err := Validate(c); err == nil || strings.Contains(err.Error(), "secret") {
				t.Fatal("invalid settings accepted or leaked")
			}
		})
	}
}

func TestTrackedTemplate(t *testing.T) {
	clearEnv(t)
	c, err := LoadFile("../../configs/config.local.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.Data.Database.DSN, "REPLACE_WITH_DATABASE_PASSWORD") {
		t.Fatal("template must use a password placeholder")
	}
	if c.Server.HTTP.Timeout != 30*time.Second || c.Data.Database.MaxOpenConns != 10 {
		t.Fatal("template schema changed unexpectedly")
	}
}

func TestDefaultUsesLocalFile(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir("configs", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(DefaultPath, []byte("data:\n  database:\n    dsn: local-dsn\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil || c.Data.Database.DSN != "local-dsn" {
		t.Fatal("default did not load private local settings")
	}
}

func TestConfigFieldsDocumented(t *testing.T) {
	body, err := os.ReadFile("../../../docs/CONFIG_RELOAD_MATRIX.md")
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{}
	configLeaves(reflect.ValueOf(defaults()), "", fields)
	for field := range fields {
		if !strings.Contains(string(body), "| "+field+" |") {
			t.Errorf("configuration field has no consumer registration: %s", field)
		}
	}
	if len(fields) != 33 {
		t.Fatalf("configuration field count changed; update matrix and consumer tests: %d", len(fields))
	}
}
