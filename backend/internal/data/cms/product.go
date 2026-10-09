package cms

import (
	"context"
	"gorm.io/gorm"
	biz "peptide-site/internal/biz/cms"
	"time"
)

func (r *Repo) Products(ctx context.Context, public bool, search string) ([]biz.Product, error) {
	rows := []biz.Product{}
	q := r.db.WithContext(ctx).Table("product").Order("updated_at DESC, id DESC").Limit(200)
	if public {
		q = q.Where("status = ?", "published")
	}
	if search != "" {
		q = q.Where("strpos(lower(name || ' ' || aliases || ' ' || cas || ' ' || sequence || ' ' || category), lower(?)) > 0", search)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repo) Product(ctx context.Context, slug string) (biz.Product, error) {
	var p biz.Product
	err := r.db.WithContext(ctx).Table("product").Where("slug = ? AND status = ?", slug, "published").Take(&p).Error
	return p, err
}

func (r *Repo) SaveProduct(ctx context.Context, p *biz.Product) error {
	p.UpdatedAt = time.Now().UTC()
	if p.Images == nil {
		p.Images = []string{}
	}
	if p.ID == 0 {
		return r.db.WithContext(ctx).Table("product").Create(p).Error
	}
	res := r.db.WithContext(ctx).Table("product").Where("id = ?", p.ID).Select("name", "slug", "category", "aliases", "cas", "formula", "molecular_weight", "sequence", "purity", "specifications", "storage", "solubility", "summary", "description", "images", "status", "seo_title", "seo_description", "updated_at").Updates(p)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *Repo) DeleteProduct(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Exec("DELETE FROM product WHERE id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
