package data_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"peptide-site/internal/data"
	"peptide-site/internal/testutil"
)

func TestSchemaBaselineRoundTrip(t *testing.T) {
	d, _ := testutil.Postgres(t)
	if err := data.Migrate(d); err != nil {
		t.Fatal("repeat migrate", err)
	}
	var count int64
	if err := d.DB.Table("schema_migration").Count(&count).Error; err != nil || count != 6 {
		t.Fatal("baseline count", count, err)
	}
	var missing int64
	if err := d.DB.Raw(`SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped WHERE n.nspname='public' AND c.relkind='r' AND (col_description(c.oid,a.attnum) IS NULL OR obj_description(c.oid) IS NULL)`).Scan(&missing).Error; err != nil || missing != 0 {
		t.Fatal("missing comments", missing, err)
	}
	if err := d.DB.Raw(`SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name IN ('product','article','media','audit_log') AND column_name='id' AND is_identity='YES' AND identity_generation='ALWAYS'`).Scan(&count).Error; err != nil || count != 4 {
		t.Fatal("identity primary keys", count, err)
	}
	if err := d.DB.Raw(`SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND ((table_name IN ('product','article') AND column_name IN ('body_format','seo_title','seo_description')) OR (table_name='product' AND column_name IN ('category','aliases','cas','formula','molecular_weight','sequence','purity','storage','solubility')))`).Scan(&count).Error; err != nil || count != 0 {
		t.Fatal("removed columns", count, err)
	}
	var index string
	if err := d.DB.Raw("SELECT indexdef FROM pg_indexes WHERE schemaname='public' AND indexname='article_public_sort'").Scan(&index).Error; err != nil || !strings.Contains(index, "first_published_at DESC NULLS LAST") {
		t.Fatal("article sort index", index, err)
	}
	for _, name := range []string{"000006_banner_leads", "000005_social_account", "000004_category_cover", "000003_whatsapp_greeting", "000002_media_order"} {
		down, err := os.ReadFile("migration/" + name + ".down.sql")
		if err != nil {
			t.Fatal(err)
		}
		if err := d.DB.Exec(string(down)).Error; err != nil {
			t.Fatal(err)
		}
	}
	down, err := os.ReadFile("migration/000001_schema.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.DB.Exec(string(down)).Error; err != nil {
		t.Fatal("down", err)
	}
	if err := d.DB.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'").Scan(&count).Error; err != nil || count != 1 {
		t.Fatal("down retained business tables", count, err)
	}
	if err := d.DB.Exec("DELETE FROM schema_migration").Error; err != nil {
		t.Fatal(err)
	}
	if err := data.Migrate(d); err != nil {
		t.Fatal("up after down", err)
	}
	if err := d.DB.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'").Scan(&count).Error; err != nil || count != 38 {
		t.Fatal("rebuilt tables", count, err)
	}
}

func TestSchemaRejectsInvalidData(t *testing.T) {
	d, _ := testutil.Postgres(t)
	for _, tt := range []struct{ name, query, code string }{
		{"required category", `INSERT INTO product(name,slug,status) VALUES('test','test','draft')`, "23502"},
		{"site version", `INSERT INTO site_config(id,payload,version) VALUES(1,'{}',0)`, "23514"},
		{"banner version", `INSERT INTO banner(slot_code,title,image_media_id,created_by,updated_by,version) VALUES('home_hero','test',1,1,1,0)`, "23514"},
		{"media size", `INSERT INTO media(name,url,mime,size,storage_root_id,storage_key,media_type) VALUES('test','/uploads/test.png','image/png',0,'test','test','image')`, "23514"},
		{"media dimensions", `INSERT INTO media(name,url,mime,size,width,storage_root_id,storage_key,media_type) VALUES('test','/uploads/test.png','image/png',1,0,'test','test','image')`, "23514"},
		{"media ready URL", `INSERT INTO media(name,url,mime,size,storage_root_id,storage_key,media_type) VALUES('test','','image/png',1,'test','test','image')`, "23514"},
		{"COS batch", `INSERT INTO media(name,url,mime,size,storage_root_id,storage_key,media_type,storage_provider,status) VALUES('test','','image/png',1,'cos:test','test','image','cos','pending')`, "23514"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := d.DB.Exec(tt.query).Error
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != tt.code {
				t.Fatal("invalid data was not rejected as expected", err)
			}
		})
	}
	// Pending uploads legitimately have no public URL or dimensions yet.
	if err := d.DB.Exec(`INSERT INTO media(name,url,mime,size,storage_root_id,storage_key,media_type,storage_provider,status,batch_id,available) VALUES('test','','image/png',1,'cos:test','test','image','cos','pending','batch',false)`).Error; err != nil {
		t.Fatal("pending upload rejected", err)
	}
}
