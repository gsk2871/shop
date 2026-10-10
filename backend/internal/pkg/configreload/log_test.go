package configreload

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"peptide-site/internal/conf"
	"strings"
	"testing"
)

func TestDynamicLogLevelAndRequestIdentity(t *testing.T) {
	t.Setenv("DATABASE_DSN", "test")
	t.Setenv("COOKIE_SECURE", "false")
	loader := conf.NewLoader()
	c, e := loader.Decode(nil)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "log.yaml")
	m, e := New(context.Background(), path, c, loader, func(context.Context, conf.Config) (*Resource, error) {
		return &Resource{Identity: "test", Schema: "test", Close: func() {}, Handler: http.NotFoundHandler()}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	var out bytes.Buffer
	handler := &runtimeLog{manager: m, inner: slog.NewTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug})}
	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("debug enabled under info")
	}
	_ = os.WriteFile(path, []byte("app:\n  name: changed-log\nlog:\n  level: debug\n"), 0600)
	if e = m.Reload(context.Background(), func(conf.Config) (func(), func(), error) { return nil, nil, nil }); e != nil {
		t.Fatal(e)
	}
	if !handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("debug level did not change")
	}
	logger := slog.New(handler)
	logger.Debug("new")
	pinned := context.WithValue(context.Background(), contextKey{}, Snapshot{Config: c})
	logger.InfoContext(pinned, "pinned")
	if !strings.Contains(out.String(), "app=changed-log") || !strings.Contains(out.String(), "app=peptide-site") {
		t.Fatal("log identity consumer stale", out.String())
	}
}
