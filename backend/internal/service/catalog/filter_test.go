package catalog

import (
	"net/http/httptest"
	"testing"
)

func TestInvalidProductStatusFilter(t *testing.T) {
	w := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(w, httptest.NewRequest("GET", "/admin/api/v1/products?status=invalid", nil))
	if w.Code != 400 {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}
}

func TestInvalidHomeFeaturedFilter(t *testing.T) {
	for _, prefix := range []string{"/api/v1/", "/admin/api/v1/"} {
		w := httptest.NewRecorder()
		NewHandler(nil).ServeHTTP(w, httptest.NewRequest("GET", prefix+"products?homeFeatured=invalid", nil))
		if w.Code != 400 {
			t.Fatalf("got %d: %s", w.Code, w.Body.String())
		}
	}
}
