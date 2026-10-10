package whatsapp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	biz "peptide-site/internal/biz/whatsapp"
	repo "peptide-site/internal/data/whatsapp"
	"peptide-site/internal/pkg/geoip"
	service "peptide-site/internal/service/whatsapp"
	"peptide-site/internal/testutil"
	"strconv"
	"testing"
)

func TestPublicReadAndHeaderBoundary(t *testing.T) {
	d, c := testutil.Postgres(t)
	c.Auth.TrustedProxies = []string{"127.0.0.1/32"}
	geo, e := geoip.Open(os.Getenv("TEST_GEOIP_PATH"))
	if e != nil {
		t.Fatal(e)
	}
	defer geo.Close()
	r := repo.NewRepo(d)
	h := service.NewHandler(r, c, geo)
	request := func(method, path, remote string, headers http.Header) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = remote
		req.Header = headers
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	for _, tc := range []struct {
		method, path, remote string
		headers              http.Header
		want                 int
	}{
		{"GET", "/api/v1/whatsapp-contact", "127.0.0.1:1234", http.Header{}, 400},
		{"GET", "/api/v1/whatsapp-contact", "127.0.0.1:1234", http.Header{"X-Forwarded-For": {"8.8.8.8"}}, 400},
		{"GET", "/api/v1/whatsapp-contact", "127.0.0.1:1234", http.Header{"X-Client-Ip": {"8.8.8.8", "1.1.1.1"}}, 400},
		{"GET", "/api/v1/whatsapp-contact", "192.0.2.1:1234", http.Header{"X-Client-Ip": {"8.8.8.8"}}, 403},
		{"GET", "/api/v1/whatsapp-contact?ip=8.8.8.8", "127.0.0.1:1234", http.Header{"X-Client-Ip": {"8.8.8.8"}}, 400},
		{"POST", "/api/v1/whatsapp-contact", "127.0.0.1:1234", http.Header{"X-Client-Ip": {"8.8.8.8"}}, 405},
		{"HEAD", "/api/v1/whatsapp-contact", "127.0.0.1:1234", http.Header{"X-Client-Ip": {"8.8.8.8"}}, 405},
		{"GET", "/api/v1/whatsapp-contact", "127.0.0.1:1234", http.Header{"X-Client-Ip": {"8.8.8.8"}}, 503},
	} {
		w := request(tc.method, tc.path, tc.remote, tc.headers)
		if w.Code != tc.want {
			t.Fatal(tc, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cacheable")
		}
	}
	// Add real metadata only in this disposable database, then verify the public
	// DTO and that status reads cannot consume a round-robin position.
	var mediaID int64
	if e = d.DB.Raw("INSERT INTO media(name,url,mime,media_type,size,available,storage_root_id,storage_key,uploaded_by) VALUES('QR','/uploads/0123456789abcdef0123456789abcdef/0123456789abcdef0123456789abcdef.png','image/png','image',1,true,'test','test.png',-1) RETURNING id").Scan(&mediaID).Error; e != nil {
		t.Fatal(e)
	}
	list, e := r.List(context.Background(), 1, 20, "")
	if e != nil {
		t.Fatal(e)
	}
	id, _ := strconv.ParseInt(list.DefaultGroupID, 10, 64)
	g, e := r.Get(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	_, e = r.Save(context.Background(), id, biz.Input{Name: g.Name, CountryCodes: g.CountryCodes, Enabled: true, ExpectedUpdatedAt: g.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), Accounts: []biz.AccountInput{{Name: "Public contact", PhoneNumber: "12025550101", QRCodeMediaID: strconv.FormatInt(mediaID, 10), Enabled: true}}})
	if e != nil {
		t.Fatal(e)
	}
	w := request("GET", "/api/v1/whatsapp-status", "192.0.2.1:1234", nil)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var count int64
	d.DB.Table("whatsapp_cursor").Select("allocation_count").Where("group_id=?", id).Scan(&count)
	if count != 0 {
		t.Fatal("status consumed")
	}
	w = request("GET", "/api/v1/whatsapp-contact", "127.0.0.1:1234", http.Header{"X-Client-Ip": {"8.8.8.8"}})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var out map[string]any
	if e = json.Unmarshal(w.Body.Bytes(), &out); e != nil {
		t.Fatal(e)
	}
	if len(out) != 4 || out["url"] != "https://wa.me/12025550101" || out["qrCodeUrl"] == "" {
		t.Fatal("public DTO", out)
	}
}
