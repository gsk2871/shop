package configreload

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"peptide-site/internal/conf"
)

func TestAtomicReloadAndDrain(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("HTTP_ADDRESS", "")
	t.Setenv("UPLOAD_DIR", "")
	t.Setenv("COOKIE_SECURE", "false")
	path := filepath.Join(t.TempDir(), "config.yaml")
	write := func(name, dsn string) {
		t.Helper()
		b := []byte("app:\n  name: " + name + "\ndata:\n  database:\n    dsn: " + dsn + "\n")
		next := path + ".new"
		if err := os.WriteFile(next, b, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(next, path); err != nil {
			t.Fatal(err)
		}
	}
	write("old", "valid")
	loader := conf.NewLoader()
	c, err := loader.Load(path, false)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	finish := make(chan struct{})
	closed := make(chan struct{})
	var candidates atomic.Int32
	factory := func(ctx context.Context, c conf.Config) (*Resource, error) {
		if c.Data.Database.DSN == "broken" {
			return nil, errors.New("password=do-not-leak")
		}
		number := candidates.Add(1)
		r := &Resource{Identity: "site", Schema: "schema", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if number == 1 {
				close(started)
				<-finish
			}
			s, _ := FromContext(r.Context())
			_, _ = w.Write([]byte(s.Config.App.Name))
		}), Close: func() {
			if number == 1 {
				close(closed)
			}
		}}
		if c.App.Name == "wrong-site" {
			r.Identity = "other"
		}
		return r, nil
	}
	m, err := New(context.Background(), path, c, loader, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	prepare := func(conf.Config) (func(), func(), error) { return nil, nil, nil }
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { m.ServeHTTP(w, httptest.NewRequest("GET", "/", nil)); close(done) }()
	<-started
	write("new", "broken")
	if err := m.Reload(context.Background(), prepare); err == nil || m.Config().App.Name != "old" || m.Status().Error != "resource preparation failed" {
		t.Fatal("failed resource partially applied or leaked")
	}
	write("wrong-site", "valid")
	if err := m.Reload(context.Background(), prepare); err == nil {
		t.Fatal("other site accepted")
	}
	write("new", "valid")
	if err := m.Reload(context.Background(), func(conf.Config) (func(), func(), error) { return nil, nil, errors.New("busy") }); err == nil || m.Config().App.Name != "old" {
		t.Fatal("occupied port accepted")
	}
	if err := m.Reload(context.Background(), prepare); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
		t.Fatal("leased resource closed early")
	default:
	}
	second := httptest.NewRecorder()
	m.ServeHTTP(second, httptest.NewRequest("GET", "/", nil))
	if second.Body.String() != "new" {
		t.Fatal("new request kept old snapshot")
	}
	close(finish)
	<-done
	if w.Body.String() != "old" {
		t.Fatal("in-flight request changed generation")
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("retired generation not drained")
	}
	if m.Status().Version != 2 {
		t.Fatal("failed loads bumped version")
	}
	if err := os.WriteFile(path, []byte("bad: ["), 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.Reload(context.Background(), prepare); err == nil {
		t.Fatal("invalid YAML accepted")
	}
	write("new", "valid")
	if err := m.Reload(context.Background(), prepare); err != nil || m.Status().Error != "" {
		t.Fatal("reverted valid file did not recover")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := m.Reload(context.Background(), prepare); err == nil || m.Status().Version != 2 || m.Config().App.Name != "new" {
		t.Fatal("missing file displaced the last good generation")
	}
	write("new", "valid")
	if err := m.Reload(context.Background(), prepare); err != nil || m.Status().Error != "" || m.Status().Version != 2 {
		t.Fatal("restored file did not recover without replacing equivalent resources")
	}
}
func TestCapturedEnvironment(t *testing.T) {
	t.Setenv("DATABASE_DSN", "original")
	l := conf.NewLoader()
	t.Setenv("DATABASE_DSN", "changed")
	c, err := l.Decode([]byte("data:\n  database:\n    dsn: file-value\n"))
	if err != nil || c.Data.Database.DSN != "original" || l.Sources()["data.database.dsn"] != "environment:DATABASE_DSN" {
		t.Fatal("environment snapshot changed")
	}
}
