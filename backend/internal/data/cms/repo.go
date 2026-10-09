// Package cms implements the site's PostgreSQL repositories.
package cms

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	biz "peptide-site/internal/biz/cms"
	"peptide-site/internal/data"
	"time"
)

// Repo persists CMS data using explicit table names.
type Repo struct{ db *gorm.DB }

// NewRepo builds a repository from the shared connection pool.
func NewRepo(d *data.Data) *Repo { return &Repo{db: d.DB} }

var _ biz.Repository = (*Repo)(nil)

func (r *Repo) Settings(ctx context.Context) (biz.Settings, error) {
	var row struct{ Payload string }
	err := r.db.WithContext(ctx).Table("site_config").Where("id = ?", 1).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return biz.DefaultSettings(), nil
	}
	if err != nil {
		return biz.Settings{}, err
	}
	var s biz.Settings
	err = json.Unmarshal([]byte(row.Payload), &s)
	return s, err
}
func (r *Repo) SaveSettings(ctx context.Context, s biz.Settings) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Exec("INSERT INTO site_config(id,payload) VALUES(1,?::jsonb) ON CONFLICT(id) DO UPDATE SET payload=EXCLUDED.payload", string(b)).Error
}
func (r *Repo) Articles(ctx context.Context, public bool) ([]biz.Article, error) {
	rows := []biz.Article{}
	q := r.db.WithContext(ctx).Table("article").Order("updated_at DESC").Limit(200)
	if public {
		q = q.Where("status = ?", "published")
	}
	err := q.Find(&rows).Error
	return rows, err
}
func (r *Repo) Article(ctx context.Context, slug string) (biz.Article, error) {
	var a biz.Article
	err := r.db.WithContext(ctx).Table("article").Where("slug = ? AND status = ?", slug, "published").Take(&a).Error
	return a, err
}
func (r *Repo) SaveArticle(ctx context.Context, a *biz.Article) error {
	a.UpdatedAt = time.Now().UTC()
	if a.ID == 0 {
		return r.db.WithContext(ctx).Table("article").Create(a).Error
	}
	res := r.db.WithContext(ctx).Table("article").Where("id = ?", a.ID).Select("title", "slug", "summary", "body", "cover", "status", "seo_title", "seo_description", "updated_at").Updates(a)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *Repo) DeleteArticle(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Exec("DELETE FROM article WHERE id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *Repo) Media(ctx context.Context) ([]biz.Media, error) {
	m := []biz.Media{}
	err := r.db.WithContext(ctx).Table("media").Order("id DESC").Limit(200).Find(&m).Error
	return m, err
}
func (r *Repo) SaveMedia(ctx context.Context, m *biz.Media) error {
	m.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Table("media").Create(m).Error
}
func (r *Repo) Admin(ctx context.Context) (biz.Admin, error) {
	var a biz.Admin
	err := r.db.WithContext(ctx).Table("admin_account").Where("id = ?", 1).Take(&a).Error
	return a, err
}
func (r *Repo) Password(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE admin_account SET password_hash=? WHERE id=1", hash).Error; err != nil {
			return err
		}
		return tx.Exec("DELETE FROM admin_session").Error
	})
}
func (r *Repo) CreateSession(ctx context.Context, hash string, expiry time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM admin_session WHERE expires_at < now()").Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO admin_session(token_hash,expires_at) VALUES(?,?)", hash, expiry).Error
	})
}
func (r *Repo) Session(ctx context.Context, hash string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("admin_session").Where("token_hash = ? AND expires_at > now()", hash).Count(&n).Error
	return n > 0, err
}
func (r *Repo) DeleteSession(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM admin_session WHERE token_hash = ?", hash).Error
}
func (r *Repo) Audit(ctx context.Context, action string) error {
	return r.db.WithContext(ctx).Exec("INSERT INTO audit_log(action) VALUES(?)", action).Error
}
func (r *Repo) Audits(ctx context.Context) ([]biz.Audit, error) {
	rows := []biz.Audit{}
	err := r.db.WithContext(ctx).Table("audit_log").Order("id DESC").Limit(100).Find(&rows).Error
	return rows, err
}
