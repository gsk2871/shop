package catalog

import (
	"context"
	"fmt"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/catalog"
	"peptide-site/internal/testutil"
	"strconv"
	"testing"
)

func TestHomeFeaturedProducts(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"product.publish", "product.price.update"}})
	image := fixtureMedia(t, r, "image")
	category, err := r.SaveCategory(ctx, 0, biz.Category{Name: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	price := "10.00"
	saved := []biz.Product{}
	for i := 0; i < 10; i++ {
		status := "published"
		if i == 8 {
			status = "draft"
		}
		if i == 9 {
			status = "offline"
		}
		p, err := r.Save(ctx, 0, biz.Product{Name: fmt.Sprintf("Home %d", i), Slug: fmt.Sprintf("home-%d", i), ImageMediaID: image, CategoryID: category.ID, Status: status, HomeFeatured: i != 7, HomeSort: 10 - i/2, SKUs: []biz.SKU{{Key: "default", Enabled: true, Currency: "USD", SalePrice: &price, SpecValueKeys: []string{}}}})
		if err != nil {
			t.Fatal(err)
		}
		if p.HomeFeatured != (i != 7) || p.HomeSort != 10-i/2 {
			t.Fatal("home settings did not round-trip", p)
		}
		saved = append(saved, p)
	}
	featured := true
	rows, total, err := r.List(ctx, true, 1, 6, "", 0, "", &featured)
	if err != nil || total != 7 || len(rows) != 6 {
		t.Fatalf("list: %v total=%d rows=%d", err, total, len(rows))
	}
	for i, index := range []int{6, 4, 5, 2, 3, 0} {
		if rows[i]["id"] != saved[index].ID {
			t.Fatalf("wrong order at %d: %v", i, rows)
		}
		if _, ok := rows[i]["homeFeatured"]; ok {
			t.Fatal("management field exposed")
		}
	}
	p := saved[6]
	id, _ := strconv.ParseInt(p.ID, 10, 64)
	p.ExpectedVersion = p.Version
	p.HomeFeatured = false
	p, err = r.Save(ctx, id, p)
	if err != nil || p.HomeFeatured {
		t.Fatalf("disable: %v", err)
	}
	rows, total, err = r.List(ctx, true, 1, 6, "", 0, "", &featured)
	if err != nil || total != 6 || rows[0]["id"] != saved[4].ID {
		t.Fatal("disabled recommendation visible", err)
	}
	id, _ = strconv.ParseInt(saved[4].ID, 10, 64)
	if _, err = r.Status(ctx, id, saved[4].Version, "offline"); err != nil {
		t.Fatal(err)
	}
	_, total, err = r.List(ctx, true, 1, 6, "", 0, "", &featured)
	if err != nil || total != 5 {
		t.Fatal("offline recommendation visible", err, total)
	}
	featured = false
	rows, total, err = r.List(ctx, false, 1, 20, "", 0, "", &featured)
	if err != nil || total != 2 {
		t.Fatal("admin filter failed", err, total)
	}
	for _, row := range rows {
		if row["homeFeatured"] != false {
			t.Fatal("admin recommendation state missing", row)
		}
	}
}
