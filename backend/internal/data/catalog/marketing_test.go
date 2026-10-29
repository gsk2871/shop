package catalog

import (
	"context"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/catalog"
	"peptide-site/internal/testutil"
	"strconv"
	"testing"
)

func TestMarketingPricesAndPermissions(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"product.price.update", "product.publish"}})
	category, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Marketing"})
	if err != nil {
		t.Fatal(err)
	}
	sale, original, stars := "8.00", "10.00", 4
	p, err := r.Save(ctx, 0, biz.Product{Name: "Marketing product", CategoryID: category.ID, ImageMediaID: fixtureMedia(t, r, "image"), Status: "published", PurityLabel: "99% purity", Recommendation: &stars, SKUs: []biz.SKU{{Key: "default", Enabled: true, Currency: "USD", SalePrice: &sale, OriginalPrice: &original}}})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := strconv.ParseInt(p.ID, 10, 64)
	skuID, _ := strconv.ParseInt(p.SKUs[0].ID, 10, 64)
	if p.FromPrice.DisplayDiscount != "USD 2.00" || p.PurityLabel != "99% purity" || p.Recommendation == nil || *p.Recommendation != 4 {
		t.Fatal(p)
	}
	public, _, err := r.List(ctx, true, 1, 20, "", 0, "", nil)
	if err != nil || len(public) != 1 || public[0]["purityLabel"] != "99% purity" {
		t.Fatal(public, err)
	}
	denied := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}})
	p.ExpectedVersion = p.Version
	changed := "12.00"
	p.SKUs[0].OriginalPrice = &changed
	if _, err = r.Save(denied, id, p); err == nil {
		t.Fatal("original-price change bypassed price permission")
	}
	if _, err = r.Price(ctx, id, skuID, p.Version, "11.00", "USD", "10.00"); err == nil {
		t.Fatal("discount exceeds original")
	}
	if _, err = r.Price(ctx, id, skuID, p.Version, "8.00", "USD", "9.001"); err == nil {
		t.Fatal("precision accepted")
	}
	p, err = r.Price(ctx, id, skuID, p.Version, "8.00", "USD", "8.00")
	if err != nil || p.FromPrice.DisplayDiscount != "" {
		t.Fatal(p, err)
	}
	p.ExpectedVersion = p.Version
	invalid := 6
	p.Recommendation = &invalid
	if _, err = r.Save(ctx, id, p); err == nil {
		t.Fatal("six stars accepted")
	}
}
