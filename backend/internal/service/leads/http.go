package leads

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	ids "peptide-site/internal/biz/catalog"
	biz "peptide-site/internal/biz/leads"
	"peptide-site/internal/conf"
	"peptide-site/internal/pkg/clientip"
	"peptide-site/internal/pkg/errdef"
	admin "peptide-site/internal/service/admin"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Store interface {
	Config(context.Context) (biz.Config, error)
	SaveConfig(context.Context, biz.Config) (biz.Config, error)
	Submit(context.Context, string) error
	List(context.Context, biz.Filter) ([]biz.Lead, int64, error)
	Update(context.Context, int64, biz.Update) error
}
type window struct {
	start time.Time
	count int
}
type limiter struct {
	mu      sync.Mutex
	clients map[string]window
	global  window
}

// Shared across runtime configuration generations; never reset on config reload.
var submissions = &limiter{clients: map[string]window{}}

func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, w := range l.clients {
		if now.Sub(w.start) >= 10*time.Minute {
			delete(l.clients, k)
		}
	}
	if now.Sub(l.global.start) >= time.Minute {
		l.global = window{start: now}
	}
	if l.global.count >= 60 {
		return false
	}
	w, exists := l.clients[key]
	if !exists {
		if len(l.clients) >= 4096 {
			return false
		}
		w = window{start: now}
	}
	if w.count >= 5 {
		return false
	}
	w.count++
	l.clients[key] = w
	l.global.count++
	return true
}

type Handler struct {
	Repo   Store
	Config conf.Config
	limit  *limiter
}

func NewHandler(r Store, c conf.Config) *Handler { return &Handler{r, c, submissions} }
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		admin.Failure(w, r, errdef.Invalid())
		return false
	}
	return true
}
func (h *Handler) visitor(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", errdef.Denied()
	}
	trusted := false
	for _, raw := range h.Config.Auth.TrustedProxies {
		_, network, e := net.ParseCIDR(raw)
		if e == nil && network.Contains(net.ParseIP(host)) {
			trusted = true
			break
		}
	}
	if !trusted {
		return "", errdef.New(403, "UNTRUSTED_PROXY")
	}
	headers := http.Header{}
	headers["X-Client-Ip"] = r.Header.Values("X-Client-IP")
	ip, err := clientip.FromHeader(headers)
	if err != nil {
		return "", errdef.Invalid()
	}
	return ip.String(), nil
}
func filter(r *http.Request) (biz.Filter, error) {
	f := biz.Filter{Contact: strings.TrimSpace(r.URL.Query().Get("contact")), Page: 1, Size: 20}
	var err error
	if v := r.URL.Query().Get("page"); v != "" {
		f.Page, err = strconv.Atoi(v)
		if err != nil {
			return f, errdef.Invalid()
		}
	}
	if v := r.URL.Query().Get("pageSize"); v != "" {
		f.Size, err = strconv.Atoi(v)
		if err != nil {
			return f, errdef.Invalid()
		}
	}
	if len(f.Contact) > 254 || f.Page < 1 || f.Page > 1000000 || f.Size < 1 || f.Size > 100 {
		return f, errdef.Invalid()
	}
	for key, dest := range map[string]**time.Time{"start": &f.Start, "end": &f.End} {
		if raw := r.URL.Query().Get(key); raw != "" {
			t, e := time.Parse(time.RFC3339, raw)
			if e != nil {
				return f, errdef.Invalid()
			}
			*dest = &t
		}
	}
	if f.Start != nil && f.End != nil && f.Start.After(*f.End) {
		return f, errdef.Invalid()
	}
	return f, nil
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	public := strings.HasPrefix(r.URL.Path, "/api/v1/")
	path := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/admin"), "/api/v1/")
	if public && path == "leads" && r.Method == "POST" {
		key, err := h.visitor(r)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		if !h.limit.allow(key, time.Now()) {
			w.Header().Set("Retry-After", "600")
			admin.Failure(w, r, errdef.New(429, "RATE_LIMITED"))
			return
		}
		var in struct {
			Contact string `json:"contact"`
		}
		if !decode(w, r, &in) {
			return
		}
		if err = h.Repo.Submit(r.Context(), in.Contact); err != nil {
			admin.Failure(w, r, err)
			return
		}
		// Never expose whether this contact already exists or its management state.
		admin.Reply(w, 200, map[string]bool{"ok": true})
		return
	}
	if path == "lead-popup" && r.Method == "GET" {
		c, err := h.Repo.Config(r.Context())
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		if public {
			admin.Reply(w, 200, biz.Public(c))
		} else {
			admin.Reply(w, 200, c)
		}
		return
	}
	if public {
		admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
		return
	}
	if path == "lead-popup" && r.Method == "PUT" {
		var c biz.Config
		if !decode(w, r, &c) {
			return
		}
		saved, err := h.Repo.SaveConfig(r.Context(), c)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, saved)
		return
	}
	if path == "leads" && r.Method == "GET" {
		f, err := filter(r)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		rows, total, err := h.Repo.List(r.Context(), f)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]any{"items": rows, "total": total, "page": f.Page, "pageSize": f.Size})
		return
	}
	if strings.HasPrefix(path, "leads/") && r.Method == "PATCH" {
		id, err := ids.ParseID(r.PathValue("id"))
		if err != nil || id == 0 {
			admin.Failure(w, r, errdef.Invalid())
			return
		}
		var in biz.Update
		if !decode(w, r, &in) {
			return
		}
		if err = h.Repo.Update(r.Context(), id, in); err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]bool{"ok": true})
		return
	}
	admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
}
