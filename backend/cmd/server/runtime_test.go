package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	adminData "peptide-site/internal/data/admin"
	"peptide-site/internal/pkg/configreload"
	"testing"
	"time"
)

func freeAddress(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

// Only explicitly supplied test infrastructure and uniquely named disposable databases are used.
func TestRuntimePostgres(t *testing.T) {
	base := os.Getenv("TEST_DATABASE_DSN")
	if base == "" {
		t.Skip("TEST_DATABASE_DSN required")
	}
	ctx := context.Background()
	for _, name := range []string{"DATABASE_DSN", "HTTP_ADDRESS", "UPLOAD_DIR", "SEED_ADMIN_PASSWORD"} {
		t.Setenv(name, "")
	}
	t.Setenv("COOKIE_SECURE", "false")
	loader := conf.NewLoader()
	c, err := loader.Decode([]byte("data:\n  database:\n    dsn: " + base + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	admin, closeAdmin, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAdmin()
	name := fmt.Sprintf("shop_runtime_%d", time.Now().UnixNano())
	other := name + "_other"
	for _, db := range []string{name, other} {
		if err := admin.DB.Exec("CREATE DATABASE " + db).Error; err != nil {
			t.Fatal(err)
		}
		defer func(db string) {
			if err := admin.DB.Exec("DROP DATABASE " + db + " WITH (FORCE)").Error; err != nil {
				t.Error(err)
			}
		}(db)
	}
	dsn := base + " dbname=" + name
	otherDSN := base + " dbname=" + other
	dir := t.TempDir()
	rootOne := filepath.Join(dir, "one")
	rootTwo := filepath.Join(dir, "two")
	configPath := filepath.Join(dir, "runtime.yaml")
	addr := freeAddress(t)
	body := func(app, db, root, address string) string {
		return fmt.Sprintf("app:\n  name: %s\nreload:\n  interval: 1h\nserver:\n  http:\n    addr: %s\ndata:\n  database:\n    dsn: %s\n  storage:\n    upload_dir: %s\nauth:\n  cookie_secure: false\n  totp:\n    issuer: Test\n    active_key_id: test\n    keys:\n      test: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n", app, address, db, root)
	}
	write := func(b string) {
		t.Helper()
		tmp := configPath + ".new"
		if err := os.WriteFile(tmp, []byte(b), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(tmp, configPath); err != nil {
			t.Fatal(err)
		}
	}
	write(body("one", dsn, rootOne, addr))
	c, err = loader.Load(configPath, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, db := range []string{dsn, otherDSN} {
		cc := c
		cc.Data.Database.DSN = db
		d, cleanup, err := data.NewData(cc)
		if err != nil {
			t.Fatal(err)
		}
		err = data.Migrate(d)
		if err == nil {
			seedConfig := adminData.SeedOptions{Username: "admin", Password: "test-initial-password"}
			err = adminData.Seed(ctx, d, seedConfig)
		}
		cleanup()
		if err != nil {
			t.Fatal(err)
		}
	}
	factory := func(ctx context.Context, c conf.Config) (*configreload.Resource, error) {
		r, cleanup, err := initializeServer(c)
		if err == nil {
			resourceClose := r.Close
			r.Close = func() {
				if resourceClose != nil {
					resourceClose()
				}
				cleanup()
			}
		}
		return r, err
	}
	manager, err := configreload.New(ctx, configPath, c, loader, factory)
	if err != nil {
		t.Fatal(err)
	}
	runtime := configreload.NewHTTP(manager)
	if err := runtime.Start(ctx); err != nil {
		manager.Close()
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runtime.Stop(ctx)
	}()
	client := &http.Client{Timeout: 2 * time.Second}
	get := func(address, path string) (int, string) {
		t.Helper()
		res, err := client.Get("http://" + address + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, string(b)
	}
	code, _ := get(addr, "/healthz")
	if code != 200 {
		t.Fatal("initial listener unavailable")
	}
	d, cleanup, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	var roots []struct{ ID, Path string }
	if err := d.DB.Table("storage_root").Find(&roots).Error; err != nil {
		t.Fatal(err)
	}
	cleanup()
	if len(roots) != 1 {
		t.Fatal("root not registered")
	}
	if err := os.WriteFile(filepath.Join(rootOne, "sample.txt"), []byte("0123456789"), 0640); err != nil {
		t.Fatal(err)
	}
	fixture, closeFixture, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.DB.Exec("INSERT INTO media(name,url,mime,size,media_type,storage_root_id,storage_key) VALUES('sample','/uploads/sample.txt','text/plain',10,'document',?,'sample.txt')", roots[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	closeFixture()
	mediaPath := "/uploads/" + roots[0].ID + "/sample.txt"
	write(body("two", dsn, rootTwo, addr))
	if err := runtime.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	code, b := get(addr, mediaPath)
	if code != 200 || b != "0123456789" {
		t.Fatal("old root unreadable")
	}
	req, _ := http.NewRequest("GET", "http://"+addr+mediaPath, nil)
	req.Header.Set("Range", "bytes=2-5")
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	rangeBytes, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != 206 || string(rangeBytes) != "2345" {
		t.Fatal("Range lost")
	}
	before := manager.Status().Version
	write(body("wrong", otherDSN, rootTwo, addr))
	if err := runtime.Reload(ctx); err == nil || manager.Status().Version != before {
		t.Fatal("different database identity accepted")
	}
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	write(body("wrong", dsn, rootTwo, occupied.Addr().String()))
	if err := runtime.Reload(ctx); err == nil || manager.Config().App.Name != "two" {
		t.Fatal("occupied address partially applied")
	}
	_ = occupied.Close()
	newAddress := freeAddress(t)
	write(body("three", dsn, rootTwo, newAddress))
	if err := runtime.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	code, _ = get(newAddress, "/readyz")
	if code != 200 {
		t.Fatal("new listener unavailable")
	}
	write(body("four", dsn+" application_name=reload-test", rootTwo, newAddress))
	if err := runtime.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	code, _ = get(newAddress, mediaPath)
	if code != 200 {
		t.Fatal("media lost after DSN switch")
	}
	for i := 0; i < 5; i++ {
		write(body(fmt.Sprintf("loop-%d", i), dsn, rootTwo, newAddress))
		if err := runtime.Reload(ctx); err != nil {
			t.Fatal(err)
		}
	}
}
