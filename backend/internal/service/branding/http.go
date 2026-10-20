package branding

import (
	"encoding/json"
	"io"
	"net/http"
	biz "peptide-site/internal/biz/branding"
	ids "peptide-site/internal/biz/catalog"
	repo "peptide-site/internal/data/branding"
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
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/v1/")
	if public {
		path = strings.TrimPrefix(r.URL.Path, "/api/v1/")
	}
	if path == "social-accounts" || strings.HasPrefix(path, "social-accounts/") {
		h.social(w, r, path)
		return
	}
	if path == "site" || path == "settings" {
		if r.Method == "GET" {
			s, e := h.Repo.Settings(r.Context())
			if e != nil {
				admin.Failure(w, r, e)
				return
			}
			if public {
				accounts, err := h.Repo.SocialAccounts(r.Context(), true)
				if err != nil {
					admin.Failure(w, r, err)
					return
				}
				admin.Reply(w, 200, map[string]any{"googleAds": s.GoogleAds, "socialAccounts": biz.PublicSocialAccounts(accounts), "brand": s.Brand, "logo": s.Logo, "theme": s.Theme, "description": s.Description, "email": s.Email, "whatsapp": s.WhatsApp, "disclaimer": s.Disclaimer, "brandMark": s.BrandMark, "homeTitle": s.HomeTitle, "homeContent": s.HomeContent, "aboutTitle": s.AboutTitle, "aboutContent": s.AboutContent, "privacyTitle": s.PrivacyTitle, "privacyContent": s.PrivacyContent, "productsTitle": s.ProductsTitle, "researchTitle": s.ResearchTitle, "newsTitle": s.NewsTitle, "cookieTitle": s.CookieTitle, "cookieContent": s.CookieContent})
			} else {
				admin.Reply(w, 200, s)
			}
			return
		}
		var s biz.Settings
		if !decode(w, r, &s) {
			return
		}
		saved, e := h.Repo.SaveSettings(r.Context(), s)
		if e != nil {
			admin.Failure(w, r, e)
			return
		}
		admin.Reply(w, 200, saved)
		return
	}
	if path == "banner-slots" {
		rows, e := h.Repo.Slots(r.Context())
		if e != nil {
			admin.Failure(w, r, e)
			return
		}
		admin.Reply(w, 200, map[string]any{"items": rows})
		return
	}
	if path == "banners" && r.Method == "GET" {
		p, size, e := page(r)
		if e != nil {
			admin.Failure(w, r, e)
			return
		}
		slot := r.URL.Query().Get("slot")
		if public && slot == "" {
			admin.Failure(w, r, errdef.Invalid())
			return
		}
		rows, total, e := h.Repo.List(r.Context(), slot, p, size, public)
		if e != nil {
			admin.Failure(w, r, e)
			return
		}
		if public {
			admin.Reply(w, 200, map[string]any{"items": rows})
		} else {
			admin.Reply(w, 200, map[string]any{"items": rows, "total": total, "page": p, "pageSize": size})
		}
		return
	}
	id, e := ids.ParseID(r.PathValue("id"))
	if e != nil {
		admin.Failure(w, r, e)
		return
	}
	if r.Method == "GET" {
		row, e := h.Repo.Get(r.Context(), id)
		if e != nil {
			admin.Failure(w, r, e)
			return
		}
		admin.Reply(w, 200, row)
		return
	}
	if strings.HasSuffix(path, "/enabled") || r.Method == "DELETE" {
		var in struct {
			ExpectedVersion int64 `json:"expectedVersion"`
			Enabled         bool  `json:"enabled"`
		}
		if !decode(w, r, &in) {
			return
		}
		if r.Method == "DELETE" {
			e = h.Repo.Delete(r.Context(), id, in.ExpectedVersion)
			if e != nil {
				admin.Failure(w, r, e)
				return
			}
			admin.Reply(w, 200, map[string]bool{"ok": true})
			return
		}
		row, e := h.Repo.Enabled(r.Context(), id, in.ExpectedVersion, in.Enabled)
		if e != nil {
			admin.Failure(w, r, e)
			return
		}
		admin.Reply(w, 200, row)
		return
	}
	var in biz.Banner
	if !decode(w, r, &in) {
		return
	}
	if id != 0 && in.ID != "" && in.ID != strconv.FormatInt(id, 10) {
		admin.Failure(w, r, errdef.Invalid())
		return
	}
	row, e := h.Repo.Save(r.Context(), id, in)
	if e != nil {
		admin.Failure(w, r, e)
		return
	}
	status := 200
	if id == 0 {
		status = 201
	}
	admin.Reply(w, status, row)
}
