package cosstore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"peptide-site/internal/conf"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestExactUploadPolicy(t *testing.T) {
	p := Policy(Location{"images-12345", "ap-hongkong"}, []string{"site/temporary/batch/image.png"}, 1024)
	body, _ := json.Marshal(p)
	text := string(body)
	if strings.Contains(text, "*") || strings.Contains(text, "DeleteObject") || !strings.Contains(text, "uid/12345:images-12345/site/temporary/batch/image.png") || !strings.Contains(text, "cos:content-length") {
		t.Fatal(text)
	}
	if len(p.Statement) != 1 || len(p.Statement[0].Action) != 1 || p.Statement[0].Action[0] != "name/cos:PutObject" {
		t.Fatal(text)
	}
}
func TestConditionalReadAndPromotion(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	get, copy := false, false
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		header := http.Header{}
		header.Set("ETag", "frozen")
		header.Set("Content-Length", "3")
		body := ""
		switch r.Method {
		case "HEAD":
		case "GET":
			get = true
			if r.Header.Get("If-Match") != "frozen" {
				t.Fatal("missing conditional read")
			}
			body = "abc"
		case "PUT":
			copy = true
			if r.Header.Get("x-cos-copy-source-If-Match") != "frozen" || r.Header.Get("x-cos-metadata-directive") != "Replaced" || r.Header.Get("Content-Type") != "image/png" {
				t.Fatal("unsafe copy headers")
			}
			body = "<CopyObjectResult><ETag>frozen</ETag></CopyObjectResult>"
		default:
			t.Fatal(r.Method)
		}
		return &http.Response{StatusCode: 200, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	c := New(conf.COS{SecretID: "test-id", SecretKey: "test-key"})
	loc := Location{"images-12345", "ap-hongkong"}
	body, etag, err := c.Read(context.Background(), loc, "temporary/a", 3)
	if err != nil || string(body) != "abc" || !get {
		t.Fatal(err)
	}
	if err = c.Promote(context.Background(), loc, "temporary/a", "images/a", etag, "image/png"); err != nil || !copy {
		t.Fatal(err)
	}
	if _, _, err = c.Read(context.Background(), loc, "temporary/a", 2); err != ErrInvalidObject {
		t.Fatal("oversized accepted", err)
	}
}
