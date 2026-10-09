// Package cms implements authenticated CMS and public content endpoints.
package cms

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	biz "peptide-site/internal/biz/cms"
	"peptide-site/internal/conf"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Handler owns CMS dependencies and a bounded login limiter.
type Handler struct {
	repo     biz.Repository
	config   conf.Config
	mux      *http.ServeMux
	mu       sync.Mutex
	attempts map[string]attempt
}
type attempt struct {
	start time.Time
	count int
}

// NewHandler creates routes for the independently deployed site.
func NewHandler(repo biz.Repository, c conf.Config) *Handler {
	h := &Handler{repo: repo, config: c, mux: http.NewServeMux(), attempts: map[string]attempt{}}
	h.mux.HandleFunc("GET /api/v1/products", func(w http.ResponseWriter, r *http.Request) { h.products(w, r, true) })
	h.mux.HandleFunc("GET /api/v1/products/{slug}", h.publicProduct)
	h.mux.HandleFunc("GET /api/v1/site", h.publicSettings)
	h.mux.HandleFunc("GET /api/v1/articles", h.publicArticles)
	h.mux.HandleFunc("GET /api/v1/articles/{slug}", h.publicArticle)
	h.mux.HandleFunc("POST /admin/api/v1/login", h.login)
	h.mux.Handle("/admin/api/v1/", h.protected(http.HandlerFunc(h.admin)))
	return h
}

// ServeHTTP delegates routing to the standard library mux.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("response write failed")
	}
}
func fail(w http.ResponseWriter, status int, msg string) {
	reply(w, status, map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		fail(w, 400, "invalid request")
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		fail(w, 400, "invalid request")
		return false
	}
	return true
}
func (h *Handler) publicSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.repo.Settings(r.Context())
	if err != nil {
		fail(w, 503, "content unavailable")
		return
	}
	reply(w, 200, s)
}
func (h *Handler) publicArticles(w http.ResponseWriter, r *http.Request) {
	rows, err := h.repo.Articles(r.Context(), true)
	if err != nil {
		fail(w, 503, "content unavailable")
		return
	}
	reply(w, 200, rows)
}
func (h *Handler) publicArticle(w http.ResponseWriter, r *http.Request) {
	a, err := h.repo.Article(r.Context(), r.PathValue("slug"))
	if err != nil {
		fail(w, 404, "article unavailable")
		return
	}
	reply(w, 200, a)
}
func tokenHash(s string) string { hash := sha256.Sum256([]byte(s)); return hex.EncodeToString(hash[:]) }
func (h *Handler) allowed(address string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	ip, _, err := net.SplitHostPort(address)
	if err != nil {
		ip = address
	}
	a := h.attempts[ip]
	if now.Sub(a.start) > time.Minute {
		a = attempt{start: now}
	}
	if a.count >= 5 {
		return false
	}
	if len(h.attempts) >= 1024 {
		for k, v := range h.attempts {
			if now.Sub(v.start) > time.Minute {
				delete(h.attempts, k)
			}
		}
		if len(h.attempts) >= 1024 {
			return false
		}
	}
	a.count++
	h.attempts[ip] = a
	return true
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Requested-With") != "peptide-admin" {
		fail(w, 403, "request header required")
		return
	}
	if !h.allowed(r.RemoteAddr) {
		fail(w, 429, "too many login attempts")
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &input) {
		return
	}
	a, err := h.repo.Admin(r.Context())
	if err != nil {
		fail(w, 401, "invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(input.Password)) != nil || input.Username != a.Username {
		fail(w, 401, "invalid credentials")
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		fail(w, 500, "session unavailable")
		return
	}
	token := hex.EncodeToString(raw)
	if err := h.repo.CreateSession(r.Context(), tokenHash(token), time.Now().Add(8*time.Hour)); err != nil {
		fail(w, 500, "session unavailable")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: token, Path: "/", HttpOnly: true, Secure: h.config.Auth.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: 8 * 3600})
	h.audit(r.Context(), "login")
	reply(w, 200, map[string]string{"username": a.Username})
}
func (h *Handler) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Header.Get("X-Requested-With") != "peptide-admin" {
			fail(w, 403, "request header required")
			return
		}
		c, err := r.Cookie("admin_session")
		if err != nil || len(c.Value) != 64 {
			fail(w, 401, "authentication required")
			return
		}
		ok, err := h.repo.Session(r.Context(), tokenHash(c.Value))
		if err != nil {
			fail(w, 503, "authentication unavailable")
			return
		}
		if !ok {
			fail(w, 401, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (h *Handler) audit(ctx context.Context, action string) {
	if err := h.repo.Audit(ctx, action); err != nil {
		slog.Error("audit persistence failed", "action", action)
	}
}
func (h *Handler) admin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/v1/")
	switch {
	case path == "me" && r.Method == http.MethodGet:
		a, err := h.repo.Admin(r.Context())
		if err != nil {
			fail(w, 500, "account unavailable")
			return
		}
		reply(w, 200, map[string]string{"username": a.Username})
	case path == "logout" && r.Method == http.MethodPost:
		c, _ := r.Cookie("admin_session")
		if err := h.repo.DeleteSession(r.Context(), tokenHash(c.Value)); err != nil {
			fail(w, 500, "logout failed")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "admin_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: h.config.Auth.CookieSecure, SameSite: http.SameSiteStrictMode})
		reply(w, 200, map[string]bool{"ok": true})
	case path == "password" && r.Method == http.MethodPost:
		var in struct {
			Current  string `json:"current"`
			Password string `json:"password"`
		}
		if !decode(w, r, &in) {
			return
		}
		a, err := h.repo.Admin(r.Context())
		if err != nil || bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(in.Current)) != nil {
			fail(w, 400, "current password incorrect")
			return
		}
		if len(in.Password) < 12 || len(in.Password) > 72 {
			fail(w, 400, "password must be 12 to 72 bytes")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			fail(w, 500, "password update failed")
			return
		}
		if err := h.repo.Password(r.Context(), string(hash)); err != nil {
			fail(w, 500, "password update failed")
			return
		}
		h.audit(r.Context(), "password.change")
		reply(w, 200, map[string]bool{"ok": true})
	case path == "audit" && r.Method == http.MethodGet:
		rows, err := h.repo.Audits(r.Context())
		if err != nil {
			fail(w, 500, "audit unavailable")
			return
		}
		reply(w, 200, rows)
	case path == "settings" && r.Method == http.MethodGet:
		h.publicSettings(w, r)
	case path == "settings" && r.Method == http.MethodPut:
		var s biz.Settings
		if !decode(w, r, &s) {
			return
		}
		if biz.ValidateSettings(s) != nil {
			fail(w, 400, "invalid branding, color or URL")
			return
		}
		if err := h.repo.SaveSettings(r.Context(), s); err != nil {
			fail(w, 500, "settings save failed")
			return
		}
		h.audit(r.Context(), "settings.save")
		reply(w, 200, s)
	case path == "products" && r.Method == http.MethodGet:
		h.products(w, r, false)
	case path == "products" && (r.Method == http.MethodPost || r.Method == http.MethodPut):
		h.saveProduct(w, r)
	case strings.HasPrefix(path, "products/") && r.Method == http.MethodDelete:
		h.deleteProduct(w, r, strings.TrimPrefix(path, "products/"))
	case path == "articles" && r.Method == http.MethodGet:
		rows, err := h.repo.Articles(r.Context(), false)
		if err != nil {
			fail(w, 500, "articles unavailable")
			return
		}
		reply(w, 200, rows)
	case path == "articles" && (r.Method == http.MethodPost || r.Method == http.MethodPut):
		var a biz.Article
		if !decode(w, r, &a) {
			return
		}
		if (r.Method == http.MethodPost && a.ID != 0) || (r.Method == http.MethodPut && a.ID <= 0) || biz.ValidateArticle(a) != nil {
			fail(w, 400, "invalid article")
			return
		}
		if err := h.repo.SaveArticle(r.Context(), &a); err != nil {
			fail(w, 409, "save failed; check slug and article ID")
			return
		}
		h.audit(r.Context(), "article.save")
		reply(w, 200, a)
	case strings.HasPrefix(path, "articles/") && r.Method == http.MethodDelete:
		id, err := strconv.ParseInt(strings.TrimPrefix(path, "articles/"), 10, 64)
		if err != nil || id <= 0 {
			fail(w, 400, "invalid article ID")
			return
		}
		if err := h.repo.DeleteArticle(r.Context(), id); err != nil {
			fail(w, 404, "article unavailable")
			return
		}
		h.audit(r.Context(), "article.delete")
		reply(w, 200, map[string]bool{"ok": true})
	case path == "media" && r.Method == http.MethodGet:
		rows, err := h.repo.Media(r.Context())
		if err != nil {
			fail(w, 500, "media unavailable")
			return
		}
		reply(w, 200, rows)
	case path == "media" && r.Method == http.MethodPost:
		h.upload(w, r)
	default:
		fail(w, 404, "route not found")
	}
}
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		fail(w, 400, "file too large or invalid upload")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, "file required")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, (20<<20)+1))
	if err != nil || len(body) == 0 || len(body) > 20<<20 {
		fail(w, 400, "file must be between 1 byte and 20MB")
		return
	}
	mime := http.DetectContentType(body)
	ext := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif", "application/pdf": ".pdf", "video/mp4": ".mp4", "video/webm": ".webm"}[mime]
	if ext == "" {
		fail(w, 400, "only JPG, PNG, WebP, GIF, PDF, MP4 and WebM supported")
		return
	}
	name := make([]byte, 16)
	if _, err := rand.Read(name); err != nil {
		fail(w, 500, "upload unavailable")
		return
	}
	filename := hex.EncodeToString(name) + ext
	if err := os.MkdirAll(h.config.Data.Storage.UploadDir, 0750); err != nil {
		fail(w, 500, "storage unavailable")
		return
	}
	full := filepath.Join(h.config.Data.Storage.UploadDir, filename)
	if err := os.WriteFile(full, body, 0640); err != nil {
		fail(w, 500, "upload failed")
		return
	}
	m := biz.Media{Name: filepath.Base(header.Filename), URL: "/uploads/" + filename, MIME: mime, Size: int64(len(body))}
	if err := h.repo.SaveMedia(r.Context(), &m); err != nil {
		if cleanupErr := os.Remove(full); cleanupErr != nil {
			slog.Error("orphan media cleanup failed")
		}
		fail(w, 500, "media record save failed")
		return
	}
	h.audit(r.Context(), "media.upload")
	reply(w, 201, m)
}

// PublicFiles serves uploaded files with browser sniffing disabled.
func (h *Handler) PublicFiles() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(405)
			return
		}
		filename := strings.TrimPrefix(r.URL.Path, "/uploads/")
		if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.HasPrefix(filename, ".") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		http.ServeFile(w, r, filepath.Join(h.config.Data.Storage.UploadDir, filename))
	})
}
