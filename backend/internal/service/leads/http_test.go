package leads

import (
	"context"
	"net/http/httptest"
	biz "peptide-site/internal/biz/leads"
	"peptide-site/internal/conf"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	calls int
	raw   string
}

func (s *fakeStore) Config(context.Context) (biz.Config, error)                     { return biz.Default(), nil }
func (s *fakeStore) SaveConfig(_ context.Context, c biz.Config) (biz.Config, error) { return c, nil }
func (s *fakeStore) Submit(_ context.Context, raw string) error                     { s.calls++; s.raw = raw; return nil }
func (s *fakeStore) List(context.Context, biz.Filter) ([]biz.Lead, int64, error)    { return nil, 0, nil }
func (s *fakeStore) Update(context.Context, int64, biz.Update) error                { return nil }
func TestPublicBoundary(t *testing.T) {
	store := &fakeStore{}
	c := conf.Config{}
	c.Auth.TrustedProxies = []string{"127.0.0.1/32"}
	h := NewHandler(store, c)
	h.limit = &limiter{clients: map[string]window{}}
	request := func(method, path, body, peer, ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.RemoteAddr = peer
		if ip != "" {
			req.Header.Set("X-Client-IP", ip)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	for _, tt := range []struct {
		path, body, peer, ip string
		status               int
	}{
		{"/api/v1/leads", `{"contact":"a@example.com"}`, "192.0.2.1:123", "203.0.113.1", 403},
		{"/api/v1/leads", `{"contact":"a@example.com"}`, "127.0.0.1:123", "", 400},
		{"/api/v1/leads", `{"contact":"a@example.com","status":"completed"}`, "127.0.0.1:123", "203.0.113.1", 400},
		{"/api/v1/leads", `{"contact":"a@example.com"} {}`, "127.0.0.1:123", "203.0.113.1", 400},
		{"/api/v1/leads", `{"contact":"` + strings.Repeat("x", 17000) + `"}`, "127.0.0.1:123", "203.0.113.1", 400},
	} {
		w := request("POST", tt.path, tt.body, tt.peer, tt.ip)
		if w.Code != tt.status {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if store.calls != 0 {
		t.Fatal("invalid request reached storage")
	}
	w := request("POST", "/api/v1/leads", `{"contact":"a@example.com"}`, "127.0.0.1:123", "203.0.113.2")
	if w.Code != 200 || strings.Contains(w.Body.String(), "contact") || store.calls != 1 {
		t.Fatal(w.Code, w.Body.String(), store.calls)
	}
	for i := 0; i < 4; i++ {
		request("POST", "/api/v1/leads", `{"contact":"a@example.com"}`, "127.0.0.1:123", "203.0.113.2")
	}
	w = request("POST", "/api/v1/leads", `{"contact":"a@example.com"}`, "127.0.0.1:123", "203.0.113.2")
	if w.Code != 429 || store.calls != 5 || w.Header().Get("Retry-After") == "" {
		t.Fatal("limit not enforced", w.Code, store.calls)
	}
	w = request("GET", "/api/v1/lead-popup", "", "127.0.0.1:123", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "version") {
		t.Fatal("public config", w.Code, w.Body.String())
	}
	for _, method := range []string{"GET", "PATCH", "DELETE"} {
		if request(method, "/api/v1/leads", "", "127.0.0.1:123", "").Code != 404 {
			t.Fatal("private lead API exposed", method)
		}
	}
}
func TestLimiterExpiryAndBound(t *testing.T) {
	l := &limiter{clients: map[string]window{}}
	now := time.Now()
	for i := 0; i < 5; i++ {
		if !l.allow("one", now) {
			t.Fatal("early limit")
		}
	}
	if l.allow("one", now.Add(time.Minute)) {
		t.Fatal("client window reset early")
	}
	if !l.allow("one", now.Add(10*time.Minute)) {
		t.Fatal("client window did not expire")
	}
	l.global = window{start: now, count: 60}
	if l.allow("two", now) {
		t.Fatal("global limit bypass")
	}
}
func TestFilters(t *testing.T) {
	for _, q := range []string{"page=0", "pageSize=101", "start=yesterday", "start=2026-10-27T00:00:00Z&end=2026-10-26T00:00:00Z"} {
		if _, err := filter(httptest.NewRequest("GET", "/admin/api/v1/leads?"+q, nil)); err == nil {
			t.Fatal("invalid filter", q)
		}
	}
	f, err := filter(httptest.NewRequest("GET", "/admin/api/v1/leads?contact=example&page=2&start=2026-10-26T00:00:00Z", nil))
	if err != nil || f.Page != 2 || f.Size != 20 || f.Contact != "example" || f.Start == nil {
		t.Fatal(f, err)
	}
}
