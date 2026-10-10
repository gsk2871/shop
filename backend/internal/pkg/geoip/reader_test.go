package geoip

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

func TestCountryDatabase(t *testing.T) {
	r, e := Open("")
	if e != nil || r.Available() {
		t.Fatal("empty path", e)
	}
	if code, e := r.Country(netip.MustParseAddr("8.8.8.8")); e != nil || code != "" {
		t.Fatal(code, e)
	}
	r.Close()
	bad := filepath.Join(t.TempDir(), "broken.mmdb")
	if e = os.WriteFile(bad, []byte("invalid"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Open(bad); e == nil {
		t.Fatal("invalid database accepted")
	}
	path := os.Getenv("TEST_GEOIP_PATH")
	if path == "" {
		return
	}
	r, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	for ip, want := range map[string]string{"8.8.8.8": "US", "2001:4860:4860::8888": "CA", "127.0.0.1": "", "192.168.1.1": ""} {
		got, e := r.Country(netip.MustParseAddr(ip))
		if e != nil || got != want {
			t.Fatal(ip, got, want, e)
		}
	}
}
