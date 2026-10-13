package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	"peptide-site/internal/pkg/configreload"
	"peptide-site/internal/testutil"
)

type pausedBody struct {
	first   *bytes.Reader
	rest    *bytes.Reader
	waiting chan struct{}
	resume  chan struct{}
	paused  bool
}

func (b *pausedBody) Read(p []byte) (int, error) {
	if b.first.Len() > 0 {
		return b.first.Read(p)
	}
	if !b.paused {
		b.paused = true
		close(b.waiting)
		<-b.resume
	}
	return b.rest.Read(p)
}
func TestReloadDrainsRequestAndTransaction(t *testing.T) {
	_, c := testutil.Postgres(t)
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime.yaml")
	write := func(c conf.Config) {
		t.Helper()
		body, err := yaml.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(c)
	loader := conf.NewLoader()
	var oldDB *data.Data
	oldClosed := make(chan struct{})
	txStarted, txResume := make(chan struct{}), make(chan struct{})
	oldRoot := ""
	generations := 0
	factory := func(ctx context.Context, c conf.Config) (*configreload.Resource, error) {
		db, closeDB, err := data.NewData(c)
		if err != nil {
			return nil, err
		}
		identity, schema, root, commit, err := db.RuntimeMetadata(ctx, c.Data.Storage.UploadDir)
		if err != nil {
			closeDB()
			return nil, err
		}
		first := generations == 0
		generations++
		if first {
			oldDB = db
			oldRoot = root
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/transaction" {
				body, err := io.ReadAll(r.Body)
				if err != nil || int64(len(body)) > c.Data.Storage.MaxImageBytes {
					w.WriteHeader(400)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"root": root, "limit": c.Data.Storage.MaxImageBytes})
				return
			}
			err := db.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(987654)).Error; err != nil {
					return err
				}
				close(txStarted)
				<-txResume
				return tx.Exec("SELECT 1").Error
			})
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(200)
		})
		return &configreload.Resource{Handler: handler, Identity: identity, Schema: schema, Commit: commit, Close: func() {
			closeDB()
			if first {
				close(oldClosed)
			}
		}}, nil
	}
	m, err := configreload.New(ctx, path, c, loader, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	payload := bytes.Repeat([]byte("x"), 2048)
	split := len(payload) - 1000
	paused := &pausedBody{first: bytes.NewReader(payload[:split]), rest: bytes.NewReader(payload[split:]), waiting: make(chan struct{}), resume: make(chan struct{})}
	// Always release blocked goroutines before Manager.Close, even on failure.
	defer func() {
		select {
		case <-paused.resume:
		default:
			close(paused.resume)
		}
		select {
		case <-txResume:
		default:
			close(txResume)
		}
	}()
	uploadDone, txDone := make(chan struct{}), make(chan struct{})
	upload := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/generation", paused)
	go func() { defer close(uploadDone); m.ServeHTTP(upload, req) }()
	transaction := httptest.NewRecorder()
	go func() {
		defer close(txDone)
		m.ServeHTTP(transaction, httptest.NewRequest("POST", "/transaction", nil))
	}()
	for _, ready := range []chan struct{}{paused.waiting, txStarted} {
		select {
		case <-ready:
		case <-time.After(3 * time.Second):
			t.Fatal("old request did not start")
		}
	}
	next := c
	next.Data.Storage.UploadDir = t.TempDir()
	next.Data.Storage.MaxImageBytes = 10
	next.Data.Database.DSN += " application_name=after-drain"
	write(next)
	if err = m.Reload(ctx, func(conf.Config) (func(), func(), error) { return nil, nil, nil }); err != nil {
		t.Fatal(err)
	}
	select {
	case <-oldClosed:
		t.Fatal("old pool closed while requests active")
	default:
	}
	pool, _ := oldDB.DB.DB()
	if err = pool.PingContext(ctx); err != nil {
		t.Fatal("old pool unavailable", err)
	}
	// New requests use the new limit while an in-flight request pins the old generation.
	fresh := httptest.NewRecorder()
	freshReq := httptest.NewRequest("POST", "/generation", bytes.NewReader(payload))
	m.ServeHTTP(fresh, freshReq)
	if fresh.Code != 400 {
		t.Fatal("new upload policy stale", fresh.Code)
	}
	close(paused.resume)
	close(txResume)
	for _, done := range []chan struct{}{uploadDone, txDone, oldClosed} {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("generation failed to drain")
		}
	}
	if upload.Code != 200 || transaction.Code != 200 {
		t.Fatal(upload.Code, upload.Body.String(), transaction.Code, transaction.Body.String())
	}
	var result struct {
		Root  string `json:"root"`
		Limit int64  `json:"limit"`
	}
	if err := json.Unmarshal(upload.Body.Bytes(), &result); err != nil || result.Root != oldRoot || result.Limit != c.Data.Storage.MaxImageBytes {
		t.Fatal("in-flight generation changed", err, result)
	}
	if err = pool.PingContext(ctx); err == nil {
		t.Fatal("drained pool not closed")
	}
}

var _ io.Reader = (*pausedBody)(nil)
