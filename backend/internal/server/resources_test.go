package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"gopkg.in/yaml.v3"
	"net/http/httptest"
	"os"
	"path/filepath"
	"peptide-site/internal/biz"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	admin "peptide-site/internal/data/admin"
	catalog "peptide-site/internal/data/catalog"
	"peptide-site/internal/pkg/admintotp"
	"peptide-site/internal/pkg/configreload"
	"peptide-site/internal/service"
	cms "peptide-site/internal/service/cms"
	"peptide-site/internal/testutil"
	"testing"
	"time"
)

func TestReloadResourceConsumers(t *testing.T) {
	d, c := testutil.Postgres(t)
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime.yaml")
	write := func(cfg conf.Config) {
		b, e := yaml.Marshal(cfg)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	write(c)
	_ = os.Unsetenv("COOKIE_SECURE")
	t.Cleanup(func() { _ = os.Setenv("COOKIE_SECURE", "false") })
	loader := conf.NewLoader()
	c, e := loader.Load(path, false)
	if e != nil {
		t.Fatal(e)
	}
	var active *data.Data
	factory := func(ctx context.Context, c conf.Config) (*configreload.Resource, error) {
		db, closeDB, e := data.NewData(c)
		if e != nil {
			return nil, e
		}
		r, e := NewResources(c, db, service.NewSystem(biz.NewSystem(db)), cms.NewHandler(c))
		if e != nil {
			closeDB()
			return nil, e
		}
		active = db
		closeResource := r.Close
		r.Close = func() { closeResource(); closeDB() }
		return r, nil
	}
	m, e := configreload.New(ctx, path, c, loader, factory)
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	prepare := func(conf.Config) (func(), func(), error) { return nil, nil, nil }
	next := c
	next.App.Name = "changed"
	next.Log.Level = "debug"
	next.Reload.Interval = 100 * time.Millisecond
	next.Server.HTTP.Timeout = 2 * time.Second
	next.Server.HTTP.ReadHeaderTimeout = time.Second
	next.Server.HTTP.ReadTimeout = 3 * time.Second
	next.Server.HTTP.WriteTimeout = 4 * time.Second
	next.Server.HTTP.IdleTimeout = 5 * time.Second
	next.Data.Database.DSN += " application_name=reload-consumer"
	next.Data.Database.MaxOpenConns = 3
	next.Data.Database.MaxIdleConns = 1
	next.Data.Database.ConnMaxLifetime = 150 * time.Millisecond
	next.Data.Storage.UploadDir = t.TempDir()
	next.Data.Storage.MaxImageBytes = 1024
	next.Data.Storage.MaxVideoBytes = 4096
	next.Data.Storage.MaxDocumentBytes = 2048
	next.Data.Storage.AllowedTypes = []string{"image/png"}
	next.Data.Storage.CleanupInterval = 100 * time.Millisecond
	next.Auth.CookieSecure = true
	next.Auth.SessionTTL = time.Hour
	next.Auth.OnboardingTTL = 9 * time.Minute
	next.Auth.MFATicketTTL = 2 * time.Minute
	next.Auth.MFAMaxAttempts = 2
	next.Auth.LoginLimit = 2
	next.Auth.TrustedProxies = []string{"127.0.0.0/8"}
	next.Auth.TOTP.Issuer = "Reloaded Admin"
	next.Auth.TOTP.ActiveKeyID = "rotated"
	next.Auth.TOTP.Keys = map[string]string{"test": c.Auth.TOTP.Keys["test"], "rotated": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))}
	next.Seed.SuperAdmin.Username = "future_admin"
	next.Seed.SuperAdmin.Password = "future-seed-password"
	next.Product.DefaultCurrency = "EUR"
	next.Product.AllowedCurrencies = []string{"USD", "EUR"}
	write(next)
	if e = m.Reload(ctx, prepare); e == nil || m.Status().Version != 1 {
		t.Fatal("missing currency partially applied", e)
	}
	if e = d.DB.Exec("INSERT INTO currency(code,minor_unit,symbol) VALUES('EUR',2,'€')").Error; e != nil {
		t.Fatal(e)
	}
	if e = m.Reload(ctx, prepare); e != nil {
		t.Fatal(e)
	}
	pool, _ := active.DB.DB()
	if pool.Stats().MaxOpenConnections != 3 {
		t.Fatal("pool consumer stale")
	}
	var dbApp string
	if e = active.DB.Raw("SHOW application_name").Scan(&dbApp).Error; e != nil || dbApp != "reload-consumer" {
		t.Fatal("DSN consumer stale", e)
	}
	time.Sleep(200 * time.Millisecond)
	if e = pool.PingContext(ctx); e != nil || pool.Stats().MaxLifetimeClosed == 0 {
		t.Fatal("connection lifetime stale", e, pool.Stats())
	}
	authRepo, e := admin.NewRepo(active, m.Config())
	if e != nil {
		t.Fatal(e)
	}
	result, e := authRepo.Login(ctx, "admin", c.Seed.SuperAdmin.Password, admin.Meta{})
	if e != nil || result.ExpiresIn != int(next.Auth.OnboardingTTL.Seconds()) {
		t.Fatal("onboarding TTL or seed reset", e, result)
	}
	identity, e := authRepo.Identity(ctx, result.Token)
	if e != nil {
		t.Fatal(e)
	}
	if e = active.DB.Exec("UPDATE admin_user SET must_change_pwd=false WHERE id=1").Error; e != nil {
		t.Fatal(e)
	}
	_, uri, e := authRepo.Setup(ctx, identity, admin.Meta{})
	if e != nil || !bytes.Contains([]byte(uri), []byte("Reloaded")) {
		t.Fatal("issuer consumer stale", e, uri)
	}
	var keyID string
	if e = active.DB.Table("admin_totp_pending").Select("key_id").Where("user_id=1").Scan(&keyID).Error; e != nil || keyID != "rotated" {
		t.Fatal("key consumer stale", e, keyID)
	}
	currencyRows, e := catalog.NewRepo(active, m.Config()).Currencies(ctx)
	if e != nil || len(currencyRows) != 2 {
		t.Fatal("currency consumer stale", e)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/api/v1/login", bytes.NewBufferString(`{"username":"admin","password":"test-initial-password"}`))
	req.Header.Set("X-Requested-With", "peptide-admin")
	m.ServeHTTP(w, req)
	if w.Code != 200 || !w.Result().Cookies()[0].Secure || w.Result().Cookies()[0].MaxAge != 540 {
		t.Fatal("cookie consumer stale", w.Code, w.Body.String())
	}
	token := w.Result().Cookies()[0].Value
	req = httptest.NewRequest("GET", "/admin/api/v1/media/upload-policy", nil)
	req.AddCookie(w.Result().Cookies()[0])
	w = httptest.NewRecorder()
	m.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("limited policy access", token)
	}
	if e = active.DB.Exec("UPDATE admin_session SET scope='full' WHERE token_hash=?", admin.Hash(token)).Error; e != nil {
		t.Fatal(e)
	}
	w = httptest.NewRecorder()
	m.ServeHTTP(w, req)
	var policy struct {
		MaxImageBytes int64    `json:"maxImageBytes"`
		AllowedTypes  []string `json:"allowedTypes"`
		Version       uint64   `json:"version"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &policy)
	if w.Code != 200 || policy.MaxImageBytes != 1024 || len(policy.AllowedTypes) != 1 || policy.Version != 2 {
		t.Fatal("policy consumer stale", w.Code, w.Body.String())
	}
	if e = authRepo.Rate(ctx, "consumer"); e != nil {
		t.Fatal(e)
	}
	if e = authRepo.Rate(ctx, "consumer"); e != nil {
		t.Fatal(e)
	}
	if e = authRepo.Rate(ctx, "consumer"); e == nil {
		t.Fatal("limiter stale")
	}
	secret, _ := admintotp.Secret()
	kid, encrypted, _ := authRepo.Codec.Encrypt(secret)
	if e = active.DB.Exec("UPDATE admin_user SET totp_key_id=?,totp_cipher=? WHERE id=1", kid, encrypted).Error; e != nil {
		t.Fatal(e)
	}
	login, e := authRepo.Login(ctx, "admin", c.Seed.SuperAdmin.Password, admin.Meta{})
	if e != nil || login.ExpiresIn != 120 {
		t.Fatal("MFA ticket TTL stale", e, login)
	}
	var expiry time.Time
	if e = active.DB.Table("admin_mfa_ticket").Select("expires_at").Where("token_hash=?", admin.Hash(login.Ticket)).Scan(&expiry).Error; e != nil || time.Until(expiry) > 121*time.Second {
		t.Fatal("ticket expiry stale", e)
	}
	_, _ = authRepo.LoginTOTP(ctx, login.Ticket, "not-a-code", admin.Meta{})
	_, _ = authRepo.LoginTOTP(ctx, login.Ticket, "not-a-code", admin.Meta{})
	code, _ := admintotp.Code(secret, time.Now().Unix()/30)
	if _, e = authRepo.LoginTOTP(ctx, login.Ticket, code, admin.Meta{}); e == nil {
		t.Fatal("MFA attempts stale")
	}
	login, e = authRepo.Login(ctx, "admin", c.Seed.SuperAdmin.Password, admin.Meta{})
	if e != nil {
		t.Fatal(e)
	}
	full, e := authRepo.LoginTOTP(ctx, login.Ticket, code, admin.Meta{})
	if e != nil || full.ExpiresIn != 3600 {
		t.Fatal("session TTL stale", e, full)
	}
	if e = active.DB.Exec("UPDATE admin_mfa_ticket SET expires_at=now()-interval '1 hour'").Error; e != nil {
		t.Fatal(e)
	}
	if e = active.DB.Exec("INSERT INTO admin_mfa_ticket(token_hash,user_id,auth_version,expires_at) VALUES('cleanup-test',1,1,now()-interval '1 hour')").Error; e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(time.Second)
	for {
		var count int64
		_ = active.DB.Table("admin_mfa_ticket").Where("token_hash='cleanup-test'").Count(&count).Error
		if count == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cleanup scheduler stale")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if e = admin.Seed(ctx, active, next.Seed); e != nil {
		t.Fatal(e)
	}
	future, e := authRepo.Login(ctx, "future_admin", next.Seed.SuperAdmin.Password, admin.Meta{})
	if e != nil || future.Scope != "onboarding" {
		t.Fatal("explicit seed consumer stale", e, future)
	}
	if m.Config().Product.DefaultCurrency != "EUR" || m.Config().Seed.SuperAdmin.Username != "future_admin" {
		t.Fatal("generation fields stale")
	}
}
