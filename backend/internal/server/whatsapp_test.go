package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"peptide-site/internal/biz"
	"peptide-site/internal/service"
	"peptide-site/internal/service/cms"
	"peptide-site/internal/testutil"
	"testing"
)

func TestCountryDatabaseReload(t *testing.T) {
	d, c := testutil.Postgres(t)
	resource, e := NewResources(c, d, service.NewSystem(biz.NewSystem(d)), cms.NewHandler(c))
	if e != nil {
		t.Fatal(e)
	}
	defer resource.Close()
	c.Contact.GeoIPDatabasePath = filepath.Join(t.TempDir(), "missing.mmdb")
	if rejected, e := NewResources(c, d, service.NewSystem(biz.NewSystem(d)), cms.NewHandler(c)); e == nil {
		rejected.Close()
		t.Fatal("bad country candidate accepted")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/whatsapp-status", nil).WithContext(context.Background())
	w := httptest.NewRecorder()
	resource.Handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("original generation unavailable", w.Code, w.Body.String())
	}
}
