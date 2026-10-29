package branding

import (
	"context"
	auth "peptide-site/internal/biz/admin"
	mediaBiz "peptide-site/internal/biz/media"
	"peptide-site/internal/testutil"
	"strconv"
	"testing"
)

func TestBackgroundReferenceLifecycle(t *testing.T) {
	d, c := testutil.Postgres(t)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}})
	_, _, root, commit, err := d.RuntimeMetadata(ctx, c.Data.Storage.UploadDir)
	if err != nil {
		t.Fatal(err)
	}
	if err = commit(); err != nil {
		t.Fatal(err)
	}
	r := NewRepo(d)
	asset := mediaBiz.Media{Status: "ready", StorageProvider: "local", Name: "Background", URL: "/uploads/" + root + "/background.png", MediaType: "image", MIME: "image/png", Size: 1, Available: true, StorageRootID: root, StorageKey: "background.png"}
	if err = d.DB.Table("media").Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	s, err := r.Settings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s.Brand = "Test"
	s.BackgroundMediaID = strconv.FormatInt(asset.ID, 10)
	s.BackgroundImage = "https://untrusted.invalid/ignored.png"
	s.ExpectedVersion = s.Version
	s, err = r.SaveSettings(ctx, s)
	if err != nil || s.BackgroundImage != asset.URL {
		t.Fatal(s, err)
	}
	if err = r.Media.Delete(ctx, asset.ID); err == nil {
		t.Fatal("referenced background could be deleted")
	}
	s.BackgroundMediaID = ""
	s.ExpectedVersion = s.Version
	s, err = r.SaveSettings(ctx, s)
	if err != nil || s.BackgroundImage != "" {
		t.Fatal(s, err)
	}
	if _, err = r.Media.Get(ctx, asset.ID); err != nil {
		t.Fatal("clearing configuration removed media", err)
	}
	var count int64
	if err = d.DB.Table("media_reference").Where("owner_type='site_config' AND field='background'").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("background reference not cleared", count, err)
	}
}
