package media

import (
	"encoding/json"
	"net/http/httptest"
	"peptide-site/internal/conf"
	"reflect"
	"strings"
	"testing"
)

func TestImageOnlyPolicy(t *testing.T) {
	c := conf.Config{}
	c.Data.Storage.AllowedTypes = []string{"image/bmp", "image/heic", "image/svg+xml"}
	c.Data.Storage.MaxImageBytes = 1 << 20
	h := &Handler{Config: c}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/admin/api/v1/media/upload-policy", nil))
	var p struct {
		Enabled           bool              `json:"enabled"`
		AllowedTypes      []string          `json:"allowedTypes"`
		AllowedExtensions map[string]string `json:"allowedExtensions"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &p) != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	if !reflect.DeepEqual(p.AllowedTypes, c.Data.Storage.AllowedTypes) || p.Enabled || p.AllowedExtensions[".heic"] != "image/heic" {
		t.Fatal(w.Body.String())
	}
	for _, body := range []string{"video", "%PDF-1.7", "image"} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "/admin/api/v1/media", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatal("legacy local upload still enabled", w.Code)
		}
	}
}
