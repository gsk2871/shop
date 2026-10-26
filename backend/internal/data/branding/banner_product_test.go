package branding

import (
	"context"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/branding"
	catalog "peptide-site/internal/biz/catalog"
	"peptide-site/internal/conf"
	catalogData "peptide-site/internal/data/catalog"
	"strconv"
	"testing"
)

func TestHomeBannerProductLifecycle(t *testing.T) {
	r, asset := socialFixture(t)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"banner.publish", "product.publish", "product.price.update"}})
	products := catalogData.NewRepo(r.Data, conf.Config{})
	category, err := products.SaveCategory(ctx, 0, catalog.Category{Name: "Banner products"})
	if err != nil {
		t.Fatal(err)
	}
	price := "10.00"
	p, err := products.Save(ctx, 0, catalog.Product{Name: "Linked product", Slug: "linked-product", CategoryID: category.ID, ImageMediaID: strconv.FormatInt(asset.ID, 10), Status: "published", SKUs: []catalog.SKU{{Key: "default", Enabled: true, Currency: "USD", SalePrice: &price, SpecValueKeys: []string{}}}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.Save(ctx, 0, biz.Banner{SlotCode: "home_hero", Title: "Internal campaign name", ImageMediaID: p.ImageMediaID, ProductID: p.ID, Link: "https://example.com/untrusted", CTAText: "ignored", Enabled: true})
	if err != nil || b.Link != "/products/"+p.Slug || b.CTAText != biz.ProductBannerCTA {
		t.Fatal(b, err)
	}
	public := biz.Public(b)
	if public["title"] != "" || public["subtitle"] != "" {
		t.Fatal("internal campaign text exposed", public)
	}
	if _, ok := public["productId"]; ok {
		t.Fatal("management metadata exposed")
	}
	choices, err := r.ProductOptions(ctx, "Linked")
	if err != nil || len(choices) != 1 || choices[0]["id"] != p.ID {
		t.Fatal(choices, err)
	}
	pid, _ := strconv.ParseInt(p.ID, 10, 64)
	bid, _ := strconv.ParseInt(b.ID, 10, 64)
	p, err = products.Status(ctx, pid, p.Version, "offline")
	if err != nil {
		t.Fatal(err)
	}
	b, err = r.Get(ctx, bid)
	if err != nil || b.Link != "" || b.Image == "" {
		t.Fatal("offline product link visible", b, err)
	}
	choices, err = r.ProductOptions(ctx, "")
	if err != nil || len(choices) != 0 {
		t.Fatal("offline selectable", choices, err)
	}
	if err := products.Delete(ctx, pid, p.Version); err != nil {
		t.Fatal(err)
	}
	b, err = r.Get(ctx, bid)
	if err != nil || b.Link != "" {
		t.Fatal("deleted product link visible", b, err)
	}
	b.ExpectedVersion = b.Version
	b.Title = "Keep the image"
	b, err = r.Save(ctx, bid, b)
	if err != nil {
		t.Fatal("existing deleted association prevents editing", err)
	}
	b.ExpectedVersion = b.Version
	b.ProductID = ""
	b, err = r.Save(ctx, bid, b)
	if err != nil || b.Link != "" || b.ProductID != "" {
		t.Fatal("clear association", b, err)
	}
	b.ExpectedVersion = b.Version
	b.ProductID = "99999999"
	if _, err := r.Save(ctx, bid, b); err == nil {
		t.Fatal("nonexistent product accepted")
	}
}
