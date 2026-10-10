// Package whatsapp exposes read-only website routes and guarded admin routes.
package whatsapp

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	ids "peptide-site/internal/biz/catalog"
	biz "peptide-site/internal/biz/whatsapp"
	"peptide-site/internal/conf"
	repo "peptide-site/internal/data/whatsapp"
	"peptide-site/internal/pkg/clientip"
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/geoip"
	admin "peptide-site/internal/service/admin"
	"strconv"
	"strings"
)

type Handler struct {
	Repo   *repo.Repo
	Config conf.Config
	Geo    *geoip.Reader
}

func NewHandler(r *repo.Repo, c conf.Config, g *geoip.Reader) *Handler { return &Handler{r, c, g} }
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		admin.Failure(w, r, errdef.Invalid())
		return false
	}
	return true
}
func (h *Handler) trusted(r *http.Request) bool {
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		return false
	}
	ip := net.ParseIP(host)
	for _, prefix := range h.Config.Auth.TrustedProxies {
		_, n, e := net.ParseCIDR(prefix)
		if e == nil && n.Contains(ip) {
			return true
		}
	}
	return false
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var result any
	var err error
	status := 200
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/v1/")
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		if r.Method != "GET" {
			admin.Failure(w, r, errdef.New(405, "METHOD_NOT_ALLOWED"))
			return
		}
		if r.URL.RawQuery != "" || r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
			admin.Failure(w, r, errdef.Invalid())
			return
		}
		if r.URL.Path == "/api/v1/whatsapp-status" {
			var available bool
			available, err = h.Repo.Status(r.Context())
			result = map[string]bool{"available": available, "geoipAvailable": h.Geo.Available()}
		} else {
			if !h.trusted(r) {
				admin.Failure(w, r, errdef.New(403, "UNTRUSTED_PROXY"))
				return
			}
			// Only the dedicated header is accepted on this boundary. Never consult
			// browser-supplied forwarding chains or use RemoteAddr for geolocation.
			headers := http.Header{}
			headers["X-Client-Ip"] = r.Header.Values("X-Client-IP")
			ip, e := clientip.FromHeader(headers)
			if e != nil {
				admin.Failure(w, r, errdef.New(400, "INVALID_CLIENT_IP"))
				return
			}
			country, e := h.Geo.Country(ip)
			if e != nil {
				admin.Failure(w, r, errdef.Internal(e))
				return
			}
			result, err = h.Repo.Contact(r.Context(), country)
		}
	} else if path == "whatsapp-countries" {
		result = map[string]any{"items": biz.Countries()}
	} else if path == "whatsapp-default" {
		var in biz.DefaultInput
		if !decode(w, r, &in) {
			return
		}
		result, err = h.Repo.Default(r.Context(), in)
	} else if path == "whatsapp-groups" && r.Method == "GET" {
		page, size := 1, 20
		if v := r.URL.Query().Get("page"); v != "" {
			page, err = strconv.Atoi(v)
		}
		if v := r.URL.Query().Get("pageSize"); v != "" && err == nil {
			size, err = strconv.Atoi(v)
		}
		q := r.URL.Query().Get("q")
		if err != nil || page < 1 || page > 10000000 || size < 1 || size > 100 || len(q) > 200 {
			admin.Failure(w, r, errdef.Invalid())
			return
		}
		var list biz.List
		list, err = h.Repo.List(r.Context(), page, size, q)
		list.GeoIPAvailable = h.Geo.Available()
		result = list
	} else {
		id := int64(0)
		if path != "whatsapp-groups" {
			id, err = ids.ParseID(r.PathValue("id"))
			if err != nil {
				admin.Failure(w, r, err)
				return
			}
		}
		switch {
		case r.Method == "GET":
			result, err = h.Repo.Get(r.Context(), id)
		case r.Method == "DELETE":
			var in struct {
				ExpectedUpdatedAt string `json:"expectedUpdatedAt"`
			}
			if !decode(w, r, &in) {
				return
			}
			err = h.Repo.Delete(r.Context(), id, in.ExpectedUpdatedAt)
			result = map[string]bool{"ok": true}
		case strings.HasSuffix(path, "/enabled"):
			var in struct {
				ExpectedUpdatedAt string `json:"expectedUpdatedAt"`
				Enabled           *bool  `json:"enabled"`
			}
			if !decode(w, r, &in) {
				return
			}
			if in.Enabled == nil {
				admin.Failure(w, r, errdef.Invalid())
				return
			}
			aid := int64(0)
			if v := r.PathValue("accountId"); v != "" {
				aid, err = ids.ParseID(v)
				if err != nil {
					admin.Failure(w, r, err)
					return
				}
			}
			result, err = h.Repo.Enabled(r.Context(), id, aid, in.ExpectedUpdatedAt, *in.Enabled)
		default:
			var in biz.Input
			if !decode(w, r, &in) {
				return
			}
			result, err = h.Repo.Save(r.Context(), id, in)
			if id == 0 {
				status = 201
			}
		}
	}
	if err != nil {
		admin.Failure(w, r, err)
		return
	}
	admin.Reply(w, status, result)
}
