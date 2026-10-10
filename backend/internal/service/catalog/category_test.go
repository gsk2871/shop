package catalog

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	biz "peptide-site/internal/biz/catalog"
	repo "peptide-site/internal/data/catalog"
	"peptide-site/internal/testutil"
	"testing"
)

func TestPublicCategoryContractAndInvalidFilter(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := repo.NewRepo(d, c)
	cat, err := r.SaveCategory(context.Background(), 0, biz.Category{Name: "Public", Sort: 1})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(r)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/categories", nil))
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != 200 || len(body.Items) != 1 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	if len(body.Items[0]) != 2 || body.Items[0]["id"] != cat.ID || body.Items[0]["name"] != "Public" {
		t.Fatal("management data leaked", body)
	}
	for _, path := range []string{"/api/v1/products?categoryId=abc", "/api/v1/products?categoryId=-1"} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
}
