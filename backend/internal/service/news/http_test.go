package news

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/news"
	repo "peptide-site/internal/data/news"
	"peptide-site/internal/testutil"
	"strings"
	"testing"
)

func TestInvalidListFilters(t *testing.T) {
	for _, query := range []string{"status=invalid", "pinned=1", "pinned=FALSE", "page=0", "pageSize=101", "q=" + strings.Repeat("x", 201)} {
		t.Run(query, func(t *testing.T) {
			w := httptest.NewRecorder()
			NewHandler(nil).ServeHTTP(w, httptest.NewRequest("GET", "/admin/api/v1/articles?"+query, nil))
			if w.Code != 400 {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestListFilterHTTPContract(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := repo.NewRepo(d, c)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"article.publish"}})
	for _, article := range []biz.Article{
		{Title: "Alpha pinned", Slug: "alpha-pinned", Status: "draft", Pinned: true},
		{Title: "Alpha published", Slug: "alpha-published", Status: "published"},
	} {
		if _, err := r.Save(ctx, 0, article); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct {
		path, slug string
		management bool
	}{
		{"/admin/api/v1/articles?q=ALPHA&status=draft&pinned=true&pageSize=1", "alpha-pinned", true},
		{"/api/v1/articles?status=draft&pinned=true", "alpha-published", false},
	} {
		w := httptest.NewRecorder()
		NewHandler(r).ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
		var response struct {
			Items []map[string]any
			Total int
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || response.Total != 1 || len(response.Items) != 1 || response.Items[0]["slug"] != tt.slug {
			t.Fatal(w.Code, w.Body.String())
		}
		if (response.Items[0]["updatedAt"] != nil) != tt.management {
			t.Fatal("management timestamp contract", w.Body.String())
		}
	}
}
