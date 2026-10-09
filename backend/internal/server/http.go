// Package server configures the single HTTP backend.
package server

import (
	"context"
	"net/http"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"peptide-site/internal/conf"
	"peptide-site/internal/service"
	cms "peptide-site/internal/service/cms"
)

// NewHTTPServer hosts infrastructure, public content and admin routes on one listener.
func NewHTTPServer(c conf.Config, s *service.System, content *cms.Handler) *khttp.Server {
	h := c.Server.HTTP
	server := khttp.NewServer(khttp.Address(h.Addr), khttp.Timeout(h.Timeout))
	server.Server.ReadHeaderTimeout = h.ReadHeaderTimeout
	server.Server.ReadTimeout = h.ReadTimeout
	server.Server.WriteTimeout = h.WriteTimeout
	server.Server.IdleTimeout = h.IdleTimeout
	server.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	server.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := s.Ready(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	if content != nil {
		server.HandlePrefix("/admin/api/v1/", content)
		server.HandlePrefix("/api/v1/", content)
		server.HandlePrefix("/uploads/", content.PublicFiles())
	}
	return server
}
