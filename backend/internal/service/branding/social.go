package branding

import (
	"net/http"
	biz "peptide-site/internal/biz/branding"
	ids "peptide-site/internal/biz/catalog"
	"peptide-site/internal/pkg/errdef"
	admin "peptide-site/internal/service/admin"
	"strings"
)

func (h *Handler) social(w http.ResponseWriter, r *http.Request, path string) {
	if path == "social-accounts" && r.Method == "GET" {
		items, err := h.Repo.SocialAccounts(r.Context(), false)
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		admin.Reply(w, 200, map[string]any{"items": items})
		return
	}
	create := path == "social-accounts" && r.Method == "POST"
	id, err := ids.ParseID(r.PathValue("id"))
	if err != nil || (!create && id == 0) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	if r.Method == "DELETE" || (r.Method == "PATCH" && strings.HasSuffix(path, "/enabled")) {
		var in struct {
			ExpectedVersion int64 `json:"expectedVersion"`
			Enabled         bool  `json:"enabled"`
		}
		if !decode(w, r, &in) {
			return
		}
		row, err := h.Repo.ChangeSocialAccount(r.Context(), id, in.ExpectedVersion, in.Enabled, r.Method == "DELETE")
		if err != nil {
			admin.Failure(w, r, err)
			return
		}
		if r.Method == "DELETE" {
			admin.Reply(w, 200, map[string]bool{"ok": true})
		} else {
			admin.Reply(w, 200, row)
		}
		return
	}
	if !create && r.Method != "PUT" {
		admin.Failure(w, r, errdef.New(404, "NOT_FOUND"))
		return
	}
	var in biz.SocialAccount
	if !decode(w, r, &in) {
		return
	}
	row, err := h.Repo.SaveSocialAccount(r.Context(), id, in)
	if err != nil {
		admin.Failure(w, r, err)
		return
	}
	status := 200
	if create {
		status = 201
	}
	admin.Reply(w, status, row)
}
