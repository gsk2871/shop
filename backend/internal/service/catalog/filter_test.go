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
