package configreload

import (
	"bufio"
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"peptide-site/internal/conf"
	"testing"
	"time"
)

func TestLiveTimeoutAndWatcherConsumers(t *testing.T) {
	for _, name := range []string{"DATABASE_DSN", "HTTP_ADDRESS", "UPLOAD_DIR", "SEED_ADMIN_PASSWORD"} {
		t.Setenv(name, "")
	}
	t.Setenv("COOKIE_SECURE", "false")
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	loader := conf.NewLoader()
	c, e := loader.Decode([]byte("data:\n  database:\n    dsn: test\n"))
	if e != nil {
		t.Fatal(e)
	}
	c.Server.HTTP.Addr = addr
	c.Reload.Interval = 100 * time.Millisecond
	c.Server.HTTP.Timeout = 300 * time.Millisecond
	c.Server.HTTP.ReadHeaderTimeout = 80 * time.Millisecond
	c.Server.HTTP.ReadTimeout = 90 * time.Millisecond
	c.Server.HTTP.WriteTimeout = 400 * time.Millisecond
	c.Server.HTTP.IdleTimeout = 100 * time.Millisecond
	path := filepath.Join(t.TempDir(), "config.yaml")
	write := func(c conf.Config) {
		b, _ := yaml.Marshal(c)
		if e := os.WriteFile(path, b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	write(c)
	m, e := New(context.Background(), path, c, loader, func(ctx context.Context, c conf.Config) (*Resource, error) {
		return &Resource{Identity: "test", Schema: "test", Close: func() {}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/context":
				<-r.Context().Done()
				_, _ = w.Write([]byte("expired"))
			case "/body":
				if _, e := io.ReadAll(r.Body); e != nil {
					http.Error(w, "expired", 408)
					return
				}
			case "/write":
				time.Sleep(350 * time.Millisecond)
				_, _ = w.Write([]byte("late"))
			default:
				_, _ = w.Write([]byte(c.App.Name))
			}
		})}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	h := NewHTTP(m)
	if e = h.Start(context.Background()); e != nil {
		t.Fatal(e)
	}
	defer h.Stop(context.Background())
	connect := func() net.Conn {
		conn, e := net.DialTimeout("tcp", addr, time.Second)
		if e != nil {
			t.Fatal(e)
		}
		_ = conn.SetDeadline(time.Now().Add(time.Second))
		return conn
	}
	conn := connect()
	_, _ = fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: localhost\r\nX-Slow:")
	start := time.Now()
	_, e = io.ReadAll(conn)
	_ = conn.Close()
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("header timeout ignored")
	}
	conn = connect()
	_, _ = fmt.Fprint(conn, "POST /body HTTP/1.1\r\nHost: localhost\r\nContent-Length: 100\r\n\r\na")
	start = time.Now()
	response, e := http.ReadResponse(bufio.NewReader(conn), nil)
	if e != nil {
		t.Fatal(e)
	}
	_ = response.Body.Close()
	_ = conn.Close()
	if response.StatusCode != 408 || time.Since(start) > 500*time.Millisecond {
		t.Fatal("body timeout ignored", response.StatusCode)
	}
	conn = connect()
	reader := bufio.NewReader(conn)
	_, _ = fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	response, e = http.ReadResponse(reader, nil)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	start = time.Now()
	_, e = reader.ReadByte()
	_ = conn.Close()
	if e == nil || time.Since(start) > 500*time.Millisecond {
		t.Fatal("idle timeout ignored")
	}
	conn = connect()
	reader = bufio.NewReader(conn)
	_, _ = fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	response, e = http.ReadResponse(reader, nil)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	_, _ = fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: localhost\r\nX-Slow:")
	start = time.Now()
	_, e = reader.ReadByte()
	_ = conn.Close()
	if e == nil || time.Since(start) > 500*time.Millisecond {
		t.Fatal("keepalive header timeout ignored")
	}
	c.App.Name = "watched"
	c.Server.HTTP.Timeout = 30 * time.Millisecond
	c.Server.HTTP.ReadTimeout = 40 * time.Millisecond
	c.Server.HTTP.WriteTimeout = 100 * time.Millisecond
	write(c)
	deadline := time.Now().Add(time.Second)
	for m.Config().App.Name != "watched" {
		if time.Now().After(deadline) {
			t.Fatal("watcher interval ignored")
		}
		time.Sleep(10 * time.Millisecond)
	}
	w := httptest.NewRecorder()
	start = time.Now()
	m.ServeHTTP(w, httptest.NewRequest("GET", "/context", nil))
	if w.Body.String() != "expired" || time.Since(start) > 200*time.Millisecond {
		t.Fatal("request timeout ignored")
	}
	response, e = http.Get("http://" + addr + "/write")
	if e == nil {
		_ = response.Body.Close()
		t.Fatal("write deadline ignored")
	}
}
