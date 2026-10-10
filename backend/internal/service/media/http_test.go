package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"gorm.io/gorm"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	mediaBiz "peptide-site/internal/biz/media"
	"peptide-site/internal/conf"
	media "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/filestaging"
	"peptide-site/internal/testutil"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStreamingUploadPolicyAndReferences(t *testing.T) {
	d, c := testutil.Postgres(t)
	ctx := context.Background()
	_, _, root, commit, e := d.RuntimeMetadata(ctx, c.Data.Storage.UploadDir)
	if e != nil {
		t.Fatal(e)
	}
	if e = commit(); e != nil {
		t.Fatal(e)
	}
	repo := media.NewRepo(d)
	h, e := NewHandler(repo, c, root)
	if e != nil {
		t.Fatal(e)
	}
	var pngData bytes.Buffer
	if e = png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 2))); e != nil {
		t.Fatal(e)
	}
	upload := func(h *Handler, data []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		m := multipart.NewWriter(&body)
		p, _ := m.CreateFormFile("file", "sample.png")
		_, _ = p.Write(data)
		_ = m.Close()
		req := httptest.NewRequest("POST", "/admin/api/v1/media", &body)
		req.Header.Set("Content-Type", m.FormDataContentType())
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	w := upload(h, pngData.Bytes())
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var asset mediaBiz.Media
	if e = json.Unmarshal(w.Body.Bytes(), &asset); e != nil {
		t.Fatal(e)
	}
	if asset.Width == nil || *asset.Width != 2 || asset.StorageRootID != "" {
		t.Fatal("unsafe or missing DTO", asset)
	}
	req := httptest.NewRequest("GET", asset.URL, nil)
	req.Header.Set("Range", "bytes=0-7")
	response := httptest.NewRecorder()
	h.PublicFiles().ServeHTTP(response, req)
	if response.Code != 206 || response.Body.Len() != 8 {
		t.Fatal("Range failed", response.Code)
	}
	small := c
	small.Data.Storage.MaxImageBytes = 10
	smallHandler, _ := NewHandler(repo, small, root)
	if upload(smallHandler, pngData.Bytes()).Code != 400 {
		t.Fatal("size limit ignored")
	}
	// Use structurally valid MP4 boxes so failures exercise the video limit,
	// rather than being rejected only by MIME detection or container validation.
	var video bytes.Buffer
	for _, box := range []struct {
		kind    string
		payload []byte
	}{
		{"ftyp", []byte("isom\x00\x00\x00\x00isommp42")}, {"moov", nil}, {"mdat", make([]byte, 600)},
	} {
		_ = binary.Write(&video, binary.BigEndian, uint32(8+len(box.payload)))
		video.WriteString(box.kind)
		video.Write(box.payload)
	}
	for _, tc := range []struct {
		name      string
		body      []byte
		configure func(*conf.Config)
	}{
		{"video", video.Bytes(), func(c *conf.Config) { c.Data.Storage.MaxVideoBytes = 10 }},
		{"document", []byte("%PDF-1.7\n" + strings.Repeat("x", 600)), func(c *conf.Config) { c.Data.Storage.MaxDocumentBytes = 10 }},
	} {
		t.Run(tc.name+" limit", func(t *testing.T) {
			limited := c
			tc.configure(&limited)
			handler, err := NewHandler(repo, limited, root)
			if err != nil {
				t.Fatal(err)
			}
			if response := upload(handler, tc.body); response.Code != 400 {
				t.Fatal(response.Code, response.Body.String())
			}
		})
	}
	if upload(h, []byte("<svg><script>bad</script></svg>")).Code != 400 {
		t.Fatal("active SVG accepted")
	}
	restricted := c
	restricted.Data.Storage.AllowedTypes = []string{"application/pdf"}
	blocked, _ := NewHandler(repo, restricted, root)
	if upload(blocked, pngData.Bytes()).Code != 400 {
		t.Fatal("type policy ignored")
	}
	entries, _ := os.ReadDir(c.Data.Storage.UploadDir)
	if len(entries) != 1 {
		t.Fatal("failed upload orphan", len(entries))
	}
	if e = d.DB.Transaction(func(tx *gorm.DB) error {
		return media.SyncReferences(tx, "article", "1", map[string][]int64{"body": {asset.ID}})
	}); e != nil {
		t.Fatal(e)
	}
	if e = repo.Delete(ctx, asset.ID); e == nil {
		t.Fatal("referenced media deleted")
	}
	if e = d.DB.Transaction(func(tx *gorm.DB) error { return media.SyncReferences(tx, "article", "1", nil) }); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	var deleted, linked error
	go func() { defer wg.Done(); deleted = repo.Delete(ctx, asset.ID) }()
	go func() {
		defer wg.Done()
		linked = d.DB.Transaction(func(tx *gorm.DB) error {
			return media.SyncReferences(tx, "article", "2", map[string][]int64{"body": {asset.ID}})
		})
	}()
	wg.Wait()
	if deleted == nil && linked == nil {
		t.Fatal("delete/reference race allowed dangling ref")
	}
	if linked == nil {
		_ = d.DB.Transaction(func(tx *gorm.DB) error { return media.SyncReferences(tx, "article", "2", nil) })
		if e = repo.Delete(ctx, asset.ID); e != nil {
			t.Fatal(e)
		}
	}
	if e = repo.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	response = httptest.NewRecorder()
	h.PublicFiles().ServeHTTP(response, httptest.NewRequest("GET", asset.URL, nil))
	if response.Code != 404 {
		t.Fatal("deleted asset public", response.Code)
	}
	orphan := filepath.Join(c.Data.Storage.UploadDir, ".upload-crash")
	_ = os.WriteFile(orphan, []byte("crash"), 0600)
	old := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(orphan, old, old)
	release := filestaging.Protect(orphan)
	if e = repo.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(orphan); e != nil {
		t.Fatal("active stage removed")
	}
	release()
	if e = repo.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	entries, _ = os.ReadDir(c.Data.Storage.UploadDir)
	if len(entries) != 0 {
		t.Fatal("cleanup failed", entries)
	}
}
