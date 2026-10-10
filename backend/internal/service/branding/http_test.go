package branding

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/branding"
	repo "peptide-site/internal/data/branding"
	"peptide-site/internal/testutil"
	"testing"
)

func TestPublicConfiguredContent(t *testing.T) {
	d, _ := testutil.Postgres(t)
	r := repo.NewRepo(d)
	s := biz.Default()
	s.Brand = "Configured brand"
	s.BrandMark = "C·"
	s.AboutTitle = "About from CMS"
	s.AboutContent = "First line\n<script>plain text</script>"
	s.HomeContent = "Configured home"
	s.NewsTitle = "Configured news"
	s.CookieContent = "Configured storage notice"
	s.ExpectedVersion = s.Version
	if _, err := r.SaveSettings(auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}}), s); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	NewHandler(r).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/site", nil))
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"brand": s.Brand, "brandMark": s.BrandMark, "aboutTitle": s.AboutTitle, "aboutContent": s.AboutContent, "homeContent": s.HomeContent, "newsTitle": s.NewsTitle, "cookieContent": s.CookieContent} {
		if result[key] != want {
			t.Errorf("%s = %v, want %q", key, result[key], want)
		}
	}
	for _, key := range []string{"version", "expectedVersion", "logoMediaId"} {
		if _, exists := result[key]; exists {
			t.Errorf("administrative field %s exposed", key)
		}
	}
}
