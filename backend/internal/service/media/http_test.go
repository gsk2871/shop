package media

import (
	"encoding/json"
	"net/http/httptest"
	"peptide-site/internal/conf"
	"strings"
	"testing"
)

func TestImageOnlyPolicy(t *testing.T) {
	c := conf.Config{}
	c.Data.Storage.AllowedTypes = []string{"image/png", "video/mp4", "video/webm", "application/pdf"}
	c.Data.Storage.MaxImageBytes = 1 << 20
	h := &Handler{Config: c}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/admin/api/v1/media/upload-policy", nil))
	var p map[string]any
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &p) != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "video") || strings.Contains(w.Body.String(), "pdf") || p["enabled"] != false {
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
