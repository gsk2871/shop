package admin

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	biz "peptide-site/internal/biz/admin"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	"peptide-site/internal/pkg/admintotp"
	"sync"
	"testing"
	"time"
)

func testRepo(t *testing.T) (*Repo, *data.Data, conf.Config) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_DSN")
	if base == "" {
		t.Skip("TEST_DATABASE_DSN required")
	}
	for _, name := range []string{"DATABASE_DSN", "HTTP_ADDRESS", "UPLOAD_DIR", "SEED_ADMIN_PASSWORD"} {
		t.Setenv(name, "")
	}
	t.Setenv("COOKIE_SECURE", "false")
	loader := conf.NewLoader()
	c, err := loader.Decode([]byte("data:\n  database:\n    dsn: " + base + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	c.Auth.TOTP.ActiveKeyID = "one"
	c.Auth.TOTP.Keys = map[string]string{"one": base64.StdEncoding.EncodeToString(make([]byte, 32))}
	c.Seed.SuperAdmin.Password = "test-initial-password"
	admin, cleanup, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("shop_auth_%d", time.Now().UnixNano())
	if err := admin.DB.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatal(err)
	}
	c.Data.Database.DSN = base + " dbname=" + name
	d, closeDB, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeDB(); _ = admin.DB.Exec("DROP DATABASE " + name + " WITH (FORCE)").Error; cleanup() })
	if err := data.Migrate(d); err != nil {
		t.Fatal(err)
	}
	if err := Seed(context.Background(), d, c.Seed); err != nil {
		t.Fatal(err)
	}
	r, err := NewRepo(d, c)
	if err != nil {
		t.Fatal(err)
	}
	return r, d, c
}
func TestSeedAndTwoPhaseLogin(t *testing.T) {
	r, d, c := testRepo(t)
	ctx := context.Background()
	now := time.Unix(1700000000, 0).UTC()
	r.Now = func() time.Time { return now }
	m := Meta{RequestID: "test-authentication"}
	var initial biz.User
	if err := d.DB.Table("admin_user").Where("username=?", "admin").Take(&initial).Error; err != nil {
		t.Fatal(err)
	}
	c.Seed.SuperAdmin.Password = "different-seed-password"
	if err := Seed(ctx, d, c.Seed); err != nil {
		t.Fatal(err)
	}
	var repeated biz.User
	_ = d.DB.Table("admin_user").Where("id=?", initial.ID).Take(&repeated).Error
	if repeated.PasswordHash != initial.PasswordHash {
		t.Fatal("seed reset existing credentials")
	}
	login, err := r.Login(ctx, "admin", "test-initial-password", m)
	if err != nil || login.Scope != "onboarding" {
		t.Fatal("initial onboarding missing", err)
	}
	identity, err := r.Identity(ctx, login.Token)
	if err != nil {
		t.Fatal(err)
	}
	m.ActorID = identity.User.ID
	if _, _, err := r.Setup(ctx, identity, m); err == nil {
		t.Fatal("binding before initial password accepted")
	}
	if err := r.Password(ctx, identity, "test-initial-password", "test-new-password", m); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Identity(ctx, login.Token); err == nil {
		t.Fatal("old session retained after password change")
	}
	login, err = r.Login(ctx, "admin", "test-new-password", m)
	if err != nil {
		t.Fatal(err)
	}
	identity, err = r.Identity(ctx, login.Token)
	if err != nil {
		t.Fatal(err)
	}
	secret, _, err := r.Setup(ctx, identity, m)
	if err != nil {
		t.Fatal(err)
	}
	code, _ := admintotp.Code(secret, now.Unix()/30)
	if err := r.Activate(ctx, identity, code, m); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Identity(ctx, login.Token); err == nil {
		t.Fatal("onboarding session survived activation")
	}
	ticket, err := r.Login(ctx, "admin", "test-new-password", m)
	if err != nil || ticket.Step != "totp" || ticket.Token != "" {
		t.Fatal("password step issued formal cookie", err)
	}
	if _, err := r.LoginTOTP(ctx, ticket.Ticket, code, m); err == nil {
		t.Fatal("activation code replay accepted")
	}
	var attempts int
	_ = d.DB.Table("admin_mfa_ticket").Select("attempts").Where("token_hash=?", Hash(ticket.Ticket)).Scan(&attempts).Error
	if attempts != 1 {
		t.Fatal("failed attempt rolled back")
	}
	now = now.Add(30 * time.Second)
	code, _ = admintotp.Code(secret, now.Unix()/30)
	var wg sync.WaitGroup
	success := make(chan LoginResult, 2)
	for j := 0; j < 2; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := r.LoginTOTP(ctx, ticket.Ticket, code, m)
			if err == nil {
				success <- result
			}
		}()
	}
	wg.Wait()
	close(success)
	var full LoginResult
	count := 0
	for result := range success {
		count++
		full = result
	}
	if count != 1 {
		t.Fatalf("ticket consumed %d times", count)
	}
	identity, err = r.Identity(ctx, full.Token)
	if err != nil || identity.Scope != "full" {
		t.Fatal("formal session missing", err)
	}
	// Removing a referenced old key is rejected; adding a new active key preserves decryption.
	old := r.Codec
	newCodec, _ := admintotp.New("two", map[string]string{"one": c.Auth.TOTP.Keys["one"], "two": base64.StdEncoding.EncodeToString(bytesRepeat(1, 32))})
	r.Codec = newCodec
	if err := r.CheckKeys(ctx); err != nil {
		t.Fatal("rotation rejected old ciphertext")
	}
	missing, _ := admintotp.New("two", map[string]string{"two": base64.StdEncoding.EncodeToString(bytesRepeat(1, 32))})
	r.Codec = missing
	if err := r.CheckKeys(ctx); err == nil {
		t.Fatal("removed referenced key accepted")
	}
	r.Codec = old
	if err := r.UserAction(ctx, identity, identity.User.ID, "status", "", "", false, m); err == nil {
		t.Fatal("last super disabled")
	}
	second, err := r.WriteUser(ctx, identity, 0, UserWrite{Username: "second", DisplayName: "Second", RoleID: identity.Role.ID, Enabled: true, Password: "second-initial-password"}, m)
	if err != nil {
		t.Fatal(err)
	}
	// Concurrent removals are serialized across target rows.
	results := make(chan error, 2)
	for _, id := range []int64{identity.User.ID, second.ID} {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			results <- r.UserAction(ctx, identity, id, "status", "", "", false, m)
		}(id)
	}
	wg.Wait()
	close(results)
	failed := 0
	for err := range results {
		if err != nil {
			failed++
		}
	}
	if failed != 1 {
		t.Fatal("concurrent last-super guard failed")
	}
	var active int64
	_ = d.DB.Table("admin_user").Where("enabled AND role_id=?", identity.Role.ID).Count(&active).Error
	if active != 1 {
		t.Fatal("zero effective super administrators")
	}
	for j := 0; j < r.Config.Auth.LoginLimit; j++ {
		if err := r.Rate(ctx, "test-shared-limit"); err != nil {
			t.Fatal(err)
		}
	}
	otherRepo, err := NewRepo(d, c)
	if err != nil {
		t.Fatal(err)
	}
	otherRepo.Now = r.Now
	if err := otherRepo.Rate(ctx, "test-shared-limit"); err == nil {
		t.Fatal("rate limiter not shared")
	}
}
func bytesRepeat(value byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = value
	}
	return b
}

func TestTOTPDisabledLoginBindingAndReenable(t *testing.T) {
	r, d, c := testRepo(t)
	ctx := context.Background()
	m := Meta{RequestID: "test-totp-policy"}
	r.Config.Auth.TOTP.Enabled = false
	initial, err := r.Login(ctx, "admin", c.Seed.SuperAdmin.Password, m)
	if err != nil || initial.Step != "session" || initial.Scope != "password_change" || len(initial.Token) != 64 {
		t.Fatalf("initial login: %+v %v", initial, err)
	}
	i, err := r.Identity(ctx, initial.Token)
	if err != nil || i.Scope != "password_change" || i.TOTPEnabled {
		t.Fatalf("initial identity: %+v %v", i, err)
	}
	if err = r.Password(ctx, i, c.Seed.SuperAdmin.Password, "test-new-password", m); err != nil {
		t.Fatal(err)
	}
	login, err := r.Login(ctx, "admin", "test-new-password", m)
	if err != nil || login.Step != "session" || login.Scope != "full" {
		t.Fatalf("password login: %+v %v", login, err)
	}
	i, err = r.Identity(ctx, login.Token)
	if err != nil || i.Scope != "full" || i.User.TOTPBound {
		t.Fatalf("unbound identity: %+v %v", i, err)
	}
	if err = r.Reauthenticate(ctx, i, "wrong-password", "", m); err == nil {
		t.Fatal("wrong password accepted")
	}
	if err = r.Reauthenticate(ctx, i, "test-new-password", "", m); err != nil {
		t.Fatal("password reauthentication", err)
	}
	// Reauthentication must not let a password-only session survive enabling MFA.
	r.Config.Auth.TOTP.Enabled = true
	if _, err = r.Identity(ctx, login.Token); err == nil {
		t.Fatal("password-only session survived enabling MFA")
	}
	stripped := login.Token[len(passwordOnlyPrefix):]
	if _, err = r.Identity(ctx, stripped); err == nil {
		t.Fatal("removing token marker bypassed policy")
	}
	onboarding, err := r.Login(ctx, "admin", "test-new-password", m)
	if err != nil || onboarding.Scope != "onboarding" {
		t.Fatalf("enabled unbound login: %+v %v", onboarding, err)
	}
	// Disabling also frees existing onboarding sessions without rebinding.
	r.Config.Auth.TOTP.Enabled = false
	i, err = r.Identity(ctx, onboarding.Token)
	if err != nil || i.Scope != "full" {
		t.Fatalf("disabled onboarding: %+v %v", i, err)
	}
	secret, _, err := r.Setup(ctx, i, m)
	if err != nil {
		t.Fatal("optional binding disabled", err)
	}
	code, _ := admintotp.Code(secret, r.Now().Unix()/30)
	if err = r.Activate(ctx, i, code, m); err != nil {
		t.Fatal("optional activation disabled", err)
	}
	login, err = r.Login(ctx, "admin", "test-new-password", m)
	if err != nil || login.Step != "session" || login.Scope != "full" {
		t.Fatalf("disabled bound login: %+v %v", login, err)
	}
	i, err = r.Identity(ctx, login.Token)
	if err != nil || !i.User.TOTPBound {
		t.Fatalf("binding lost: %+v %v", i, err)
	}
	r.Config.Auth.TOTP.Enabled = true
	if _, err = r.Identity(ctx, login.Token); err == nil {
		t.Fatal("bound password-only session survived enabling MFA")
	}
	ticket, err := r.Login(ctx, "admin", "test-new-password", m)
	if err != nil || ticket.Step != "totp" || ticket.Token != "" {
		t.Fatalf("enabled bound login: %+v %v", ticket, err)
	}
	if err = r.Reauthenticate(ctx, i, "test-new-password", "", m); err == nil {
		t.Fatal("enabled reauthentication accepted missing code")
	}
	var count int64
	if err = d.DB.Table("admin_mfa_ticket").Where("token_hash=?", Hash(ticket.Ticket)).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("MFA ticket missing", err)
	}
}
