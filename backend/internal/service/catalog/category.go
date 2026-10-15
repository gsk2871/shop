package catalog

import (
	"net/http"
	biz "peptide-site/internal/biz/catalog"
	"peptide-site/internal/pkg/errdef"
	admin "peptide-site/internal/service/admin"
	"strconv"
	"strings"
)

func (h *Handler) categories(w http.ResponseWriter, r *http.Request, path string, public bool) {
	if r.Method == "GET" && (path == "categories" || path == "categories/options") {
		p, size, err := page(r)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		all := public || path == "categories/options"
		rows, total, err := h.Repo.Categories(r.Context(), p, size, strings.TrimSpace(r.URL.Query().Get("q")), all)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		if all {
			items := []map[string]any{}
			for _, row := range rows {
				items = append(items, map[string]any{"id": row.ID, "name": row.Name, "cover": row.Cover})
			}
			admin.Reply(w, 200, map[string]any{"items": items})
		} else {
			admin.Reply(w, 200, map[string]any{"items": rows, "total": total, "page": p, "pageSize": size})
		}
		return
	}
	if public {
		admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
		return
	}
	id, err := biz.ParseID(r.PathValue("id"))
	if err != nil || (path != "categories" && id == 0) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	if r.Method == "GET" {
		row, err := h.Repo.Category(r.Context(), id)
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
		if err := h.Repo.DeleteCategory(r.Context(), id, in.ExpectedVersion); err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]bool{"ok": true})
		return
	}
	var in biz.Category
	if !decode(w, r, &in) {
		return
	}
	if id != 0 && in.ID != "" && in.ID != strconv.FormatInt(id, 10) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	saved, err := h.Repo.SaveCategory(r.Context(), id, in)
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
