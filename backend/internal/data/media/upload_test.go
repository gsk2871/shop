package media_test

import (
	"bytes"
	"context"
	"errors"
	"gorm.io/gorm"
	"image"
	"image/png"
	"os"
	"path/filepath"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/media"
	"peptide-site/internal/conf"
	. "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/cosstore"
	"peptide-site/internal/testutil"
	"strings"
	"sync"
	"testing"
	"time"
)

type testStore struct {
	mu                   sync.Mutex
	objects              map[string][]byte
	copies               int
	authorized           cosstore.Location
	failRead, failDelete bool
	delay                time.Duration
	deletes              map[string]int
}

func (s *testStore) Authorize(_ context.Context, l cosstore.Location, keys []string, _ int64, _ time.Duration) (cosstore.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authorized = l
	for _, key := range keys {
		if !strings.Contains(key, "/temporary/") {
			return cosstore.Credential{}, errors.New("unsafe grant")
		}
	}
	return cosstore.Credential{ExpiredTime: int(time.Now().Add(30 * time.Minute).Unix())}, nil
}
func (s *testStore) Read(_ context.Context, _ cosstore.Location, key string, limit int64) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failRead {
		return nil, "", errors.New("network")
	}
	b, ok := s.objects[key]
	if !ok {
		return nil, "", errors.New("missing")
	}
	if int64(len(b)) > limit {
		return nil, "", cosstore.ErrInvalidObject
	}
	return b, "etag", nil
}
func (s *testStore) Promote(_ context.Context, _ cosstore.Location, source, target, etag, mime string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.copies++
	s.objects[target] = append([]byte(nil), s.objects[source]...)
	return nil
}
func (s *testStore) Delete(ctx context.Context, _ cosstore.Location, key string) error {
	if s.delay > 0 {
		timer := time.NewTimer(s.delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failDelete {
		return errors.New("network")
	}
	if s.deletes != nil {
		s.deletes[key]++
	}
	delete(s.objects, key)
	return nil
}
func imageBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func testSetup(t *testing.T) (*Repo, conf.Storage, context.Context, *testStore) {
	d, c := testutil.Postgres(t)
	s := &testStore{objects: map[string][]byte{}}
	r := NewRepo(d)
	r.Store = s
	c.Data.Storage.COS = conf.COS{Enabled: true, Bucket: "test-12345", Region: "ap-hongkong", PublicDomain: "https://media.example.com", KeyPrefix: "site-a"}
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}})
	return r, c.Data.Storage, ctx, s
}
func initImage(t *testing.T, r *Repo, c conf.Storage, ctx context.Context, s *testStore, group *int64) (UploadResult, []byte) {
	t.Helper()
	body := imageBytes(t)
	res, err := r.InitUpload(ctx, c, UploadInput{GroupID: group, Files: []UploadFile{{Name: "test.png", MIME: "image/png", Size: int64(len(body))}}})
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.objects[res.Items[0].Key] = body
	s.mu.Unlock()
	return res, body
}
func TestUploadSnapshotAndLifecycle(t *testing.T) {
	r, c, ctx, s := testSetup(t)
	group, err := r.SaveGroup(ctx, 0, "产品")
	if err != nil {
		t.Fatal(err)
	}
	batch, _ := initImage(t, r, c, ctx, s, &group.ID)
	id := batch.Items[0].ID
	if _, err := r.Get(ctx, id); err == nil {
		t.Fatal("pending selectable")
	}
	if err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return SyncReferences(tx, "product", "1", map[string][]int64{"image": {id}}) }); err == nil {
		t.Fatal("pending referenced")
	}
	if err := r.DeleteGroup(ctx, group.ID); err != nil {
		t.Fatal(err)
	}
	other := auth.WithIdentity(ctx, &auth.Identity{User: auth.User{ID: 999}})
	if _, err := r.CompleteUpload(other, id); err == nil {
		t.Fatal("foreign completion")
	}
	if _, err := r.RefreshUpload(other, batch.BatchID); err == nil {
		t.Fatal("foreign refresh")
	}
	if _, err := r.RefreshUpload(ctx, batch.BatchID); err != nil {
		t.Fatal(err)
	}
	s.failRead = true
	if _, err := r.CompleteUpload(ctx, id); err == nil {
		t.Fatal("network ignored")
	}
	s.failRead = false
	c.COS.Bucket = "other-54321"
	c.COS.PublicDomain = "https://other.example.com"
	c.MaxImageBytes = 1
	row, err := r.CompleteUpload(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if row.GroupID != nil || row.Status != "ready" || !row.Available || *row.Width != 2 || *row.Height != 3 || !strings.HasPrefix(row.URL, "https://media.example.com/site-a/images/") {
		t.Fatal(row)
	}
	if s.authorized.Bucket != "test-12345" {
		t.Fatal("batch location changed")
	}
	if _, err := r.CompleteUpload(ctx, id); err != nil || s.copies != 1 {
		t.Fatal("completion not idempotent", err)
	}
	if err := r.CancelUpload(ctx, id); err == nil {
		t.Fatal("ready canceled")
	}
	if err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return SyncReferences(tx, "product", "1", map[string][]int64{"image": {id}}) }); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, id); err == nil {
		t.Fatal("referenced image deleted")
	}
	if err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return SyncReferences(tx, "product", "1", nil) }); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	s.failDelete = true
	if err := r.Cleanup(ctx); err == nil {
		t.Fatal("cleanup failure lost")
	}
	s.failDelete = false
	if err := r.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.objects[row.StorageKey]; ok {
		t.Fatal("object not deleted")
	}
}
func TestInvalidCanceledAndExpiredUploads(t *testing.T) {
	r, c, ctx, s := testSetup(t)
	for _, file := range []UploadFile{{"video.mp4", "video/mp4", 12}, {"bad.png", "image/png", c.MaxImageBytes + 1}, {"bad.txt", "image/png", 12}} {
		if _, err := r.InitUpload(ctx, c, UploadInput{Files: []UploadFile{file}}); err == nil {
			t.Fatal("invalid init", file)
		}
	}
	batch, _ := initImage(t, r, c, ctx, s, nil)
	s.objects[batch.Items[0].Key] = []byte("fake")
	if _, err := r.CompleteUpload(ctx, batch.Items[0].ID); err == nil {
		t.Fatal("fake image accepted")
	}
	var invalid biz.Media
	r.Data.DB.Table("media").Where("id=?", batch.Items[0].ID).Take(&invalid)
	if invalid.Status != "failed" || invalid.Available {
		t.Fatal(invalid)
	}
	canceled, _ := initImage(t, r, c, ctx, s, nil)
	for i := 0; i < 2; i++ {
		if err := r.CancelUpload(ctx, canceled.Items[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.CompleteUpload(ctx, canceled.Items[0].ID); err == nil {
		t.Fatal("canceled completed")
	}
	expired, _ := initImage(t, r, c, ctx, s, nil)
	if err := r.Data.DB.Exec("UPDATE media_upload_batch SET expires_at=now()-interval '2 hours',cleanup_after=now()-interval '1 hour'").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := r.CompleteUpload(ctx, expired.Items[0].ID); err == nil {
		t.Fatal("expired completed")
	}
	if _, err := r.RefreshUpload(ctx, expired.BatchID); err == nil {
		t.Fatal("expired refreshed")
	}
	if err := r.CleanupUploads(ctx); err != nil {
		t.Fatal(err)
	}
	if len(s.objects) != 0 {
		t.Fatal("temporary objects retained", len(s.objects))
	}
}
func TestDirectoriesAndReferenceRace(t *testing.T) {
	r, c, ctx, s := testSetup(t)
	g, err := r.SaveGroup(ctx, 0, "产品")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.SaveGroup(ctx, 0, "产品"); err == nil {
		t.Fatal("duplicate group")
	}
	a, _ := initImage(t, r, c, ctx, s, nil)
	b, _ := initImage(t, r, c, ctx, s, nil)
	id, other := a.Items[0].ID, b.Items[0].ID
	for _, n := range []int64{id, other} {
		if _, err := r.CompleteUpload(ctx, n); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Move(ctx, []int64{id, other}, &g.ID); err != nil {
		t.Fatal(err)
	}
	if err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return SyncReferences(tx, "article", "1", map[string][]int64{"body": {id}}) }); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteBatch(ctx, []int64{id, other}); err == nil {
		t.Fatal("batch partially deleted")
	}
	if _, err := r.Get(ctx, other); err != nil {
		t.Fatal("batch deletion not atomic")
	}
	if err := r.DeleteGroup(ctx, g.ID); err != nil {
		t.Fatal(err)
	}
	if err := r.Move(ctx, []int64{id}, &g.ID); err == nil {
		t.Fatal("moved to deleted group")
	}
	if _, err := r.InitUpload(ctx, c, UploadInput{GroupID: &g.ID, Files: []UploadFile{{"test.png", "image/png", 1}}}); err == nil {
		t.Fatal("upload into deleted group")
	}
	var wg sync.WaitGroup
	wg.Add(2)
	var deleted, linked error
	go func() { defer wg.Done(); deleted = r.Delete(ctx, other) }()
	go func() {
		defer wg.Done()
		linked = r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return SyncReferences(tx, "article", "2", map[string][]int64{"body": {other}})
		})
	}()
	wg.Wait()
	if deleted == nil && linked == nil {
		t.Fatal("dangling reference")
	}
}

func (s *testStore) PutImage(_ context.Context, _ cosstore.Location, key, mime string, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = append([]byte(nil), body...)
	return nil
}
func TestCleanupPersistsProgressAcrossDeadlines(t *testing.T) {
	r, c, ctx, s := testSetup(t)
	body := imageBytes(t)
	b, err := r.InitUpload(ctx, c, UploadInput{Files: []UploadFile{{"one.png", "image/png", int64(len(body))}, {"two.png", "image/png", int64(len(body))}, {"three.png", "image/png", int64(len(body))}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range b.Items {
		s.objects[item.Key] = body
	}
	if err := r.Data.DB.Exec("UPDATE media_upload_batch SET expires_at=now()-interval '2 hours',cleanup_after=now()-interval '1 hour'").Error; err != nil {
		t.Fatal(err)
	}
	s.delay = 20 * time.Millisecond
	s.deletes = map[string]int{}
	limited, cancel := context.WithTimeout(ctx, 55*time.Millisecond)
	err = r.CleanupUploads(limited)
	cancel()
	if err == nil {
		t.Fatal("expected interrupted cleanup")
	}
	var completed int64
	if err := r.Data.DB.Table("media").Where("temporary_cleaned").Count(&completed).Error; err != nil || completed < 1 {
		t.Fatal("cleanup progress rolled back", completed, err)
	}
	s.delay = 0
	if err := r.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.CleanupUploads(ctx); err != nil {
		t.Fatal(err)
	}
	if len(s.objects) != 0 || s.deletes[b.Items[0].Key] != 1 {
		t.Fatal("cleanup repeated completed objects", s.deletes)
	}
	var batch Batch
	r.Data.DB.Table("media_upload_batch").Where("id=?", b.BatchID).Take(&batch)
	if batch.CleanedAt == nil {
		t.Fatal("cleanup not completed")
	}
}
func TestLocalMigrationPreservesReferencesAndFiles(t *testing.T) {
	r, c, ctx, s := testSetup(t)
	pool, err := r.Data.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, _, root, commit, err := r.Data.RuntimeMetadata(ctx, c.UploadDir)
	if err != nil {
		t.Fatal(err)
	}
	if err = commit(); err != nil {
		t.Fatal(err)
	}
	body := imageBytes(t)
	key := "0123456789abcdef0123456789abcdef.png"
	path := filepath.Join(c.UploadDir, key)
	if err = os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	os.Chtimes(path, old, old)
	row := biz.Media{Name: "local.png", URL: "/uploads/" + root + "/" + key, MIME: "image/png", Size: int64(len(body)), MediaType: "image", Available: true, Status: "ready", StorageProvider: "local", StorageRootID: root, StorageKey: key}
	if err = r.Data.DB.Table("media").Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err = r.Data.DB.Transaction(func(tx *gorm.DB) error {
		return SyncReferences(tx, "product", "1", map[string][]int64{"image": {row.ID}})
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = r.MigrateLocal(ctx, c, row.ID, s); err != nil {
			t.Fatal(err)
		}
	}
	migrated, target, err := r.FindFile(ctx, root, key)
	if err != nil || migrated.ID != row.ID || migrated.URL != row.URL || migrated.StorageProvider != "cos" || !strings.HasPrefix(target, "https://media.example.com/site-a/images/") {
		t.Fatal(migrated, target, err)
	}
	refs, err := r.References(ctx, row.ID)
	if err != nil || len(refs) != 1 {
		t.Fatal("references lost", err)
	}
	if err = r.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal("local backup removed", err)
	}
}
