// Package admin exposes exact registered routes, authentication and authorization.
package admin

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"net"
	"net/http"
	biz "peptide-site/internal/biz/admin"
	repo "peptide-site/internal/data/admin"
	"peptide-site/internal/data/seed"
	"peptide-site/internal/pkg/errdef"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	repo    *repo.Repo
	mux     *http.ServeMux
	content http.Handler
}

func NewHandler(r *repo.Repo, content http.Handler) (*Handler, error) {
	registry, err := seed.Load()
	if err != nil {
		return nil, err
	}
	h := &Handler{r, http.NewServeMux(), content}
	for _, n := range registry.Nodes {
		for _, api := range n.APIs {
			permission := *n.PermissionCode
			h.mux.Handle(api.Method+" "+api.Path, h.guard(permission, []string{"full"}, http.HandlerFunc(h.business)))
		}
	}
	for _, self := range registry.SelfService {
		for _, api := range self.APIs {
			h.mux.Handle(api.Method+" "+api.Path, h.guard("", self.AllowedScopes, http.HandlerFunc(h.self)))
		}
	}
	h.mux.HandleFunc("POST /admin/api/v1/login", h.login)
	h.mux.HandleFunc("POST /admin/api/v1/login/totp", h.loginTOTP)
	h.mux.Handle("/admin/api/v1/", h.guard("__unregistered__", []string{"full"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { Failure(w, r, errdef.Denied()) })))
	return h, nil
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")
	if len(id) < 8 || len(id) > 100 || strings.ContainsAny(id, "\r\n \t") {
		id = repo.Hash(strconv.FormatInt(time.Now().UnixNano(), 10) + r.RemoteAddr)[:24]
	}
	r.Header.Set("X-Request-ID", id)
	w.Header().Set("X-Request-ID", id)
	h.mux.ServeHTTP(w, r)
}
func Reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func Failure(w http.ResponseWriter, r *http.Request, err error) {
	status := 503
	code := "SERVICE_UNAVAILABLE"
	var business *errdef.Error
	if errors.As(err, &business) {
		status = business.Status
		code = business.Code
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		status = 404
		code = "NOT_FOUND"
	} else if strings.Contains(err.Error(), "SQLSTATE 23505") {
		status = 409
		code = "CONFLICT"
	} else {
		slog.ErrorContext(r.Context(), "request failed", "requestId", r.Header.Get("X-Request-ID"))
	}
	Reply(w, status, map[string]string{"code": code, "message": code, "requestId": r.Header.Get("X-Request-ID")})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		Failure(w, r, errdef.Invalid())
		return false
	}
	return true
}
func (h *Handler) csrf(r *http.Request) bool {
	if r.Method == "GET" || r.Method == "HEAD" {
		return true
	}
	if r.Header.Get("X-Requested-With") != "peptide-admin" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	for _, prefix := range h.repo.Config.Auth.TrustedProxies {
		_, network, _ := net.ParseCIDR(prefix)
		if network != nil && network.Contains(net.ParseIP(host)) {
			if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded == "https" || forwarded == "http" {
				scheme = forwarded
			}
			break
		}
	}
	return origin == scheme+"://"+r.Host
}
func (h *Handler) ip(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	address := net.ParseIP(host)
	trusted := false
	for _, prefix := range h.repo.Config.Auth.TrustedProxies {
		_, network, _ := net.ParseCIDR(prefix)
		if network != nil && network.Contains(address) {
			trusted = true
			break
		}
	}
	if trusted {
		forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		for j := len(forwarded) - 1; j >= 0; j-- {
			candidate := net.ParseIP(strings.TrimSpace(forwarded[j]))
			if candidate == nil {
				return host
			}
			isProxy := false
			for _, p := range h.repo.Config.Auth.TrustedProxies {
				_, n, _ := net.ParseCIDR(p)
				if n.Contains(candidate) {
					isProxy = true
					break
				}
			}
			if !isProxy {
				return candidate.String()
			}
		}
	}
	return host
}
func (h *Handler) meta(r *http.Request) repo.Meta {
	m := repo.Meta{IP: h.ip(r), RequestID: r.Header.Get("X-Request-ID")}
	if i := biz.Current(r.Context()); i != nil {
		m.ActorID = i.User.ID
	}
	return m
}
func (h *Handler) guard(permission string, scopes []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.csrf(r) {
			Failure(w, r, errdef.Denied())
			return
		}
		cookie, err := r.Cookie("admin_session")
		if err != nil || len(cookie.Value) != 64 {
			Failure(w, r, errdef.New(401, "AUTHENTICATION_REQUIRED"))
			return
		}
		i, err := h.repo.Identity(r.Context(), cookie.Value)
		if err != nil {
			Failure(w, r, err)
			return
		}
		allowed := false
		for _, scope := range scopes {
			if i.Scope == scope {
				allowed = true
			}
		}
		if !allowed {
			Failure(w, r, errdef.New(403, "ONBOARDING_REQUIRED"))
			return
		}
		if permission != "" && !i.Allows(permission) {
			Failure(w, r, errdef.Denied())
			return
		}
		next.ServeHTTP(w, r.WithContext(biz.WithIdentity(r.Context(), i)))
	})
}
func (h *Handler) limit(w http.ResponseWriter, r *http.Request, keys ...string) bool {
	for _, key := range keys {
		if err := h.repo.Rate(r.Context(), key); err != nil {
			Failure(w, r, err)
			return false
		}
	}
	return true
}
func (h *Handler) cookie(w http.ResponseWriter, value string, ttl int) {
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: value, Path: "/", HttpOnly: true, Secure: h.repo.Config.Auth.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: ttl})
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if !h.csrf(r) {
		Failure(w, r, errdef.Denied())
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Username) > 100 || len(in.Password) > 72 {
		Failure(w, r, errdef.New(401, "INVALID_CREDENTIALS"))
		return
	}
	if !h.limit(w, r, "ip:"+h.ip(r), "account:"+in.Username) {
		return
	}
	result, err := h.repo.Login(r.Context(), in.Username, in.Password, h.meta(r))
	if err != nil {
		_ = h.repo.AuditFailure(r.Context(), h.meta(r), "auth.password")
		Failure(w, r, err)
		return
	}
	if result.Token != "" {
		h.cookie(w, result.Token, result.ExpiresIn)
	}
	Reply(w, 200, result)
}
func (h *Handler) loginTOTP(w http.ResponseWriter, r *http.Request) {
	if !h.csrf(r) {
		Failure(w, r, errdef.Denied())
		return
	}
	var in struct {
		Ticket string `json:"ticket"`
		Code   string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Ticket) != 64 {
		Failure(w, r, errdef.New(401, "MFA_TICKET_INVALID"))
		return
	}
	if !h.limit(w, r, "totp-ip:"+h.ip(r), "ticket:"+in.Ticket) {
		return
	}
	result, err := h.repo.LoginTOTP(r.Context(), in.Ticket, in.Code, h.meta(r))
	if err != nil {
		Failure(w, r, err)
		return
	}
	h.cookie(w, result.Token, result.ExpiresIn)
	Reply(w, 200, result)
}
func (h *Handler) self(w http.ResponseWriter, r *http.Request) {
	i := biz.Current(r.Context())
	m := h.meta(r)
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/v1/")
	switch path {
	case "me", "me/permissions":
		if i.Scope != "full" {
			i.Permissions = []string{}
			i.Menu = []biz.Permission{}
		}
		Reply(w, 200, i)
	case "logout":
		if err := h.repo.Logout(r.Context(), i); err != nil {
			Failure(w, r, err)
			return
		}
		h.cookie(w, "", -1)
		Reply(w, 200, map[string]bool{"ok": true})
	case "password":
		var in struct {
			Current  string `json:"current"`
			Password string `json:"password"`
		}
		if !decode(w, r, &in) {
			return
		}
		if !h.limit(w, r, "password:"+strconv.FormatInt(i.User.ID, 10)) {
			return
		}
		if err := h.repo.Password(r.Context(), i, in.Current, in.Password, m); err != nil {
			Failure(w, r, err)
			return
		}
		h.cookie(w, "", -1)
		Reply(w, 200, map[string]bool{"ok": true})
	case "totp/setup":
		if !h.limit(w, r, "setup:"+strconv.FormatInt(i.User.ID, 10)) {
			return
		}
		secret, uri, err := h.repo.Setup(r.Context(), i, m)
		if err != nil {
			Failure(w, r, err)
			return
		}
		Reply(w, 200, map[string]string{"secret": secret, "uri": uri})
	case "totp/activate":
		var in struct {
			Code string `json:"code"`
		}
		if !decode(w, r, &in) {
			return
		}
		if !h.limit(w, r, "activate:"+strconv.FormatInt(i.User.ID, 10), "totp-ip:"+h.ip(r)) {
			return
		}
		if err := h.repo.Activate(r.Context(), i, in.Code, m); err != nil {
			Failure(w, r, err)
			return
		}
		h.cookie(w, "", -1)
		Reply(w, 200, map[string]bool{"ok": true})
	case "me/reauth":
		var in struct {
			Password string `json:"password"`
			Code     string `json:"code"`
		}
		if !decode(w, r, &in) {
			return
		}
		if !h.limit(w, r, "reauth:"+strconv.FormatInt(i.User.ID, 10)) {
			return
		}
		if err := h.repo.Reauthenticate(r.Context(), i, in.Password, in.Code, m); err != nil {
			Failure(w, r, err)
			return
		}
		Reply(w, 200, map[string]bool{"ok": true})
	default:
		Failure(w, r, errdef.Denied())
	}
}
func page(r *http.Request) (int, int, error) {
	p, size := 1, 20
	var err error
	if raw := r.URL.Query().Get("page"); raw != "" {
		p, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errdef.Invalid()
		}
	}
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		size, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errdef.Invalid()
		}
	}
	if p < 1 || p > 10000000 || size < 1 || size > 100 {
		return 0, 0, errdef.Invalid()
	}
	return p, size, nil
}
func id(r *http.Request) int64 { value, _ := strconv.ParseInt(r.PathValue("id"), 10, 64); return value }
func paginated(w http.ResponseWriter, rows any, total int64, p, size int) {
	Reply(w, 200, map[string]any{"items": rows, "total": total, "page": p, "pageSize": size})
}
func (h *Handler) business(w http.ResponseWriter, r *http.Request) {
	i := biz.Current(r.Context())
	m := h.meta(r)
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/v1/")
	parts := strings.Split(path, "/")
	entity := parts[0]
	objectID := id(r)
	if len(parts) > 1 && objectID <= 0 && (entity == "users" || entity == "roles") {
		Failure(w, r, errdef.Invalid())
		return
	}
	p, size, err := page(r)
	if err != nil {
		Failure(w, r, err)
		return
	}
	switch entity {
	case "users":
		if r.Method == "GET" {
			if objectID > 0 {
				row, err := h.repo.User(r.Context(), objectID)
				if err != nil {
					Failure(w, r, err)
					return
				}
				Reply(w, 200, row)
			} else {
				rows, total, err := h.repo.Users(r.Context(), p, size)
				if err != nil {
					Failure(w, r, err)
					return
				}
				paginated(w, rows, total, p, size)
			}
			return
		}
		if len(parts) > 2 || r.Method == "DELETE" {
			action := "delete"
			if len(parts) > 2 {
				action = parts[2]
			}
			var in struct {
				Enabled  bool   `json:"enabled"`
				Password string `json:"password"`
				Reason   string `json:"reason"`
			}
			if r.Method != "DELETE" && !decode(w, r, &in) {
				return
			}
			if err := h.repo.UserAction(r.Context(), i, objectID, action, in.Password, in.Reason, in.Enabled, m); err != nil {
				Failure(w, r, err)
				return
			}
			Reply(w, 200, map[string]bool{"ok": true})
			return
		}
		var in repo.UserWrite
		if !decode(w, r, &in) {
			return
		}
		row, err := h.repo.WriteUser(r.Context(), i, objectID, in, m)
		if err != nil {
			Failure(w, r, err)
			return
		}
		status := 200
		if objectID == 0 {
			status = 201
		}
		Reply(w, status, row)
	case "roles":
		if len(parts) > 2 {
			if r.Method == "GET" {
				values, err := h.repo.RoleGrants(r.Context(), objectID)
				if err != nil {
					Failure(w, r, err)
					return
				}
				Reply(w, 200, map[string]any{"permissionCodes": values})
				return
			}
			var in struct {
				PermissionCodes []string `json:"permissionCodes"`
			}
			if !decode(w, r, &in) {
				return
			}
			if err := h.repo.Grant(r.Context(), i, objectID, in.PermissionCodes, m); err != nil {
				Failure(w, r, err)
				return
			}
			Reply(w, 200, map[string]bool{"ok": true})
			return
		}
		if r.Method == "GET" {
			rows, total, err := h.repo.Roles(r.Context(), p, size)
			if err != nil {
				Failure(w, r, err)
				return
			}
			if objectID > 0 {
				row, err := h.repo.Role(r.Context(), objectID)
				if err != nil {
					Failure(w, r, err)
					return
				}
				Reply(w, 200, row)
				return
			}
			paginated(w, rows, total, p, size)
			return
		}
		if r.Method == "DELETE" {
			if err := h.repo.DeleteRole(r.Context(), i, objectID, m); err != nil {
				Failure(w, r, err)
				return
			}
			Reply(w, 200, map[string]bool{"ok": true})
			return
		}
		var in repo.RoleWrite
		if !decode(w, r, &in) {
			return
		}
		row, err := h.repo.WriteRole(r.Context(), i, objectID, in, m)
		if err != nil {
			Failure(w, r, err)
			return
		}
		status := 200
		if objectID == 0 {
			status = 201
		}
		Reply(w, status, row)
	case "permissions":
		tree, err := h.repo.Tree(r.Context(), nil, true)
		if err != nil {
			Failure(w, r, err)
			return
		}
		Reply(w, 200, tree)
	case "audit":
		rows, total, err := h.repo.Audits(r.Context(), p, size)
		if err != nil {
			Failure(w, r, err)
			return
		}
		paginated(w, rows, total, p, size)
	default:
		h.content.ServeHTTP(w, r)
	}
}
