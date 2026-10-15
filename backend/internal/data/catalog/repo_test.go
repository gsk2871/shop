package catalog

import (
	"context"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/catalog"
	"peptide-site/internal/pkg/richtext"
	"peptide-site/internal/testutil"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
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
	for _, key := range []string{"status", "version", "expectedVersion", "imageMediaIds", "reportMediaIds", "updatedBy"} {
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

func TestPriceRowsOnlyChangeWhenPriceChanges(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"product.price.update"}})
	category, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Prices"})
	if err != nil {
		t.Fatal(err)
	}
	value := "12.34"
	p, err := r.Save(ctx, 0, biz.Product{Name: "Prices", Status: "draft", CategoryID: category.ID, ImageMediaID: fixtureMedia(t, r, "image"), SKUs: []biz.SKU{{Key: "default", Enabled: true, Currency: "USD", SalePrice: &value}}})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := strconv.ParseInt(p.ID, 10, 64)
	type priceRow struct {
		ID, Amount           int64
		CreatedAt, UpdatedAt time.Time
	}
	read := func() priceRow {
		t.Helper()
		var row priceRow
		if err := d.DB.Table("product_sku_price").Where("sku_id=? AND deleted_at IS NULL", p.SKUs[0].ID).Take(&row).Error; err != nil {
			t.Fatal(err)
		}
		return row
	}
	counts := func(total, active int64) {
		t.Helper()
		var got struct{ Total, Active int64 }
		if err := d.DB.Raw("SELECT count(*) total,count(*) FILTER(WHERE deleted_at IS NULL) active FROM product_sku_price").Scan(&got).Error; err != nil || got.Total != total || got.Active != active {
			t.Fatal("price rows", got, err)
		}
	}
	save := func() {
		t.Helper()
		p.ExpectedVersion = p.Version
		p, err = r.Save(ctx, id, p)
		if err != nil {
			t.Fatal(err)
		}
	}
	original := read()
	p.Name = "Name only"
	save()
	counts(1, 1)
	if got := read(); got != original {
		t.Fatal("unchanged price was rewritten", original, got)
	}
	value = "15.00"
	p.SKUs[0].SalePrice = &value
	save()
	changed := read()
	if changed.ID != original.ID || changed.Amount != 1500 || changed.CreatedAt != original.CreatedAt {
		t.Fatal("changed price lost its identity", changed)
	}
	counts(1, 1)
	skuID, _ := strconv.ParseInt(p.SKUs[0].ID, 10, 64)
	p, err = r.Price(ctx, id, skuID, p.Version, "15.00", "USD")
	if err != nil || read() != changed {
		t.Fatal("same dedicated price changed metadata", err)
	}
	p.SKUs[0].SalePrice = nil
	save()
	counts(1, 0)
	p.SKUs[0].SalePrice = &value
	save()
	counts(2, 1)
	if err := d.DB.Exec("INSERT INTO currency(code,minor_unit,symbol) VALUES('EUR',2,'€')").Error; err != nil {
		t.Fatal(err)
	}
	p.SKUs[0].Currency = "EUR"
	save()
	counts(3, 1)
	var code string
	if err := d.DB.Table("product_sku_price").Select("currency").Where("deleted_at IS NULL").Scan(&code).Error; err != nil || code != "EUR" {
		t.Fatal("old currency remained active", code, err)
	}
	// Replacing a SKU retires the previous SKU's price even if the key is reused.
	p.SKUs[0].ID = ""
	save()
	counts(4, 1)
	var dangling int64
	if err := d.DB.Raw("SELECT count(*) FROM product_sku_price p JOIN product_sku s ON s.id=p.sku_id WHERE p.deleted_at IS NULL AND s.deleted_at IS NOT NULL").Scan(&dangling).Error; err != nil || dangling != 0 {
		t.Fatal("removed SKU retained active price", dangling, err)
	}
}

func TestProductRichTextSurvivesSavePreviewAndPublic(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"product.publish", "product.price.update"}})
	category, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Rich text"})
	if err != nil {
		t.Fatal(err)
	}
	image := fixtureMedia(t, r, "image")
	body := richtext.Document{HTML: `<p><strong>Rich content</strong></p><img data-media-id="` + image + `" src="/uploads/test/image">`, MediaIDs: []string{image}}
	price := "1.00"
	p, err := r.Save(ctx, 0, biz.Product{Name: "Rich text", Status: "published", CategoryID: category.ID, ImageMediaID: image, Detail: body, SKUs: []biz.SKU{{Key: "default", Enabled: true, Currency: "USD", SalePrice: &price}}})
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := d.DB.Table("product").Select("detail_html").Where("id=?", p.ID).Scan(&stored).Error; err != nil || stored != p.Detail.HTML || !strings.Contains(stored, "Rich content") {
		t.Fatal("body storage", stored, err)
	}
	public, err := r.Public(ctx, p.Slug)
	if err != nil || !reflect.DeepEqual(public["detail"], p.Detail) {
		t.Fatal("public body", public, err)
	}
	preview, err := r.Preview(ctx, p)
	if err != nil || !reflect.DeepEqual(preview["detail"], p.Detail) {
		t.Fatal("preview body", preview, err)
	}
	for _, key := range []string{"aliases", "cas", "formula", "molecularWeight", "sequence", "purity", "storage", "solubility", "seoTitle", "seoDescription", "bodyFormat"} {
		if _, exists := public[key]; exists {
			t.Fatal("removed field exposed", key)
		}
	}
}
