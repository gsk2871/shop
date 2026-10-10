package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	repo "peptide-site/internal/data/admin"
	"peptide-site/internal/pkg/admintotp"
	"peptide-site/internal/testutil"
	"testing"
	"time"
)

func TestLoginScopesPermissionsAndRevocation(t *testing.T) {
	d, c := testutil.Postgres(t)
	r, e := repo.NewRepo(d, c)
	if e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { Reply(w, 200, map[string]bool{"ok": true}) }))
	if e != nil {
		t.Fatal(e)
	}
	request := func(method, path, token string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "/admin/api/v1/"+path, bytes.NewReader(b))
		req.Header.Set("X-Requested-With", "peptide-admin")
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "admin_session", Value: token})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	w := request("POST", "login", "", map[string]string{"username": "admin", "password": "test-initial-password"})
	if w.Code != 200 || len(w.Result().Cookies()) != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
	onboarding := w.Result().Cookies()[0]
	if !onboarding.HttpOnly || onboarding.SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe cookie")
	}
	if request("GET", "products", onboarding.Value, nil).Code != 403 {
		t.Fatal("onboarding business access")
	}
	if request("POST", "totp/setup", onboarding.Value, nil).Code != 409 {
		t.Fatal("initial password bypass")
	}
	w = request("POST", "password", onboarding.Value, map[string]string{"current": "test-initial-password", "password": "new-test-password-123"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if request("GET", "me", onboarding.Value, nil).Code != 401 {
		t.Fatal("password did not revoke")
	}
	secret, _ := admintotp.Secret()
	key, cipher, _ := r.Codec.Encrypt(secret)
	if e = d.DB.Exec("UPDATE admin_user SET totp_key_id=?,totp_cipher=? WHERE username='admin'", key, cipher).Error; e != nil {
		t.Fatal(e)
	}
	w = request("POST", "login", "", map[string]string{"username": "admin", "password": "new-test-password-123"})
	if w.Code != 200 || len(w.Result().Cookies()) != 0 {
		t.Fatal("password alone signed full cookie", w.Code, w.Body.String())
	}
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &ticket)
	code, _ := admintotp.Code(secret, time.Now().Unix()/30)
	w = request("POST", "login/totp", "", map[string]string{"ticket": ticket.Ticket, "code": code})
	if w.Code != 200 || len(w.Result().Cookies()) != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
	token := w.Result().Cookies()[0].Value
	if request("GET", "products", token, nil).Code != 200 {
		t.Fatal("full denied")
	}
	if request("PUT", "products", token, map[string]bool{}).Code != 403 {
		t.Fatal("collection PUT bypass")
	}
	if request("POST", "login/totp", "", map[string]string{"ticket": ticket.Ticket, "code": code}).Code == 200 {
		t.Fatal("ticket replay")
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("readonly-test-password"), bcrypt.MinCost)
	var userID int64
	if e = d.DB.Raw("INSERT INTO admin_user(username,display_name,password_hash,role_id,enabled,must_change_pwd) SELECT 'reader','Reader',?,id,true,false FROM admin_role WHERE code='viewer' RETURNING id", string(hash)).Scan(&userID).Error; e != nil {
		t.Fatal(e)
	}
	if userID == 0 {
		t.Fatal("viewer fixture missing")
	}
	readerToken := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if e = d.DB.Exec("INSERT INTO admin_session(token_hash,user_id,scope,auth_version,expires_at) VALUES(?,?,'full',1,now()+interval '1 hour')", repo.Hash(readerToken), userID).Error; e != nil {
		t.Fatal(e)
	}
	if request("GET", "products", readerToken, nil).Code != 200 {
		t.Fatal("reader list denied")
	}
	for _, entry := range []struct{ method, path string }{{"POST", "products"}, {"POST", "products/preview"}, {"PUT", "settings"}, {"POST", "media"}, {"PATCH", "banners/1/enabled"}} {
		if request(entry.method, entry.path, readerToken, nil).Code != 403 {
			t.Fatal("reader mutation allowed", entry)
		}
	}
	if e = d.DB.Exec("UPDATE admin_user SET enabled=false WHERE id=?", userID).Error; e != nil {
		t.Fatal(e)
	}
	if request("GET", "products", readerToken, nil).Code != 401 {
		t.Fatal("disable did not revoke")
	}
	req := httptest.NewRequest("POST", "/admin/api/v1/logout", nil)
	req.Header.Set("X-Requested-With", "peptide-admin")
	req.Header.Set("Origin", "https://attacker.invalid")
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: token})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("CSRF origin allowed")
	}
	if _, e = r.Identity(context.Background(), token); e != nil {
		t.Fatal("rejected CSRF changed session")
	}
}
func TestTrustedProxyConsumers(t *testing.T) {
	d, c := testutil.Postgres(t)
	r, e := repo.NewRepo(d, c)
	if e != nil {
		t.Fatal(e)
	}
	h, _ := NewHandler(r, http.NotFoundHandler())
	req := httptest.NewRequest("POST", "http://shop.test/admin/api/v1/logout", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Requested-With", "peptide-admin")
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 127.0.0.2")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("Origin", "https://shop.test")
	if h.ip(req) != "127.0.0.1" || h.csrf(req) {
		t.Fatal("untrusted forwarded values accepted")
	}
	r.Config.Auth.TrustedProxies = []string{"127.0.0.0/8"}
	if h.ip(req) != "203.0.113.5" || !h.csrf(req) {
		t.Fatal("trusted proxy consumer stale")
	}
}
