package catalog

import (
	"context"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/catalog"
	"peptide-site/internal/pkg/richtext"
	"peptide-site/internal/testutil"
	"strconv"
	"sync"
	"testing"
)

func TestProductAggregateAndConcurrentVersion(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	imageID := fixtureMedia(t, r, "image")
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"product.publish", "product.price.update"}})
	category, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Compounds"})
	if err != nil {
		t.Fatal(err)
	}
	price := "12.34"
	input := biz.Product{ImageMediaID: imageID, CategoryID: category.ID, Name: "Research compound", Slug: "research-compound", Status: "draft", Detail: richtext.Document{HTML: "<p>Test</p>", MediaIDs: []string{}}, Specs: []biz.Spec{{Key: "size", Name: "Size", Values: []biz.Value{{Key: "small", Name: "Small"}, {Key: "large", Name: "Large"}}}}, SKUs: []biz.SKU{{Key: "small-sku", Enabled: true, SpecValueKeys: []string{"small"}, Currency: "USD", SalePrice: &price}, {Key: "large-sku", Enabled: false, SpecValueKeys: []string{"large"}, Currency: "USD", SalePrice: &price}}}
	product, err := r.Save(ctx, 0, input)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := strconv.ParseInt(product.ID, 10, 64)
	if product.FromPrice == nil || product.FromPrice.SalePrice != "12.34" || product.Detail.HTML != "<p>Test</p>" {
		t.Fatal("aggregate price or body corrupted", product)
	}
	if _, err := r.Public(ctx, product.Slug); err == nil {
		t.Fatal("draft exposed")
	}
	product, err = r.Status(ctx, id, product.Version, "published")
	if err != nil {
		t.Fatal(err)
	}
	public, err := r.Public(ctx, product.Slug)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"status", "version", "expectedVersion", "images", "updatedBy"} {
		if _, ok := public[key]; ok {
			t.Fatal("management field leaked", key)
		}
	}
	if len(public["skus"].([]biz.PublicSKU)) != 1 {
		t.Fatal("disabled SKU leaked")
	}
	originalSpec, originalSKU := product.Specs[0].ID, product.SKUs[0].ID
	product.ExpectedVersion = product.Version
	product.Name = "Updated"
	saved, err := r.Save(ctx, id, product)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Specs[0].ID != originalSpec || saved.SKUs[0].ID != originalSKU {
		t.Fatal("stable aggregate IDs replaced")
	}
	stale := product
	stale.Name = "Stale"
	if _, err := r.Save(ctx, id, stale); err == nil {
		t.Fatal("stale aggregate accepted")
	}
	unprivileged := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"product.update"}})
	unauthorized := saved
	unauthorized.ExpectedVersion = saved.Version
	unauthorized.SKUs = append([]biz.SKU(nil), saved.SKUs...)
	value := "10.00"
	unauthorized.SKUs[0].SalePrice = &value
	if _, err := r.Save(unprivileged, id, unauthorized); err == nil {
		t.Fatal("PUT bypassed price permission")
	}
	wrong := saved
	wrong.ExpectedVersion = saved.Version
	wrong.Specs = append([]biz.Spec(nil), saved.Specs...)
	wrong.Specs[0].ID = "999999"
	if _, err := r.Save(ctx, id, wrong); err == nil {
		t.Fatal("foreign aggregate ID accepted")
	}
	skuID, _ := strconv.ParseInt(saved.SKUs[0].ID, 10, 64)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, value := range []string{"0.01", "20.00"} {
		wg.Add(1)
		go func(value string) {
			defer wg.Done()
			_, err := r.Price(ctx, id, skuID, saved.Version, value, "USD")
			results <- err
		}(value)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("concurrent version guard failed", success)
	}
	latest, err := r.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != saved.Version+1 {
		t.Fatal("price did not bump product version")
	}
	preview := latest
	preview.Name = "Unsaved"
	preview.ExpectedVersion = latest.Version
	if _, err := r.Preview(ctx, preview); err != nil {
		t.Fatal(err)
	}
	after, _ := r.Get(ctx, id)
	if after.Name == "Unsaved" || after.Version != latest.Version {
		t.Fatal("preview saved content")
	}
	latest, err = r.Status(ctx, id, latest.Version, "offline")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Public(ctx, latest.Slug); err == nil {
		t.Fatal("offline exposed")
	}
}
