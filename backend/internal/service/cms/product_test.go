package cms

import (
	"context"
	"encoding/json"
	"errors"
	biz "peptide-site/internal/biz/cms"
	"peptide-site/internal/conf"
	"strings"
	"testing"
)

type productRepo struct {
	fakeRepo
	products []biz.Product
	public   bool
	search   string
	saved    *biz.Product
	deleted  int64
	err      error
}

func (f *productRepo) Products(_ context.Context, public bool, q string) ([]biz.Product, error) {
	f.public = public
	f.search = q
	return f.products, f.err
}
func (f *productRepo) Product(_ context.Context, slug string) (biz.Product, error) {
	for _, p := range f.products {
		if p.Slug == slug && p.Status == "published" {
			return p, nil
		}
	}
	return biz.Product{}, errors.New("missing")
}
func (f *productRepo) SaveProduct(_ context.Context, p *biz.Product) error {
	if f.err != nil {
		return f.err
	}
	p.ID = 1
	f.saved = p
	return nil
}
func (f *productRepo) DeleteProduct(_ context.Context, id int64) error { f.deleted = id; return f.err }

func TestProductEndpoints(t *testing.T) {
	repo := &productRepo{fakeRepo: fakeRepo{authenticated: true}, products: []biz.Product{{Slug: "draft", Status: "draft"}, {Slug: "live", Status: "published"}}}
	h := NewHandler(repo, conf.Config{})
	if w := request(h, "GET", "/api/v1/products?q=CAS", "", false, false); w.Code != 200 || !repo.public || repo.search != "CAS" {
		t.Fatal("public search failed")
	}
	if w := request(h, "GET", "/admin/api/v1/products", "", true, false); w.Code != 200 || repo.public {
		t.Fatal("admin list failed")
	}
	for _, slug := range []string{"draft", "missing"} {
		if w := request(h, "GET", "/api/v1/products/"+slug, "", false, false); w.Code != 404 {
			t.Fatal("nonpublic product exposed")
		}
	}
	if w := request(h, "GET", "/api/v1/products/live", "", false, false); w.Code != 200 {
		t.Fatal("published detail unavailable")
	}
	p := biz.Product{Name: "Test", Slug: "test", Status: "published", Images: []string{"/uploads/a.png", "/uploads/b.png"}}
	body, _ := json.Marshal(p)
	for _, tc := range []struct {
		cookie, header bool
		code           int
	}{{false, true, 401}, {true, false, 403}, {true, true, 200}} {
		if w := request(h, "POST", "/admin/api/v1/products", string(body), tc.cookie, tc.header); w.Code != tc.code {
			t.Fatalf("save status %d: %s", w.Code, w.Body.String())
		}
	}
	if repo.saved == nil || len(repo.saved.Images) != 2 || repo.actions[len(repo.actions)-1] != "product.save" {
		t.Fatal("gallery or audit missing")
	}
	for _, method := range []string{"POST", "PUT"} {
		invalid := p
		if method == "POST" {
			invalid.ID = 1
		} else {
			invalid.ID = 0
		}
		body, _ := json.Marshal(invalid)
		if w := request(h, method, "/admin/api/v1/products", string(body), true, true); w.Code != 400 {
			t.Fatal("invalid ID accepted")
		}
	}
	if w := request(h, "DELETE", "/admin/api/v1/products/1", "", true, true); w.Code != 200 || repo.deleted != 1 {
		t.Fatal("delete failed")
	}
	if w := request(h, "DELETE", "/admin/api/v1/products/0", "", true, true); w.Code != 400 {
		t.Fatal("invalid delete accepted")
	}
	repo.err = errors.New("database secret")
	if w := request(h, "POST", "/admin/api/v1/products", string(body), true, true); w.Code != 409 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal("database error exposed")
	}
}
