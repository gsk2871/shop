package branding

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/branding"
	mediaBiz "peptide-site/internal/biz/media"
	"peptide-site/internal/testutil"
	"strconv"
	"testing"
	"time"
)

func TestUnlimitedSchedulingThemeAndReferences(t *testing.T) {
	d, c := testutil.Postgres(t)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"banner.publish"}})
	_, _, root, commit, e := d.RuntimeMetadata(ctx, c.Data.Storage.UploadDir)
	if e != nil {
		t.Fatal(e)
	}
	if e = commit(); e != nil {
		t.Fatal(e)
	}
	r := NewRepo(d)
	asset := mediaBiz.Media{Status: "ready", StorageProvider: "local", Name: "Test", URL: "/uploads/" + root + "/sample.png", MediaType: "image", MIME: "image/png", Size: 1, Available: true, StorageRootID: root, StorageKey: "sample.png"}
	if e = d.DB.Table("media").Create(&asset).Error; e != nil {
		t.Fatal(e)
	}
	mediaID := strconv.FormatInt(asset.ID, 10)
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	var first biz.Banner
	for _, slot := range []string{"home_hero", "products_header", "research_header"} {
		for i := 0; i < 6; i++ {
			b := biz.Banner{SlotCode: slot, Title: fmt.Sprintf("Banner %d", i), ImageMediaID: mediaID, Sort: i, Enabled: true}
			if i == 4 {
				b.StartsAt = &future
			}
			if i == 5 {
				b.EndsAt = &past
			}
			saved, e := r.Save(ctx, 0, b)
			if e != nil {
				t.Fatal(e)
			}
			if first.ID == "" {
				first = saved
			}
		}
		items, total, e := r.List(ctx, slot, 1, 1, true)
		if e != nil || total != 4 || len(items) != 4 {
			t.Fatal("public limited or scheduling broken", len(items), total, e)
		}
	}
	if e = r.Media.Delete(ctx, asset.ID); e == nil {
		t.Fatal("referenced media deleted")
	}
	id, _ := strconv.ParseInt(first.ID, 10, 64)
	if _, e = r.Enabled(ctx, id, first.Version-1, false); e == nil {
		t.Fatal("stale banner accepted")
	}
	s, e := r.Settings(ctx)
	if e != nil {
		t.Fatal(e)
	}
	s.GoogleAds = biz.GoogleAds{Enabled: true, ID: "AW-18370098818", ConversionLabel: "test_label-1"}
	s.Brand = "Configured brand"
	s.BrandMark = "C·"
	s.HomeTitle = "Home from CMS"
	s.HomeContent = "First line\nSecond line"
	s.AboutTitle = "About from CMS"
	s.AboutContent = "Configured company details"
	s.PrivacyContent = "Configured privacy details"
	s.NewsTitle = "Company news"
	s.ResearchTitle = "Research articles"
	s.CookieContent = "Configured storage notice"
	s.LogoMediaID = mediaID
	s.Theme.Primary = "#112233"
	s.ExpectedVersion = s.Version
	s, e = r.SaveSettings(ctx, s)
	if e != nil || s.Logo == "" {
		t.Fatal(e)
	}
	if _, e = r.SaveSettings(ctx, s); e == nil {
		t.Fatal("stale theme accepted")
	}
	s, e = r.Settings(ctx)
	if e != nil || s.GoogleAds != (biz.GoogleAds{Enabled: true, ID: "AW-18370098818", ConversionLabel: "test_label-1"}) || s.Theme.Primary != "#112233" || s.BrandMark != "C·" || s.AboutContent != "Configured company details" || s.HomeContent != "First line\nSecond line" || s.PrivacyContent != "Configured privacy details" || s.NewsTitle != "Company news" || s.CookieContent != "Configured storage notice" {
		t.Fatal(e, s)
	}
	if e = r.Media.Delete(ctx, asset.ID); e == nil {
		t.Fatal("logo ref ignored")
	}
	if e = d.DB.Transaction(func(tx *gorm.DB) error { return tx.Exec("DELETE FROM media_reference WHERE owner_type='banner'").Error }); e != nil {
		t.Fatal(e)
	}
}
