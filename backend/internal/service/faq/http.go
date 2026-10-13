package faq

import (
	"encoding/json"
	"io"
	"net/http"
	catalog "peptide-site/internal/biz/catalog"
	biz "peptide-site/internal/biz/faq"
	repo "peptide-site/internal/data/faq"
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
	public := r.URL.Path == "/api/v1/faqs"
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/v1/")
	if r.Method == "GET" && (public || path == "faqs") {
		p, size, err := page(r)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		rows, total, err := h.Repo.List(r.Context(), p, size, strings.TrimSpace(r.URL.Query().Get("q")), public)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		if public {
			items := []map[string]any{}
			for _, row := range rows {
				items = append(items, map[string]any{"id": row.ID, "question": row.Question, "answer": row.Answer})
			}
			admin.Reply(w, 200, map[string]any{"items": items})
		} else {
			admin.Reply(w, 200, map[string]any{"items": rows, "total": total, "page": p, "pageSize": size})
		}
		return
	}
	if public || !(path == "faqs" && r.Method == "POST" || strings.HasPrefix(path, "faqs/") && (r.Method == "GET" || r.Method == "PUT" || r.Method == "DELETE")) {
		admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
		return
	}
	id, err := catalog.ParseID(r.PathValue("id"))
	if err != nil || (path != "faqs" && id == 0) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	if r.Method == "GET" {
		row, err := h.Repo.FAQ(r.Context(), id)
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
		if err := h.Repo.DeleteFAQ(r.Context(), id, in.ExpectedVersion); err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]bool{"ok": true})
		return
	}
	var in biz.FAQ
	if !decode(w, r, &in) {
		return
	}
	if id != 0 && in.ID != "" && in.ID != strconv.FormatInt(id, 10) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	saved, err := h.Repo.SaveFAQ(r.Context(), id, in)
	if err != nil {
		admin.Failure(w, r, err)
		return
	}
	status := 200
	if id == 0 {
		status = 201
	}
	admin.Reply(w, status, saved)
}
