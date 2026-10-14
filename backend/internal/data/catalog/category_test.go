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

func TestCategoriesAndProductFilters(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	imageID := fixtureMedia(t, r, "image")
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"product.publish", "product.price.update"}})
	first, err := r.SaveCategory(ctx, 0, biz.Category{Name: " Compounds ", Sort: 2})
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Supplies", Sort: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.SaveCategory(ctx, 0, biz.Category{Name: "COMPOUNDS"}); err == nil {
		t.Fatal("duplicate category accepted")
	}
	categories, total, err := r.Categories(ctx, 1, 20, "", true)
	if err != nil || total != 2 || categories[0].ID != second.ID || first.Name != "Compounds" {
		t.Fatal(categories, total, err)
	}
	categories, total, err = r.Categories(ctx, 1, 20, "POUND", false)
	if err != nil || total != 1 || categories[0].ID != first.ID {
		t.Fatal(categories, total, err)
	}
	price := "12.00"
	makeProduct := func(name, slug, category string) biz.Product {
		p, err := r.Save(ctx, 0, biz.Product{ImageMediaID: imageID, Name: name, Slug: slug, CategoryID: category, Status: "draft", Detail: richtext.Document{MediaIDs: []string{}}, SKUs: []biz.SKU{{Key: "default", Enabled: true, Currency: "USD", SalePrice: &price, SpecValueKeys: []string{}}}})
		if err != nil {
			t.Fatal(err)
		}
		id, _ := strconv.ParseInt(p.ID, 10, 64)
		p, err = r.Status(ctx, id, p.Version, "published")
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	p1 := makeProduct("Alpha 100%", "alpha", first.ID)
	makeProduct("Alpha supply", "alpha-supply", second.ID)
	makeProduct("Other product", "other", second.ID)
	id, _ := strconv.ParseInt(first.ID, 10, 64)
	for _, tt := range []struct {
		q    string
		cat  int64
		want int64
	}{{"", 0, 3}, {"", id, 1}, {"ALPHA", 0, 2}, {"alpha", id, 1}, {"hidden-alias", 0, 0}, {"Compounds", 0, 0}, {"%", 0, 1}, {"supply", id, 0}, {"", 999999, 0}} {
		rows, n, err := r.List(ctx, true, 1, 20, tt.q, tt.cat, "", nil)
		if err != nil || n != tt.want || int64(len(rows)) != tt.want {
			t.Fatalf("filter %+v: %v %d %v", tt, rows, n, err)
		}
	}
	for _, tt := range []struct {
		status string
		public bool
		want   int64
	}{
		{"draft", false, 0}, {"published", false, 3}, {"offline", false, 0}, {"draft", true, 3},
	} {
		rows, total, err := r.List(ctx, tt.public, 1, 1, "", 0, tt.status, nil)
		if err != nil || total != tt.want || len(rows) != min(1, int(tt.want)) {
			t.Fatal(tt, rows, total, err)
		}
	}

	if err = r.DeleteCategory(ctx, id, first.Version); err == nil {
		t.Fatal("used category deleted")
	}
	first.ExpectedVersion = first.Version
	first.Name = "Renamed"
	renamed, err := r.SaveCategory(ctx, id, first)
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.ParseInt(p1.ID, 10, 64)
	current, err := r.Get(ctx, pid)
	if err != nil || current.Category != "Renamed" {
		t.Fatal(current, err)
	}
	for _, public := range []bool{false, true} {
		rows, total, err := r.List(ctx, public, 1, 20, "", id, "", nil)
		if err != nil || total != 1 || len(rows) != 1 || rows[0]["category"] != "Renamed" {
			t.Fatal("list category must use current related name", rows, total, err)
		}
	}
	if _, err = r.SaveCategory(ctx, id, first); err == nil {
		t.Fatal("stale category accepted")
	}
	if err = r.Delete(ctx, pid, current.Version); err != nil {
		t.Fatal(err)
	}
	if err = r.DeleteCategory(ctx, id, renamed.Version); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Category(ctx, id); err == nil {
		t.Fatal("deleted category returned")
	}
	current.ExpectedVersion = 0
	current.ID = ""
	current.Slug = "invalid"
	current.Status = "draft"
	if _, err = r.Save(ctx, 0, current); err == nil {
		t.Fatal("deleted category selected")
	}
	current.CategoryID = ""
	if _, err = r.Save(ctx, 0, current); err == nil {
		t.Fatal("missing category accepted")
	}
}
func TestCategoryDeleteRacesProductCreation(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	imageID := fixtureMedia(t, r, "image")
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}})
	cat, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Race"})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := strconv.ParseInt(cat.ID, 10, 64)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	go func() { defer wg.Done(); <-start; errs <- r.DeleteCategory(ctx, id, cat.Version) }()
	go func() {
		defer wg.Done()
		<-start
		_, err := r.Save(ctx, 0, biz.Product{ImageMediaID: imageID, Name: "Race", Slug: "race", CategoryID: cat.ID, Status: "draft", Detail: richtext.Document{MediaIDs: []string{}}, SKUs: []biz.SKU{{Key: "default", Enabled: false, Currency: "USD", SpecValueKeys: []string{}}}})
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("creation/deletion must have exactly one winner", success)
	}
	var orphan int64
	if err = d.DB.Raw("SELECT count(*) FROM product p JOIN product_category c ON c.id=p.category_id WHERE p.deleted_at IS NULL AND c.deleted_at IS NOT NULL").Scan(&orphan).Error; err != nil || orphan != 0 {
		t.Fatal("orphan", orphan, err)
	}
}
func TestCurrencyDictionaryDrivesOptionsAndDefaultSKU(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	imageID := fixtureMedia(t, r, "image")
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}})
	cat, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Currency"})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.DB.Exec("INSERT INTO currency(code,minor_unit,symbol,enabled) VALUES ('EUR',2,'€',true),('JPY',0,'¥',true),('GBP',2,'£',false)").Error; err != nil {
		t.Fatal(err)
	}
	rows, err := r.Currencies(ctx)
	if err != nil || len(rows) != 3 || rows[0].Code != "EUR" || rows[2].Code != "USD" {
		t.Fatal("enabled dictionary options", rows, err)
	}
	product := biz.Product{ImageMediaID: imageID, Name: "Default", Slug: "default", CategoryID: cat.ID, Status: "draft", Detail: richtext.Document{MediaIDs: []string{}}}
	if err = r.Prepare(ctx, &product); err != nil || len(product.SKUs) != 1 || product.SKUs[0].Currency != "EUR" {
		t.Fatal("dictionary default SKU", product, err)
	}
	product.SKUs[0].Currency = "JPY"
	if _, err = r.Save(ctx, 0, product); err != nil {
		t.Fatal("enabled currency rejected", err)
	}
	product.Slug = "disabled"
	product.SKUs[0].Currency = "GBP"
	if _, err = r.Save(ctx, 0, product); err == nil {
		t.Fatal("disabled currency accepted")
	}
}
