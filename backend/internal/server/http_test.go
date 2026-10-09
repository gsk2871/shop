package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"peptide-site/internal/biz"
	"peptide-site/internal/conf"
	"peptide-site/internal/service"
	"testing"
	"time"
)

type databaseCheck struct{ err error }

func (d databaseCheck) Check(context.Context) error { return d.err }

func TestHealthRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, path, method string
		failure            error
		status             int
	}{
		{"live", "/healthz", http.MethodGet, nil, 200},
		{"ready", "/readyz", http.MethodGet, nil, 200},
		{"database failure", "/readyz", http.MethodGet, errors.New("private connection details"), 503},
		{"wrong method", "/healthz", http.MethodPost, nil, 405},
		{"missing business API", "/admin/api/v1/products", http.MethodGet, nil, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewHTTPServer(conf.Config{}, service.NewSystem(biz.NewSystem(databaseCheck{err: tc.failure})), nil)
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("got %d, want %d", w.Code, tc.status)
			}
			if tc.status == 503 && w.Body.String() != `{"status":"unavailable"}` {
				t.Fatalf("unsafe response: %s", w.Body.String())
			}
		})
	}
}

func TestHTTPConfiguration(t *testing.T) {
	h := conf.HTTP{Addr: "127.0.0.1:8090", Timeout: 20 * time.Second, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 120 * time.Second}
	c := conf.Config{Server: conf.Server{HTTP: h}}
	s := NewHTTPServer(c, service.NewSystem(biz.NewSystem(databaseCheck{})), nil)
	if s.Server.ReadHeaderTimeout != h.ReadHeaderTimeout || s.Server.ReadTimeout != h.ReadTimeout || s.Server.WriteTimeout != h.WriteTimeout || s.Server.IdleTimeout != h.IdleTimeout {
		t.Fatal("HTTP connection configuration was not applied")
	}
}
