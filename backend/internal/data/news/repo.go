// Package news persists news and media references in one transaction.
package news

import (
	"context"
	"database/sql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	auth "peptide-site/internal/biz/admin"
	catalogBiz "peptide-site/internal/biz/catalog"
	biz "peptide-site/internal/biz/news"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	catalog "peptide-site/internal/data/catalog"
	media "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/contentslug"
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/richtext"
	"strconv"
	"time"
)

type Repo struct {
	Data    *data.Data
	Catalog *catalog.Repo
	Media   *media.Repo
}

func NewRepo(d *data.Data, c conf.Config) *Repo {
	return &Repo{d, catalog.NewRepo(d, c), media.NewRepo(d)}
}

type row struct {
	ID                                                               int64
	Title, Slug, Summary, BodyHTML, Status, SEOTitle, SEODescription string
	CoverMediaID                                                     *int64
	Pinned                                                           bool
	Sort                                                             int
	Version                                                          int64
	FirstPublishedAt                                                 *time.Time
	UpdatedAt                                                        time.Time
}

func actor(ctx context.Context) int64 {
	if i := auth.Current(ctx); i != nil {
		return i.User.ID
	}
	return -1
}
func audit(tx *gorm.DB, ctx context.Context, action string, id int64) error {
	return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,?,'article',?::bigint::text)", actor(ctx), action, id).Error
}
func can(ctx context.Context, p string) bool { i := auth.Current(ctx); return i != nil && i.Allows(p) }
func (r *Repo) prepare(ctx context.Context, a *biz.Article) error {
	if a.Slug == "" {
		value, err := contentslug.New(a.Title)
		if err != nil {
			return err
		}
		a.Slug = value
	}
	a.SEOTitle, a.SEODescription = "", ""
	if err := biz.Validate(*a); err != nil {
		return err
	}
	clean, err := r.Catalog.Sanitize(ctx, a.Body)
	if err != nil {
		return err
	}
	a.Body = clean
	a.Cover = ""
	if a.CoverMediaID != "" {
		id, err := catalogBiz.ParseID(a.CoverMediaID)
		if err != nil {
			return err
		}
		asset, err := r.Media.Get(ctx, id)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err != nil || asset.MediaType != "image" {
			return errdef.Invalid()
		}
		a.Cover = asset.URL
	}
	return nil
}
func (r *Repo) get(tx *gorm.DB, id int64) (biz.Article, error) {
	var record row
	if err := tx.Table("article").Where("id=? AND deleted_at IS NULL", id).Take(&record).Error; err != nil {
		return biz.Article{}, err
	}
	a := biz.Article{ID: strconv.FormatInt(record.ID, 10), Title: record.Title, Slug: record.Slug, Summary: record.Summary, Body: richtext.Document{HTML: record.BodyHTML, MediaIDs: []string{}}, Status: record.Status, SEOTitle: record.SEOTitle, SEODescription: record.SEODescription, Pinned: record.Pinned, Sort: record.Sort, Version: record.Version, UpdatedAt: record.UpdatedAt.UTC().Format(time.RFC3339)}
	if record.FirstPublishedAt != nil {
		formatted := record.FirstPublishedAt.UTC().Format(time.RFC3339)
		a.FirstPublishedAt = &formatted
	}
	if record.CoverMediaID != nil {
		a.CoverMediaID = strconv.FormatInt(*record.CoverMediaID, 10)
		var asset struct{ URL string }
		if err := tx.Table("media").Where("id=? AND available AND deleted_at IS NULL", *record.CoverMediaID).Take(&asset).Error; err != nil {
			return a, err
		}
		a.Cover = asset.URL
	}
	var ids []int64
	if err := tx.Table("media_reference").Where("owner_type='article' AND owner_id=? AND field='body'", a.ID).Order("media_id").Pluck("media_id", &ids).Error; err != nil {
		return a, err
	}
	for _, id := range ids {
		a.Body.MediaIDs = append(a.Body.MediaIDs, strconv.FormatInt(id, 10))
	}
	return a, nil
}
func (r *Repo) Get(ctx context.Context, id int64) (biz.Article, error) {
	var a biz.Article
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { var err error; a, err = r.get(tx, id); return err }, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return a, err
}
func (r *Repo) Public(ctx context.Context, slug string) (map[string]any, error) {
	var output map[string]any
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record row
		if err := tx.Table("article").Where("slug=? AND status='published' AND deleted_at IS NULL", slug).Take(&record).Error; err != nil {
			return err
		}
		a, err := r.get(tx, record.ID)
		if err != nil {
			return err
		}
		output = biz.Public(a)
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return output, err
}
func (r *Repo) List(ctx context.Context, public bool, page, size int, search, status string, pinned *bool) ([]map[string]any, int64, error) {
	output := []map[string]any{}
	var total int64
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Table("article").Where("deleted_at IS NULL")
		if public {
			q = q.Where("status='published'")
		}
		if !public {
			if search != "" {
				q = q.Where("strpos(lower(title),lower(?))>0", search)
			}
			if status != "" {
				q = q.Where("status=?", status)
			}
			if pinned != nil {
				q = q.Where("pinned=?", *pinned)
			}
		}
		if err := q.Count(&total).Error; err != nil {
			return err
		}
		rows := []row{}
		if err := q.Order("pinned DESC,sort,first_published_at DESC NULLS LAST,id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			a, err := r.get(tx, row.ID)
			if err != nil {
				return err
			}
			item := map[string]any{"id": a.ID, "title": a.Title, "slug": a.Slug, "summary": a.Summary, "cover": a.Cover, "firstPublishedAt": a.FirstPublishedAt}
			if !public {
				item["status"] = a.Status
				item["version"] = a.Version
				item["pinned"] = a.Pinned
				item["sort"] = a.Sort
				item["updatedAt"] = a.UpdatedAt
			}
			output = append(output, item)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return output, total, err
}
func (r *Repo) Save(ctx context.Context, id int64, a biz.Article) (biz.Article, error) {
	if err := r.prepare(ctx, &a); err != nil {
		return a, err
	}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		version := int64(1)
		var old row
		if id != 0 {
			if err := tx.Table("article").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&old).Error; err != nil {
				return err
			}
			if old.Version != a.ExpectedVersion {
				return errdef.New(409, "VERSION_CONFLICT")
			}
			a.Slug = old.Slug
			version = old.Version + 1
		} else if a.ID != "" || a.ExpectedVersion != 0 {
			return errdef.Invalid()
		}
		if (id == 0 && a.Status != "draft" || id != 0 && a.Status != old.Status) && !can(ctx, "article.publish") {
			return errdef.Denied()
		}
		var cover any
		if a.CoverMediaID != "" {
			value, _ := strconv.ParseInt(a.CoverMediaID, 10, 64)
			cover = value
		}
		published := old.FirstPublishedAt
		if a.Status == "published" && published == nil {
			now := time.Now().UTC()
			published = &now
		}
		columns := map[string]any{"title": a.Title, "slug": a.Slug, "summary": a.Summary, "body_html": a.Body.HTML, "cover_media_id": cover, "status": a.Status, "seo_title": a.SEOTitle, "seo_description": a.SEODescription, "pinned": a.Pinned, "sort": a.Sort, "first_published_at": published, "version": version, "updated_by": actor(ctx), "updated_at": time.Now().UTC()}
		if id == 0 {
			columns["created_by"] = actor(ctx)
			if err := tx.Table("article").Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&columns).Error; err != nil {
				return err
			}
			id = columns["id"].(int64)
		} else if err := tx.Table("article").Where("id=?", id).Updates(columns).Error; err != nil {
			return err
		}
		references := map[string][]int64{"body": richtext.IDs(a.Body)}
		if cover != nil {
			references["cover"] = []int64{cover.(int64)}
		}
		if err := media.SyncReferences(tx, "article", strconv.FormatInt(id, 10), references); err != nil {
			return err
		}
		action := "article.update"
		if old.ID == 0 {
			action = "article.create"
		}
		return audit(tx, ctx, action, id)
	})
	if err != nil {
		return a, err
	}
	return r.Get(ctx, id)
}
func (r *Repo) Status(ctx context.Context, id, version int64, status string) (biz.Article, error) {
	if status != "draft" && status != "published" && status != "offline" {
		return biz.Article{}, errdef.Invalid()
	}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record row
		if err := tx.Table("article").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&record).Error; err != nil {
			return err
		}
		if record.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		if err := tx.Exec("UPDATE article SET status=?,first_published_at=CASE WHEN ?='published' THEN COALESCE(first_published_at,now()) ELSE first_published_at END,version=version+1,updated_at=now(),updated_by=? WHERE id=?", status, status, actor(ctx), id).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "article."+status, id)
	})
	if err != nil {
		return biz.Article{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repo) Delete(ctx context.Context, id, version int64) error {
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record row
		if err := tx.Table("article").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&record).Error; err != nil {
			return err
		}
		if record.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		if err := tx.Exec("UPDATE article SET deleted_at=now(),version=version+1,updated_at=now(),updated_by=? WHERE id=?", actor(ctx), id).Error; err != nil {
			return err
		}
		if err := media.SyncReferences(tx, "article", strconv.FormatInt(id, 10), nil); err != nil {
			return err
		}
		return audit(tx, ctx, "article.delete", id)
	})
}
func (r *Repo) Preview(ctx context.Context, a biz.Article) (map[string]any, error) {
	if err := r.prepare(ctx, &a); err != nil {
		return nil, err
	}
	a.ID = ""
	return biz.Public(a), nil
}
