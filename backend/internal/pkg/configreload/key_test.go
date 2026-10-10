package configreload

import (
	"context"
	"encoding/base64"
	"gopkg.in/yaml.v3"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"peptide-site/internal/conf"
	"testing"
	"time"
)

func TestKeyRemovalWaitsForOldLease(t *testing.T) {
	for _, n := range []string{"DATABASE_DSN", "HTTP_ADDRESS", "UPLOAD_DIR", "SEED_ADMIN_PASSWORD"} {
		t.Setenv(n, "")
	}
	t.Setenv("COOKIE_SECURE", "false")
	loader := conf.NewLoader()
	c, e := loader.Decode([]byte("data:\n  database:\n    dsn: test\n"))
	if e != nil {
		t.Fatal(e)
	}
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	c.Auth.TOTP.ActiveKeyID = "one"
	c.Auth.TOTP.Keys = map[string]string{"one": key}
	path := filepath.Join(t.TempDir(), "keys.yaml")
	write := func(c conf.Config) { b, _ := yaml.Marshal(c); _ = os.WriteFile(path, b, 0600) }
	write(c)
	started, finish, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	m, e := New(context.Background(), path, c, loader, func(ctx context.Context, c conf.Config) (*Resource, error) {
		return &Resource{Identity: "site", Schema: "schema", Close: func() {}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c.Auth.TOTP.ActiveKeyID == "one" {
				close(started)
				<-finish
			}
		})}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	go func() { m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil)); close(done) }()
	<-started
	prepare := func(conf.Config) (func(), func(), error) { return nil, nil, nil }
	c.Auth.TOTP.ActiveKeyID = "two"
	c.Auth.TOTP.Keys = map[string]string{"one": key, "two": key}
	write(c)
	if e = m.Reload(context.Background(), prepare); e != nil {
		t.Fatal(e)
	}
	c.Auth.TOTP.Keys = map[string]string{"two": key}
	write(c)
	if e = m.Reload(context.Background(), prepare); e == nil {
		t.Fatal("old active key removed during lease")
	}
	close(finish)
	<-done
	deadline := time.Now().Add(time.Second)
	for {
		e = m.Reload(context.Background(), prepare)
		if e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(e)
		}
		time.Sleep(time.Millisecond)
	}
}
