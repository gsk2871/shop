// Package cms composes exact versioned routes. Authentication is applied by administration.
package cms

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"peptide-site/internal/conf"
	"peptide-site/internal/pkg/configreload"
	"peptide-site/internal/pkg/errdef"
	admin "peptide-site/internal/service/admin"
	"strings"
)

type Handler struct{ administration, catalog, faq, news, branding, whatsapp, mediaHandler, publicMedia http.Handler }

func NewHandler(c conf.Config) *Handler { return &Handler{} }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")
	if len(id) < 8 || len(id) > 100 || strings.ContainsAny(id, "\r\n \t") {
		var b [16]byte
		_, _ = rand.Read(b[:])
		id = hex.EncodeToString(b[:])
	}
	r.Header.Set("X-Request-ID", id)
	w.Header().Set("X-Request-ID", id)
	if strings.HasPrefix(r.URL.Path, "/admin/api/v1/") {
		if h.administration != nil {
			h.administration.ServeHTTP(w, r)
			return
		}
		admin.Failure(w, r, errdef.New(503, "SERVICE_UNAVAILABLE"))
		return
	}
	if r.Method != "GET" {
		admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
		return
	}
	switch {
	case r.URL.Path == "/api/v1/faqs":
		if h.faq != nil {
			h.faq.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/api/v1/contact" || r.URL.Path == "/api/v1/whatsapp-status":
		if h.whatsapp != nil {
			h.whatsapp.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/api/v1/site" || r.URL.Path == "/api/v1/banners":
		if h.branding != nil {
			h.branding.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/api/v1/categories" || r.URL.Path == "/api/v1/products" || strings.HasPrefix(r.URL.Path, "/api/v1/products/"):
		if h.catalog != nil {
			r.SetPathValue("slug", strings.TrimPrefix(r.URL.Path, "/api/v1/products/"))
			h.catalog.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/api/v1/articles" || strings.HasPrefix(r.URL.Path, "/api/v1/articles/"):
		if h.news != nil {
			r.SetPathValue("slug", strings.TrimPrefix(r.URL.Path, "/api/v1/articles/"))
			h.news.ServeHTTP(w, r)
			return
		}
	}
	admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
}
func (h *Handler) admin(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/admin/api/v1/faqs" || strings.HasPrefix(r.URL.Path, "/admin/api/v1/faqs/"):
		if h.faq != nil {
			h.faq.ServeHTTP(w, r)
			return
		}
	case strings.HasPrefix(r.URL.Path, "/admin/api/v1/whatsapp-"):
		if h.whatsapp != nil {
			h.whatsapp.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/admin/api/v1/settings" || r.URL.Path == "/admin/api/v1/banner-slots" || r.URL.Path == "/admin/api/v1/banners" || strings.HasPrefix(r.URL.Path, "/admin/api/v1/banners/"):
		if h.branding != nil {
			h.branding.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/admin/api/v1/categories" || strings.HasPrefix(r.URL.Path, "/admin/api/v1/categories/") || r.URL.Path == "/admin/api/v1/products" || r.URL.Path == "/admin/api/v1/currencies" || strings.HasPrefix(r.URL.Path, "/admin/api/v1/products/"):
		if h.catalog != nil {
			h.catalog.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/admin/api/v1/articles" || strings.HasPrefix(r.URL.Path, "/admin/api/v1/articles/"):
		if h.news != nil {
			h.news.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/admin/api/v1/media" || strings.HasPrefix(r.URL.Path, "/admin/api/v1/media/"):
		if h.mediaHandler != nil {
			h.mediaHandler.ServeHTTP(w, r)
			return
		}
	case r.URL.Path == "/admin/api/v1/config/status":
		if s, ok := configreload.FromContext(r.Context()); ok {
			admin.Reply(w, 200, s.Status)
			return
		}
		admin.Failure(w, r, errdef.New(503, "SERVICE_UNAVAILABLE"))
		return
	}
	admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
}
func (h *Handler) AdministrativeRoutes() http.Handler { return http.HandlerFunc(h.admin) }
func (h *Handler) SetAdministration(v http.Handler)   { h.administration = v }
func (h *Handler) SetCatalog(v http.Handler)          { h.catalog = v }
func (h *Handler) SetNews(v http.Handler)             { h.news = v }
func (h *Handler) SetBranding(v http.Handler)         { h.branding = v }
func (h *Handler) SetMedia(v, files http.Handler)     { h.mediaHandler = v; h.publicMedia = files }
func (h *Handler) PublicFiles() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.publicMedia != nil {
			h.publicMedia.ServeHTTP(w, r)
			return
		}
		admin.Failure(w, r, errdef.New(503, "SERVICE_UNAVAILABLE"))
	})
}

func (h *Handler) SetWhatsApp(v http.Handler) { h.whatsapp = v }

func (h *Handler) SetFAQ(v http.Handler) { h.faq = v }
