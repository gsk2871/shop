// Package catalog implements product aggregate transactions and public DTOs.
package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/catalog"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	media "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/money"
	"peptide-site/internal/pkg/richtext"

	"strconv"
	"time"
)

type Repo struct {
	Data   *data.Data
	Config conf.Config
	Media  *media.Repo
}

func NewRepo(d *data.Data, c conf.Config) *Repo { return &Repo{d, c, media.NewRepo(d)} }

type Currency struct {
	Code      string `json:"code"`
	MinorUnit int    `json:"minorUnit"`
	Symbol    string `json:"symbol"`
}

func (r *Repo) Currencies(ctx context.Context) ([]Currency, error) {
	rows := []Currency{}
	err := r.Data.DB.WithContext(ctx).Table("currency").Where("enabled").Order("code").Find(&rows).Error
	return rows, err
}
func currency(tx *gorm.DB, code string) (Currency, error) {
	var c Currency
	err := tx.Table("currency").Where("code=? AND enabled", code).Take(&c).Error
	return c, err
}
func (r *Repo) Sanitize(ctx context.Context, d richtext.Document) (richtext.Document, error) {
	return richtext.Sanitize(ctx, d, func(ctx context.Context, id int64) (richtext.Media, error) {
		row, err := r.Media.Get(ctx, id)
		if err == gorm.ErrRecordNotFound {
			err = errdef.Invalid()
		}
		return richtext.Media{Type: row.MediaType, URL: row.URL}, err
	})
}

type productRow struct {
	CategoryID                                                                                                                                                 *int64
	ID                                                                                                                                                         int64
	Name, Slug, Category, Aliases, CAS, Formula, MolecularWeight, Sequence, Purity, Storage, Solubility, Summary, DetailHTML, Status, SEOTitle, SEODescription string
	Version                                                                                                                                                    int64
	UpdatedAt                                                                                                                                                  time.Time
}
type specRow struct {
	ID        int64
	Key, Name string
	Sort      int
}
type valueRow struct {
	ID, SpecID int64
	Key, Name  string
	Sort       int
}
type skuRow struct {
	ID            int64
	Key, Currency string
	Enabled       bool
	Sort          int
	Amount        *int64
	MinorUnit     int
	Symbol        string
}

func (r *Repo) get(tx *gorm.DB, id int64) (biz.Product, error) {
	var row productRow
	if err := tx.Table("product").Where("id=? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
		return biz.Product{}, err
	}
	p := biz.Product{ID: strconv.FormatInt(row.ID, 10), Name: row.Name, Slug: row.Slug, Category: row.Category, Aliases: row.Aliases, CAS: row.CAS, Formula: row.Formula, MolecularWeight: row.MolecularWeight, Sequence: row.Sequence, Purity: row.Purity, Storage: row.Storage, Solubility: row.Solubility, Summary: row.Summary, Status: row.Status, SEOTitle: row.SEOTitle, SEODescription: row.SEODescription, Version: row.Version, UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339), Specs: []biz.Spec{}, SKUs: []biz.SKU{}, Detail: richtext.Document{HTML: row.DetailHTML, MediaIDs: []string{}}}
	if row.CategoryID != nil {
		c, err := category(tx, *row.CategoryID, false)
		if err != nil {
			return p, err
		}
		p.CategoryID = strconv.FormatInt(c.ID, 10)
		p.Category = c.Name
	}
	var refs []int64
	if err := tx.Table("media_reference").Where("owner_type='product' AND owner_id=? AND field='detail'", p.ID).Order("media_id").Pluck("media_id", &refs).Error; err != nil {
		return p, err
	}
	for _, id := range refs {
		p.Detail.MediaIDs = append(p.Detail.MediaIDs, strconv.FormatInt(id, 10))
	}
	specs := []specRow{}
	values := []valueRow{}
	if err := tx.Table("product_spec").Where("product_id=? AND deleted_at IS NULL", id).Order("sort,id").Find(&specs).Error; err != nil {
		return p, err
	}
	if err := tx.Table("product_spec_value").Where("product_id=? AND deleted_at IS NULL", id).Order("sort,id").Find(&values).Error; err != nil {
		return p, err
	}
	for _, spec := range specs {
		item := biz.Spec{ID: strconv.FormatInt(spec.ID, 10), Key: spec.Key, Name: spec.Name, Sort: spec.Sort, Values: []biz.Value{}}
		for _, value := range values {
			if value.SpecID == spec.ID {
				item.Values = append(item.Values, biz.Value{ID: strconv.FormatInt(value.ID, 10), Key: value.Key, Name: value.Name, Sort: value.Sort})
			}
		}
		p.Specs = append(p.Specs, item)
	}
	skus := []skuRow{}
	if err := tx.Table("product_sku s").Select("s.id,s.key,s.currency,s.enabled,s.sort,p.amount,c.minor_unit,c.symbol").Joins("LEFT JOIN product_sku_price p ON p.sku_id=s.id AND p.currency=s.currency AND p.deleted_at IS NULL").Joins("JOIN currency c ON c.code=s.currency").Where("s.product_id=? AND s.deleted_at IS NULL", id).Order("s.sort,s.id").Scan(&skus).Error; err != nil {
		return p, err
	}
	var minimum *int64
	for _, sku := range skus {
		item := biz.SKU{ID: strconv.FormatInt(sku.ID, 10), Key: sku.Key, Currency: sku.Currency, Enabled: sku.Enabled, Sort: sku.Sort, SpecValueKeys: []string{}}
		if err := tx.Table("product_sku_value sv").Select("v.key").Joins("JOIN product_spec_value v ON v.id=sv.spec_value_id").Where("sv.sku_id=? AND v.deleted_at IS NULL", sku.ID).Order("v.sort,v.id").Scan(&item.SpecValueKeys).Error; err != nil {
			return p, err
		}
		if sku.Amount != nil {
			formatted, err := money.Format(*sku.Amount, sku.MinorUnit)
			if err != nil {
				return p, err
			}
			item.SalePrice = &formatted
			item.Price = &money.Price{Currency: sku.Currency, SalePrice: formatted, DisplayPrice: sku.Currency + " " + formatted}
			if sku.Enabled && (minimum == nil || *sku.Amount < *minimum) {
				amount := *sku.Amount
				minimum = &amount
				price := *item.Price
				p.FromPrice = &price
			}
		}
		p.SKUs = append(p.SKUs, item)
	}
	return p, nil
}
func (r *Repo) Get(ctx context.Context, id int64) (biz.Product, error) {
	var p biz.Product
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { var err error; p, err = r.get(tx, id); return err }, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return p, err
}
func (r *Repo) Public(ctx context.Context, slug string) (map[string]any, error) {
	var output map[string]any
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var id int64
		err := tx.Table("product p").Select("p.id").Where("p.slug=? AND p.status='published' AND p.deleted_at IS NULL", slug).Where("EXISTS(SELECT 1 FROM product_sku s JOIN product_sku_price sp ON sp.sku_id=s.id AND sp.currency=s.currency AND sp.deleted_at IS NULL WHERE s.product_id=p.id AND s.enabled AND s.deleted_at IS NULL)").Take(&id).Error
		if err != nil {
			return err
		}
		p, err := r.get(tx, id)
		if err != nil {
			return err
		}
		output = biz.Public(p)
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return output, err
}
func (r *Repo) List(ctx context.Context, public bool, page, size int, search string, categoryID int64) ([]map[string]any, int64, error) {
	output := []map[string]any{}
	var total int64
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows := []struct{ ID int64 }{}
		q := tx.Table("product p").Where("p.deleted_at IS NULL")
		if public {
			q = q.Where("p.status='published'").Where("EXISTS(SELECT 1 FROM product_sku s JOIN product_sku_price sp ON sp.sku_id=s.id AND sp.currency=s.currency AND sp.deleted_at IS NULL WHERE s.product_id=p.id AND s.enabled AND s.deleted_at IS NULL)")
		}
		if search != "" {
			q = q.Where("strpos(lower(p.name),lower(?))>0", search)
		}
		if categoryID != 0 {
			q = q.Where("p.category_id=?", categoryID)
		}
		if err := q.Count(&total).Error; err != nil {
			return err
		}
		if err := q.Select("p.id").Order("p.updated_at DESC,p.id DESC").Offset((page - 1) * size).Limit(size).Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			p, err := r.get(tx, row.ID)
			if err != nil {
				return err
			}
			item := map[string]any{"id": p.ID, "name": p.Name, "slug": p.Slug, "category": p.Category, "categoryId": p.CategoryID, "summary": p.Summary, "fromPrice": p.FromPrice}
			if !public {
				item["status"] = p.Status
				item["version"] = p.Version
				item["updatedAt"] = p.UpdatedAt
			}
			output = append(output, item)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return output, total, err
}
func actor(ctx context.Context) int64 {
	if i := auth.Current(ctx); i != nil {
		return i.User.ID
	}
	return -1
}
func audit(tx *gorm.DB, ctx context.Context, action string, id int64) error {
	return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,?,'product',?::bigint::text)", actor(ctx), action, id).Error
}
func can(ctx context.Context, p string) bool { i := auth.Current(ctx); return i != nil && i.Allows(p) }
func (r *Repo) Prepare(ctx context.Context, p *biz.Product) error {
	rows, err := r.Currencies(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return errdef.New(503, "SERVICE_UNAVAILABLE")
	}
	if err := biz.Validate(p, rows[0].Code); err != nil {
		return err
	}
	categoryID, _ := biz.ParseID(p.CategoryID)
	c, err := category(r.Data.DB.WithContext(ctx), categoryID, false)
	if err == gorm.ErrRecordNotFound {
		return errdef.Invalid()
	}
	if err != nil {
		return err
	}
	p.Category = c.Name
	clean, err := r.Sanitize(ctx, p.Detail)
	if err != nil {
		return err
	}
	p.Detail = clean
	p.FromPrice = nil
	var minimum *int64
	common := ""
	for j := range p.SKUs {
		sku := &p.SKUs[j]
		if common == "" {
			common = sku.Currency
		}
		if sku.Currency != common {
			return errdef.Invalid()
		}
		c, err := currency(r.Data.DB.WithContext(ctx), sku.Currency)
		if err != nil {
			return errdef.Invalid()
		}
		sku.Price = nil
		if sku.SalePrice != nil {
			amount, err := money.Parse(*sku.SalePrice, c.MinorUnit)
			if err != nil {
				return errdef.Invalid()
			}
			formatted, _ := money.Format(amount, c.MinorUnit)
			sku.SalePrice = &formatted
			sku.Price = &money.Price{Currency: sku.Currency, SalePrice: formatted, DisplayPrice: sku.Currency + " " + formatted}
			if sku.Enabled && (minimum == nil || amount < *minimum) {
				minimum = &amount
				price := *sku.Price
				p.FromPrice = &price
			}
		}
	}
	if p.Status == "published" && minimum == nil {
		return errdef.Invalid()
	}
	return nil
}
func pricesDiffer(old, new biz.Product) bool {
	prior := map[string]biz.SKU{}
	for _, sku := range old.SKUs {
		prior[sku.ID] = sku
	}
	for _, sku := range new.SKUs {
		existing, ok := prior[sku.ID]
		if !ok {
			if sku.SalePrice != nil {
				return true
			}
			continue
		}
		if existing.Currency != sku.Currency || (existing.SalePrice == nil) != (sku.SalePrice == nil) || (existing.SalePrice != nil && sku.SalePrice != nil && *existing.SalePrice != *sku.SalePrice) {
			return true
		}
		delete(prior, sku.ID)
	}
	for _, sku := range prior {
		if sku.SalePrice != nil {
			return true
		}
	}
	return false
}
func validateOwnership(old, new biz.Product) error {
	specs, values, skus := map[string]bool{}, map[string]string{}, map[string]bool{}
	for _, s := range old.Specs {
		specs[s.ID] = true
		for _, v := range s.Values {
			values[v.ID] = s.ID
		}
	}
	for _, s := range old.SKUs {
		skus[s.ID] = true
	}
	seen := map[string]bool{}
	for _, s := range new.Specs {
		if s.ID != "" {
			if !specs[s.ID] || seen["s"+s.ID] {
				return errdef.Invalid()
			}
			seen["s"+s.ID] = true
		}
		for _, v := range s.Values {
			if v.ID != "" {
				if values[v.ID] != s.ID || seen["v"+v.ID] {
					return errdef.Invalid()
				}
				seen["v"+v.ID] = true
			}
		}
	}
	for _, s := range new.SKUs {
		if s.ID != "" {
			if !skus[s.ID] || seen["k"+s.ID] {
				return errdef.Invalid()
			}
			seen["k"+s.ID] = true
		}
	}
	return nil
}
func (r *Repo) Save(ctx context.Context, id int64, p biz.Product) (biz.Product, error) {
	if err := r.Prepare(ctx, &p); err != nil {
		return p, err
	}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		categoryID, _ := biz.ParseID(p.CategoryID)
		c, err := category(tx, categoryID, true)
		if err == gorm.ErrRecordNotFound {
			return errdef.Invalid()
		}
		if err != nil {
			return err
		}
		p.Category = c.Name
		old := biz.Product{}
		if id != 0 {
			var row productRow
			if err := tx.Table("product").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
				return err
			}
			if p.ExpectedVersion != row.Version {
				return errdef.New(409, "VERSION_CONFLICT")
			}
			var err error
			old, err = r.get(tx, id)
			if err != nil {
				return err
			}
		} else if p.ExpectedVersion != 0 || p.ID != "" {
			return errdef.Invalid()
		}
		if err := validateOwnership(old, p); err != nil {
			return err
		}
		if p.Status != old.Status && p.Status != "draft" && !can(ctx, "product.publish") {
			return errdef.Denied()
		}
		if id != 0 && p.Status != old.Status && !can(ctx, "product.publish") {
			return errdef.Denied()
		}
		if pricesDiffer(old, p) && !can(ctx, "product.price.update") {
			return errdef.Denied()
		}
		columns := map[string]any{"name": p.Name, "slug": p.Slug, "category": p.Category, "category_id": categoryID, "aliases": p.Aliases, "cas": p.CAS, "formula": p.Formula, "molecular_weight": p.MolecularWeight, "sequence": p.Sequence, "purity": p.Purity, "storage": p.Storage, "solubility": p.Solubility, "summary": p.Summary, "detail_html": p.Detail.HTML, "status": p.Status, "seo_title": p.SEOTitle, "seo_description": p.SEODescription, "updated_at": time.Now().UTC(), "updated_by": actor(ctx)}
		if id == 0 {
			columns["created_by"] = actor(ctx)
			columns["version"] = 1
			if err := tx.Table("product").Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&columns).Error; err != nil {
				return err
			}
			value := columns["id"]
			switch v := value.(type) {
			case int64:
				id = v
			case int:
				id = int64(v)
			default:
				parsed, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
				if err != nil {
					return err
				}
				id = parsed
			}
		} else {
			columns["version"] = old.Version + 1
			if err := tx.Table("product").Where("id=?", id).Updates(columns).Error; err != nil {
				return err
			}
		}
		for _, table := range []string{"product_spec", "product_spec_value", "product_sku"} {
			if err := tx.Exec("UPDATE "+table+" SET deleted_at=now(),updated_at=now() WHERE product_id=? AND deleted_at IS NULL", id).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("UPDATE product_sku_price SET deleted_at=now(),updated_at=now() WHERE sku_id IN (SELECT id FROM product_sku WHERE product_id=?) AND deleted_at IS NULL", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM product_sku_value WHERE sku_id IN (SELECT id FROM product_sku WHERE product_id=?)", id).Error; err != nil {
			return err
		}
		valueIDs := map[string]int64{}
		for _, spec := range p.Specs {
			specID, _ := biz.ParseID(spec.ID)
			if specID == 0 {
				if err := tx.Raw("INSERT INTO product_spec(product_id,key,name,sort,created_by,updated_by) VALUES(?,?,?,?,?,?) RETURNING id", id, spec.Key, spec.Name, spec.Sort, actor(ctx), actor(ctx)).Scan(&specID).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Exec("UPDATE product_spec SET key=?,name=?,sort=?,deleted_at=NULL,updated_at=now(),updated_by=? WHERE id=?", spec.Key, spec.Name, spec.Sort, actor(ctx), specID).Error; err != nil {
					return err
				}
			}
			for _, value := range spec.Values {
				valueID, _ := biz.ParseID(value.ID)
				if valueID == 0 {
					if err := tx.Raw("INSERT INTO product_spec_value(product_id,spec_id,key,name,sort,created_by,updated_by) VALUES(?,?,?,?,?,?,?) RETURNING id", id, specID, value.Key, value.Name, value.Sort, actor(ctx), actor(ctx)).Scan(&valueID).Error; err != nil {
						return err
					}
				} else {
					if err := tx.Exec("UPDATE product_spec_value SET key=?,name=?,sort=?,deleted_at=NULL,updated_at=now(),updated_by=? WHERE id=?", value.Key, value.Name, value.Sort, actor(ctx), valueID).Error; err != nil {
						return err
					}
				}
				valueIDs[value.Key] = valueID
			}
		}
		for _, sku := range p.SKUs {
			skuID, _ := biz.ParseID(sku.ID)
			if skuID == 0 {
				if err := tx.Raw("INSERT INTO product_sku(product_id,key,currency,enabled,sort,created_by,updated_by) VALUES(?,?,?,?,?,?,?) RETURNING id", id, sku.Key, sku.Currency, sku.Enabled, sku.Sort, actor(ctx), actor(ctx)).Scan(&skuID).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Exec("UPDATE product_sku SET key=?,currency=?,enabled=?,sort=?,deleted_at=NULL,updated_at=now(),updated_by=? WHERE id=?", sku.Key, sku.Currency, sku.Enabled, sku.Sort, actor(ctx), skuID).Error; err != nil {
					return err
				}
			}
			for _, valueKey := range sku.SpecValueKeys {
				if err := tx.Exec("INSERT INTO product_sku_value(sku_id,spec_value_id) VALUES(?,?)", skuID, valueIDs[valueKey]).Error; err != nil {
					return err
				}
			}
			if sku.SalePrice != nil {
				c, err := currency(tx, sku.Currency)
				if err != nil {
					return err
				}
				amount, err := money.Parse(*sku.SalePrice, c.MinorUnit)
				if err != nil {
					return err
				}
				if err := tx.Exec("INSERT INTO product_sku_price(sku_id,currency,amount,created_by,updated_by) VALUES(?,?,?,?,?)", skuID, sku.Currency, amount, actor(ctx), actor(ctx)).Error; err != nil {
					return err
				}
			}
		}
		if err := media.SyncReferences(tx, "product", strconv.FormatInt(id, 10), map[string][]int64{"detail": richtext.IDs(p.Detail)}); err != nil {
			return err
		}
		action := "product.update"
		if old.ID == "" {
			action = "product.create"
		}
		return audit(tx, ctx, action, id)
	})
	if err != nil {
		return p, err
	}
	return r.Get(ctx, id)
}
func (r *Repo) Preview(ctx context.Context, p biz.Product) (map[string]any, error) {
	p.ExpectedVersion = 0
	p.ID = ""
	if err := r.Prepare(ctx, &p); err != nil {
		return nil, err
	}
	return biz.Public(p), nil
}
func (r *Repo) Status(ctx context.Context, id, version int64, status string) (biz.Product, error) {
	if status != "draft" && status != "published" && status != "offline" {
		return biz.Product{}, errdef.Invalid()
	}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row productRow
		if err := tx.Table("product").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		p, err := r.get(tx, id)
		if err != nil {
			return err
		}
		if status == "published" && p.FromPrice == nil {
			return errdef.Invalid()
		}
		if err := tx.Exec("UPDATE product SET status=?,version=version+1,updated_at=now(),updated_by=? WHERE id=?", status, actor(ctx), id).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "product."+status, id)
	})
	if err != nil {
		return biz.Product{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repo) Delete(ctx context.Context, id, version int64) error {
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row productRow
		if err := tx.Table("product").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		if err := tx.Exec("UPDATE product SET deleted_at=now(),version=version+1,updated_at=now(),updated_by=? WHERE id=?", actor(ctx), id).Error; err != nil {
			return err
		}
		if err := media.SyncReferences(tx, "product", strconv.FormatInt(id, 10), nil); err != nil {
			return err
		}
		return audit(tx, ctx, "product.delete", id)
	})
}
func (r *Repo) Price(ctx context.Context, id, skuID, version int64, value, code string) (biz.Product, error) {
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row productRow
		if err := tx.Table("product").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		var sku skuRow
		if err := tx.Table("product_sku").Where("id=? AND product_id=? AND deleted_at IS NULL", skuID, id).Take(&sku).Error; err != nil {
			return errdef.Invalid()
		}
		if code != sku.Currency {
			return errdef.Invalid()
		}
		c, err := currency(tx, code)
		if err != nil {
			return errdef.Invalid()
		}
		amount, err := money.Parse(value, c.MinorUnit)
		if err != nil {
			return errdef.Invalid()
		}
		if err := tx.Exec("INSERT INTO product_sku_price(sku_id,currency,amount,created_by,updated_by) VALUES(?,?,?,?,?) ON CONFLICT(sku_id,currency) WHERE deleted_at IS NULL DO UPDATE SET amount=EXCLUDED.amount,updated_at=now(),updated_by=EXCLUDED.updated_by", skuID, code, amount, actor(ctx), actor(ctx)).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE product SET version=version+1,updated_at=now(),updated_by=? WHERE id=?", actor(ctx), id).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "product.price.update", id)
	})
	if err != nil {
		return biz.Product{}, err
	}
	return r.Get(ctx, id)
}
