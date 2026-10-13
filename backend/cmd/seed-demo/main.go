// seed-demo adds isolated, repeatable fixtures to a local development database.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
	auth "peptide-site/internal/biz/admin"
	bb "peptide-site/internal/biz/branding"
	cb "peptide-site/internal/biz/catalog"
	fb "peptide-site/internal/biz/faq"
	mb "peptide-site/internal/biz/media"
	nb "peptide-site/internal/biz/news"
	wb "peptide-site/internal/biz/whatsapp"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	br "peptide-site/internal/data/branding"
	cr "peptide-site/internal/data/catalog"
	fr "peptide-site/internal/data/faq"
	mr "peptide-site/internal/data/media"
	nr "peptide-site/internal/data/news"
	wr "peptide-site/internal/data/whatsapp"
	"peptide-site/internal/pkg/richtext"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	apply := flag.Bool("apply", false, "Insert local demonstration data; default only inspects")
	flag.Parse()
	c, err := conf.LoadFile("configs/config.local.yaml")
	must(err)
	pg, err := pgx.ParseConfig(c.Data.Database.DSN)
	must(err)
	if pg.Host != "127.0.0.1" && pg.Host != "localhost" && pg.Host != "::1" {
		panic("demo seeding requires a loopback database host")
	}
	fmt.Printf("Local database: %s:%d/%s\n", pg.Host, pg.Port, pg.Database)
	d, closeDB, err := data.NewData(c)
	must(err)
	defer closeDB()
	for _, table := range []string{"media", "product_category", "product", "article", "banner", "faq", "whatsapp_group"} {
		var count int64
		must(d.DB.Table(table).Where("deleted_at IS NULL").Count(&count).Error)
		fmt.Printf("%s: %d existing rows\n", table, count)
	}
	if !*apply {
		return
	}
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: -1}, Permissions: []string{"product.publish", "product.price.update", "article.publish", "banner.publish"}})
	var storage struct {
		ID   string
		Path string
	}
	must(d.DB.Table("storage_root").Where("path=?", absolute(c.Data.Storage.UploadDir)).Take(&storage).Error)
	createdFiles := []string{}
	counts := map[string]int{}
	err = d.DB.Transaction(func(tx *gorm.DB) error {
		local := &data.Data{DB: tx}
		catalog := cr.NewRepo(local, c)
		branding := br.NewRepo(local)
		news := nr.NewRepo(local, c)
		faqs := fr.NewRepo(local)
		media := mr.NewRepo(local)
		whatsapp := wr.NewRepo(local)
		exists := func(table, field, value string) int64 {
			var id int64
			must(tx.Table(table).Select("id").Where(field+"=? AND deleted_at IS NULL", value).Limit(1).Scan(&id).Error)
			return id
		}
		assets := []string{}
		for i := 0; i < 6; i++ {
			name := fmt.Sprintf("[TEST] Demo image %02d.png", i+1)
			id := exists("media", "name", name)
			if id == 0 {
				key := fmt.Sprintf("demo-operations-%d-%d.png", time.Now().UnixNano(), i)
				path := filepath.Join(storage.Path, key)
				file, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if e != nil {
					return e
				}
				createdFiles = append(createdFiles, path)
				img := image.NewRGBA(image.Rect(0, 0, 800, 600))
				draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{uint8(224 + i*3), 239, uint8(230 - i*4), 255}}, image.Point{}, draw.Src)
				draw.Draw(img, image.Rect(290, 125, 510, 495), &image.Uniform{color.RGBA{250, 250, 248, 255}}, image.Point{}, draw.Src)
				draw.Draw(img, image.Rect(290, 105, 510, 160), &image.Uniform{color.RGBA{uint8(35 + i*20), 95, 90, 255}}, image.Point{}, draw.Src)
				draw.Draw(img, image.Rect(305, 240, 495, 350), &image.Uniform{color.RGBA{uint8(35 + i*20), 95, 90, 255}}, image.Point{}, draw.Src)
				e = png.Encode(file, img)
				closeErr := file.Close()
				if e != nil {
					return e
				}
				if closeErr != nil {
					return closeErr
				}
				stat, e := os.Stat(path)
				if e != nil {
					return e
				}
				w, h := 800, 600
				row := mb.Media{Name: name, URL: "/uploads/" + storage.ID + "/" + key, MIME: "image/png", MediaType: "image", Size: stat.Size(), Width: &w, Height: &h, Available: true, StorageRootID: storage.ID, StorageKey: key, CreatedAt: time.Now(), UpdatedAt: time.Now()}
				if e = media.Save(ctx, &row); e != nil {
					return e
				}
				id = row.ID
				counts["media"]++
			}
			assets = append(assets, strconv.FormatInt(id, 10))
		}
		for i, entry := range []struct{ title, subtitle, cta, link string }{
			{"Explore the demonstration catalog", "Browse sample products and packaging options. All displayed items are fictional test content.", "Explore products", "/products"},
			{"Discover our latest sample news", "Read demonstration articles and explore the website's news section.", "Read news", "/news"},
		} {
			title := "[TEST] " + entry.title
			if exists("banner", "title", title) != 0 {
				continue
			}
			if _, e := branding.Save(ctx, 0, bb.Banner{
				SlotCode: "home_hero", Title: title, Subtitle: entry.subtitle,
				ImageMediaID: assets[i], CTAText: entry.cta, Link: entry.link,
				Sort: (i + 1) * 10, Enabled: true,
			}); e != nil {
				return fmt.Errorf("home banner %d: %w", i+1, e)
			}
			counts["banners"]++
		}
		categories := []string{}
		for i, name := range []string{"[TEST] Research Samples", "[TEST] Reference Materials", "[TEST] Laboratory Supplies"} {
			id := exists("product_category", "name", name)
			if id == 0 {
				row, e := catalog.SaveCategory(ctx, 0, cb.Category{Name: name, Sort: (i + 1) * 10})
				if e != nil {
					return e
				}
				id, _ = strconv.ParseInt(row.ID, 10, 64)
				counts["categories"]++
			}
			categories = append(categories, strconv.FormatInt(id, 10))
		}
		for i := 0; i < 9; i++ {
			slug := fmt.Sprintf("test-demo-product-%02d", i+1)
			if exists("product", "slug", slug) != 0 {
				continue
			}
			status := []string{"published", "published", "draft", "offline"}[i%4]
			price := fmt.Sprintf("%d.00", 25+i*10)
			large := fmt.Sprintf("%d.50", 60+i*10)
			p := cb.Product{Name: fmt.Sprintf("[TEST] Research Sample %02d", i+1), Slug: slug, CategoryID: categories[i%3], ImageMediaID: assets[i%6], Status: status, Summary: "Demonstration product for interface testing. Not an actual item for sale.", Detail: richtext.Document{HTML: "<h2>Test product</h2><p>Fictional content for testing product details, specifications and prices.</p><ul><li>Sample packaging</li><li>Test data only</li></ul>", MediaIDs: []string{}}, Specs: []cb.Spec{{Key: "pack", Name: "Pack size", Values: []cb.Value{{Key: "small", Name: "Sample pack"}, {Key: "large", Name: "Large pack", Sort: 1}}}}, SKUs: []cb.SKU{{Key: "small", Enabled: true, SpecValueKeys: []string{"small"}, Currency: "USD", SalePrice: &price}, {Key: "large", Enabled: i%3 != 0, SpecValueKeys: []string{"large"}, Currency: "USD", SalePrice: &large}}}
			if _, e := catalog.Save(ctx, 0, p); e != nil {
				return fmt.Errorf("product %d: %w", i, e)
			}
			counts["products"]++
		}
		titles := []string{"Welcome to the demonstration catalog", "How to browse sample product specifications", "Understanding the sample document layout", "Demo packaging and label preview", "Editorial draft for review", "Archived demonstration announcement"}
		for i, title := range titles {
			slug := fmt.Sprintf("test-demo-news-%02d", i+1)
			if exists("article", "slug", slug) != 0 {
				continue
			}
			status := "published"
			if i == 4 {
				status = "draft"
			}
			if i == 5 {
				status = "offline"
			}
			a := nb.Article{Title: "[TEST] " + title, Slug: slug, Summary: "Sample news content for layout and publishing workflow verification.", CoverMediaID: assets[i], Status: status, Pinned: i == 0, Sort: i * 10, Body: richtext.Document{HTML: "<h2>Demonstration article</h2><p>This article contains fictional test content. It is intended for checking the news list, cover image and detail layout.</p><h3>Preview checklist</h3><ul><li>Cover and title display</li><li>Paragraph spacing</li><li>Published content visibility</li></ul>", MediaIDs: []string{}}}
			if _, e := news.Save(ctx, 0, a); e != nil {
				return e
			}
			counts["articles"]++
		}
		questions := [][2]string{{"What is this demonstration website?", "This is fictional test content used to verify the website layout and navigation."}, {"How can I browse the sample catalog?", "Open Products and choose a test category.\nYou can also search for a sample product by name."}, {"Are the displayed prices real quotations?", "No. All prices on TEST products are demonstration values."}, {"Where can I find sample product details?", "Select a published test product to view its details and packaging options."}, {"How do I find recent announcements?", "Open News to view published demonstration articles."}, {"How are these questions ordered?", "Questions are arranged by the configured sort value, with lower values appearing first."}, {"Disabled question for visibility testing", "This answer must only appear in the administration interface."}, {"Second disabled question", "Enable this test question to check the public display."}}
		for i, pair := range questions {
			question := "[TEST] " + pair[0]
			if exists("faq", "question", question) != 0 {
				continue
			}
			if _, e := faqs.SaveFAQ(ctx, 0, fb.FAQ{Question: question, Answer: pair[1], Sort: i * 10, Enabled: i < 6}); e != nil {
				return e
			}
			counts["faqs"]++
		}
		for i, country := range []string{"GB", "CA"} {
			name := "[TEST] Disabled support " + country
			if exists("whatsapp_group", "name", name) != 0 {
				continue
			}
			var assigned int64
			if e := tx.Table("whatsapp_group_country").Where("country_code=? AND deleted_at IS NULL", country).Count(&assigned).Error; e != nil {
				return e
			}
			if assigned > 0 {
				fmt.Printf("Skip support %s: country already assigned\n", country)
				continue
			}
			in := wb.Input{Name: name, CountryCodes: []string{country}, Enabled: false, Accounts: []wb.AccountInput{{Name: "[TEST] Placeholder - do not contact", PhoneNumber: fmt.Sprintf("120255501%02d", i), QRCodeMediaID: assets[5], Enabled: false}}}
			if _, e := whatsapp.Save(ctx, 0, in); e != nil {
				return e
			}
			counts["support groups"]++
		}
		return nil
	})
	if err != nil {
		for _, p := range createdFiles {
			_ = os.Remove(p)
		}
		must(err)
	}
	fmt.Printf("Created: %v\n", counts)
}
func absolute(path string) string { p, e := filepath.Abs(path); must(e); return p }
