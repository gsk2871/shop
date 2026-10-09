package cms

import (
	"bytes"
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	biz "peptide-site/internal/biz/cms"
	"peptide-site/internal/conf"
	"strings"
	"testing"
	"time"
)

type fakeRepo struct {
	biz.Repository
	authenticated bool
	sessionErr    error
	settings      biz.Settings
	saved         bool
	media         *biz.Media
	saveMediaErr  error
	account       biz.Admin
	sessionHash   string
	actions       []string
}

func (f *fakeRepo) Session(context.Context, string) (bool, error) {
	return f.authenticated, f.sessionErr
}
func (f *fakeRepo) Settings(context.Context) (biz.Settings, error) { return f.settings, nil }
func (f *fakeRepo) SaveSettings(_ context.Context, s biz.Settings) error {
	f.settings = s
	f.saved = true
	return nil
}
func (f *fakeRepo) SaveMedia(_ context.Context, m *biz.Media) error {
	f.media = m
	return f.saveMediaErr
}
func (f *fakeRepo) Admin(context.Context) (biz.Admin, error) { return f.account, nil }
func (f *fakeRepo) CreateSession(_ context.Context, hash string, _ time.Time) error {
	f.sessionHash = hash
	return nil
}
func (f *fakeRepo) Audit(_ context.Context, action string) error {
	f.actions = append(f.actions, action)
	return nil
}
func request(h *Handler, method, path, body string, cookie, header bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie {
		r.AddCookie(&http.Cookie{Name: "admin_session", Value: strings.Repeat("a", 64)})
	}
	if header {
		r.Header.Set("X-Requested-With", "peptide-admin")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name, method            string
		cookie, header, session bool
		err                     error
		code                    int
	}{{"anonymous", "GET", false, false, false, nil, 401}, {"expired", "GET", true, false, false, nil, 401}, {"authenticated", "GET", true, false, true, nil, 200}, {"csrf", "PUT", true, false, true, nil, 403}, {"store failure", "GET", true, false, false, errors.New("secret"), 503}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{authenticated: tc.session, sessionErr: tc.err, settings: biz.DefaultSettings()}
			w := request(NewHandler(repo, conf.Config{}), tc.method, "/admin/api/v1/settings", "", tc.cookie, tc.header)
			if w.Code != tc.code || strings.Contains(w.Body.String(), "secret") {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
		})
	}
}
func TestSaveSettings(t *testing.T) {
	repo := &fakeRepo{authenticated: true}
	h := NewHandler(repo, conf.Config{})
	w := request(h, "PUT", "/admin/api/v1/settings", `{"brand":"Lab","primaryColor":"#123456","backgroundColor":"#ffffff","banners":[]}`, true, true)
	if w.Code != 200 || !repo.saved || len(repo.actions) != 1 {
		t.Fatalf("save failed: %s", w.Body.String())
	}
	repo.saved = false
	w = request(h, "PUT", "/admin/api/v1/settings", `{"brand":"Lab","primaryColor":"red","backgroundColor":"#ffffff"}`, true, true)
	if w.Code != 400 || repo.saved {
		t.Fatal("invalid config persisted")
	}
}
func TestLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password-123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{account: biz.Admin{Username: "admin", PasswordHash: string(hash)}}
	h := NewHandler(repo, conf.Config{Auth: conf.Auth{CookieSecure: true}})
	w := request(h, "POST", "/admin/api/v1/login", `{"username":"admin","password":"wrong"}`, false, true)
	if w.Code != 401 {
		t.Fatal("wrong password accepted")
	}
	w = request(h, "POST", "/admin/api/v1/login", `{"username":"admin","password":"test-password-123"}`, false, true)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe cookie")
	}
	if repo.sessionHash == cookies[0].Value || len(repo.sessionHash) != 64 {
		t.Fatal("raw token persisted")
	}
}
func TestRateLimit(t *testing.T) {
	h := NewHandler(&fakeRepo{}, conf.Config{})
	for i := 0; i < 5; i++ {
		if !h.allowed("127.0.0.1:8000") {
			t.Fatal("early rejection")
		}
	}
	if h.allowed("127.0.0.1:9000") {
		t.Fatal("port bypass")
	}
}
func TestUpload(t *testing.T) {
	for _, tc := range []struct {
		name, body  string
		dbErr       error
		code, files int
	}{{"pdf", "%PDF-1.4\nexample", nil, 201, 1}, {"html", "<html><script>x</script></html>", nil, 400, 0}, {"rollback", "%PDF-1.4\nexample", errors.New("db down"), 500, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			repo := &fakeRepo{authenticated: true, saveMediaErr: tc.dbErr}
			h := NewHandler(repo, conf.Config{Data: conf.Data{Storage: conf.Storage{UploadDir: dir}}})
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "../../report.pdf")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write([]byte(tc.body)); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/admin/api/v1/media", &body)
			r.Header.Set("Content-Type", writer.FormDataContentType())
			r.Header.Set("X-Requested-With", "peptide-admin")
			r.AddCookie(&http.Cookie{Name: "admin_session", Value: strings.Repeat("a", 64)})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.code {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
			files, err := os.ReadDir(dir)
			if err != nil || len(files) != tc.files {
				t.Fatal("file rollback failed")
			}
			if repo.media != nil && repo.media.Name != "report.pdf" {
				t.Fatal("unsafe filename")
			}
		})
	}
}
func TestTraversal(t *testing.T) {
	h := NewHandler(&fakeRepo{}, conf.Config{Data: conf.Data{Storage: conf.Storage{UploadDir: t.TempDir()}}})
	w := httptest.NewRecorder()
	h.PublicFiles().ServeHTTP(w, httptest.NewRequest("GET", "/uploads/../secret", nil))
	if w.Code != 404 {
		t.Fatal("traversal allowed")
	}
}
