package cms

import (
	"net/http"
	biz "peptide-site/internal/biz/cms"
	"strconv"
	"strings"
)

func (h *Handler) products(w http.ResponseWriter, r *http.Request, public bool) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(search) > 200 {
		fail(w, 400, "search too long")
		return
	}
	rows, err := h.repo.Products(r.Context(), public, search)
	if err != nil {
		fail(w, 503, "products unavailable")
		return
	}
	reply(w, 200, rows)
}

func (h *Handler) publicProduct(w http.ResponseWriter, r *http.Request) {
	p, err := h.repo.Product(r.Context(), r.PathValue("slug"))
	if err != nil {
		fail(w, 404, "product unavailable")
		return
	}
	reply(w, 200, p)
}

func (h *Handler) saveProduct(w http.ResponseWriter, r *http.Request) {
	var p biz.Product
	if !decode(w, r, &p) {
		return
	}
	if (r.Method == http.MethodPost && p.ID != 0) || (r.Method == http.MethodPut && p.ID <= 0) || biz.ValidateProduct(p) != nil {
		fail(w, 400, "invalid product")
		return
	}
	if err := h.repo.SaveProduct(r.Context(), &p); err != nil {
		fail(w, 409, "save failed; check slug and product ID")
		return
	}
	h.audit(r.Context(), "product.save")
	reply(w, 200, p)
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		fail(w, 400, "invalid product ID")
		return
	}
	if err := h.repo.DeleteProduct(r.Context(), id); err != nil {
		fail(w, 404, "product unavailable")
		return
	}
	h.audit(r.Context(), "product.delete")
	reply(w, 200, map[string]bool{"ok": true})
}
