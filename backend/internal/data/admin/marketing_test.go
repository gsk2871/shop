package admin_test

import (
	"context"
	"encoding/json"
	"gorm.io/gorm"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/branding"
	admin "peptide-site/internal/data/admin"
	branding "peptide-site/internal/data/branding"
	"peptide-site/internal/testutil"
	"testing"
)

func TestMarketingSeedPreservesEdits(t *testing.T) {
	d, _ := testutil.Postgres(t)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}})
	r := branding.NewRepo(d)
	s, err := r.Settings(ctx)
	if err != nil || s.BrandIntro == nil || len(s.BrandIntro.Parts) != 2 || s.Announcement == nil {
		t.Fatal(s, err)
	}
	s.Brand = "Test"
	s.HeaderBackground = "#123456"
	s.BrandIntro = &biz.BrandIntro{Tagline: "Edited", Parts: []biz.BrandPart{}, Metrics: []biz.BrandMetric{}}
	s.Announcement.Items = []biz.AnnouncementItem{}
	s.ExpectedVersion = s.Version
	saved, err := r.SaveSettings(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	if err = d.DB.Transaction(func(tx *gorm.DB) error { return admin.SeedMarketing(tx) }); err != nil {
		t.Fatal(err)
	}
	got, err := r.Settings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(saved)
	b, _ := json.Marshal(got)
	if string(a) != string(b) {
		t.Fatal("seed overwrote saved values", string(a), string(b))
	}
}
