package catalog

import (
	"context"
	"strconv"
	"testing"

	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/catalog"
	"peptide-site/internal/testutil"
)

func fixtureMedia(t *testing.T, r *Repo, kind string) string {
	t.Helper()
	var id int64
	if err := r.Data.DB.Raw("INSERT INTO media(name,url,mime,size,media_type,storage_root_id,storage_key) VALUES(?,?,?,1,?,'test',gen_random_uuid()::text) RETURNING id", kind, "/uploads/test/"+kind, kind+"/test", kind).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	return strconv.FormatInt(id, 10)
}

func TestProductMediaAndAutomaticPath(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"product.publish", "product.price.update"}})
	category, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Media"})
	if err != nil {
		t.Fatal(err)
	}
	image, video, report := fixtureMedia(t, r, "image"), fixtureMedia(t, r, "video"), fixtureMedia(t, r, "image")
	unavailable, deleted := fixtureMedia(t, r, "image"), fixtureMedia(t, r, "image")
	if err := d.DB.Exec("UPDATE media SET available=false WHERE id=?", unavailable).Error; err != nil {
		t.Fatal(err)
	}
	if err := d.DB.Exec("UPDATE media SET deleted_at=now() WHERE id=?", deleted).Error; err != nil {
		t.Fatal(err)
	}
	fixtureMedia(t, r, "document")
	assets, total, err := r.Media.List(ctx, 1, 20, "", "")
	if err != nil || total != 2 || len(assets) != 2 {
		t.Fatal("media list includes unsupported/unavailable assets", total, err)
	}
	if _, _, err := r.Media.List(ctx, 1, 20, "document", ""); err == nil {
		t.Fatal("document filter accepted")
	}
	input := biz.Product{Name: "Test product", CategoryID: category.ID, Status: "draft", ImageMediaID: image, ReportMediaID: report}
	for _, test := range []struct {
		name   string
		change func(*biz.Product)
	}{
		{"unavailable", func(p *biz.Product) { p.ImageMediaID = unavailable }},
		{"deleted", func(p *biz.Product) { p.ImageMediaID = deleted }},
		{"missing image", func(p *biz.Product) { p.ImageMediaID = "" }},
		{"video as image", func(p *biz.Product) { p.ImageMediaID = video }},
		{"video as report", func(p *biz.Product) { p.ReportMediaID = video }},
		{"missing asset", func(p *biz.Product) { p.ImageMediaID = "999999" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := input
			test.change(&bad)
			if _, err := r.Save(ctx, 0, bad); err == nil {
				t.Fatal("invalid media saved")
			}
			if _, err := r.Preview(ctx, bad); err == nil {
				t.Fatal("invalid media previewed")
			}
		})
	}
	preview, err := r.Preview(ctx, input)
	if err != nil || preview["image"] != "/uploads/test/image" {
		t.Fatal(preview, err)
	}
	p, err := r.Save(ctx, 0, input)
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug == "" || p.ImageMediaID != image || p.ReportMediaID != report || p.Report == "" {
		t.Fatal(p)
	}
	duplicate, err := r.Save(ctx, 0, input)
	if err != nil || duplicate.Slug == p.Slug {
		t.Fatal("duplicate title collided", err)
	}
	duplicateID, _ := strconv.ParseInt(duplicate.ID, 10, 64)
	if err := r.Delete(ctx, duplicateID, duplicate.Version); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{image, report} {
		id, _ := strconv.ParseInt(raw, 10, 64)
		if err := r.Media.Delete(ctx, id); err == nil {
			t.Fatal("referenced media deleted")
		}
	}
	p.ExpectedVersion = p.Version
	originalSlug := p.Slug
	p.Slug = ""
	p.Name = "Renamed"
	p.ReportMediaID = ""
	id, _ := strconv.ParseInt(p.ID, 10, 64)
	saved, err := r.Save(ctx, id, p)
	if err != nil || saved.Slug != originalSlug || saved.ReportMediaID != "" {
		t.Fatal(saved, err)
	}
	for _, raw := range []string{report} {
		assetID, _ := strconv.ParseInt(raw, 10, 64)
		refs, err := r.Media.References(ctx, assetID)
		if err != nil || len(refs) != 0 {
			t.Fatal("stale references", refs, err)
		}
	}
	// Legacy products may still be read, but cannot be published without an image.
	if err := d.DB.Exec("DELETE FROM media_reference WHERE owner_type='product' AND owner_id=?", p.ID).Error; err != nil {
		t.Fatal(err)
	}
	price := "1.00"
	saved.ImageMediaID = image
	saved.ExpectedVersion = saved.Version
	saved.SKUs[0].SalePrice = &price
	saved, err = r.Save(ctx, id, saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.DB.Exec("DELETE FROM media_reference WHERE owner_type='product' AND owner_id=?", p.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = r.Status(ctx, id, saved.Version, "published"); err == nil {
		t.Fatal("published without image")
	}
}
