// Package news adapts the product aggregate to exact management/public routes.
package news

import (
	"encoding/json"
	"io"
	"net/http"
	catalogBiz "peptide-site/internal/biz/catalog"
	biz "peptide-site/internal/biz/news"
	repo "peptide-site/internal/data/news"
	"peptide-site/internal/pkg/errdef"
	admin "peptide-site/internal/service/admin"
	"strconv"
	"strings"
)

type Handler struct{ Repo *repo.Repo }

func NewHandler(r *repo.Repo) *Handler { return &Handler{r} }
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
func page(r *http.Request) (int, int, error) {
	p, size := 1, 20
	var err error
	if raw := r.URL.Query().Get("page"); raw != "" {
		p, err = strconv.Atoi(raw)
	}
	if err == nil {
		if raw := r.URL.Query().Get("pageSize"); raw != "" {
			size, err = strconv.Atoi(raw)
		}
	}
	if err != nil || p < 1 || p > 10000000 || size < 1 || size > 100 || len(r.URL.Query().Get("q")) > 200 {
		return 0, 0, errdef.Invalid()
	}
	return p, size, nil
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	public := strings.HasPrefix(r.URL.Path, "/api/v1/")
	prefix := "/admin/api/v1/"
	if public {
		prefix = "/api/v1/"
	}
	path := strings.TrimPrefix(r.URL.Path, prefix)
	if path == "articles" && r.Method == "GET" {
		p, size, err := page(r)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		search, status := "", ""
		var pinned *bool
		if !public {
			search = strings.TrimSpace(r.URL.Query().Get("q"))
			status = r.URL.Query().Get("status")
			if status != "" && status != "draft" && status != "published" && status != "offline" {
				admin.Failure(w, r, errdef.Invalid())
				return
			}
			if raw := r.URL.Query().Get("pinned"); raw != "" {
				if raw != "true" && raw != "false" {
					admin.Failure(w, r, errdef.Invalid())
					return
				}
				value := raw == "true"
				pinned = &value
			}
		}
		rows, total, err := h.Repo.List(r.Context(), public, p, size, search, status, pinned)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]any{"items": rows, "total": total, "page": p, "pageSize": size})
		return
	}
	if public {
		if r.Method != "GET" || !strings.HasPrefix(path, "articles/") {
			admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
			return
		}
		row, err := h.Repo.Public(r.Context(), r.PathValue("slug"))
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, row)
		return
	}
	if path == "articles/preview" {
		var in biz.Article
		if !decode(w, r, &in) {
			return
		}
		row, err := h.Repo.Preview(r.Context(), in)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, row)
		return
	}
	id, err := catalogBiz.ParseID(r.PathValue("id"))
	if err != nil {
		admin.Failure(w, r, err)
		return
	}
	if r.Method == "GET" {
		row, err := h.Repo.Get(r.Context(), id)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, row)
		return
	}
	if strings.HasSuffix(path, "/status") {
		var in struct {
			Status          string `json:"status"`
			ExpectedVersion int64  `json:"expectedVersion"`
		}
		if !decode(w, r, &in) {
			return
		}
		row, err := h.Repo.Status(r.Context(), id, in.ExpectedVersion, in.Status)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, row)
		return
	}
	if r.Method == "DELETE" {
		var in struct {
			ExpectedVersion int64 `json:"expectedVersion"`
		}
		if !decode(w, r, &in) {
			return
		}
		if err := h.Repo.Delete(r.Context(), id, in.ExpectedVersion); err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]bool{"ok": true})
		return
	}
	var in biz.Article
	if !decode(w, r, &in) {
		return
	}
	if id != 0 && in.ID != "" && in.ID != strconv.FormatInt(id, 10) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	if id == 0 && r.Method != "POST" {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	row, err := h.Repo.Save(r.Context(), id, in)
	if err != nil {
		admin.Failure(w, r, err)
		return
	}
	status := 200
	if id == 0 {
		status = 201
	}
	admin.Reply(w, status, row)
}
