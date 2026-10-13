package faq

import (
	"context"
	"database/sql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/faq"
	"peptide-site/internal/data"
	"peptide-site/internal/pkg/errdef"
	"strconv"
)

type faqRow struct {
	ID       int64
	Question string
	Answer   string
	Enabled  bool
	Sort     int
	Version  int64
}

func faqDTO(row faqRow) biz.FAQ {
	return biz.FAQ{ID: strconv.FormatInt(row.ID, 10), Question: row.Question, Answer: row.Answer, Enabled: row.Enabled, Sort: row.Sort, Version: row.Version}
}
func (r *Repo) FAQ(ctx context.Context, id int64) (biz.FAQ, error) {
	var row faqRow
	err := r.Data.DB.WithContext(ctx).Table("faq").Where("id=? AND deleted_at IS NULL", id).Take(&row).Error
	return faqDTO(row), err
}
func (r *Repo) List(ctx context.Context, page, size int, search string, public bool) ([]biz.FAQ, int64, error) {
	items := []biz.FAQ{}
	var total int64
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Table("faq").Where("deleted_at IS NULL")
		if public {
			q = q.Where("enabled = ?", true)
		}
		if search != "" {
			q = q.Where("strpos(lower(question),lower(?))>0", search)
		}
		if err := q.Count(&total).Error; err != nil {
			return err
		}
		var rows []faqRow
		q = q.Order("sort,id")
		if !public {
			q = q.Offset((page - 1) * size).Limit(size)
		}
		if err := q.Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			items = append(items, faqDTO(row))
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return items, total, err
}
func (r *Repo) SaveFAQ(ctx context.Context, id int64, c biz.FAQ) (biz.FAQ, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old faqRow
		if id != 0 {
			if err := tx.Table("faq").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&old).Error; err != nil {
				return err
			}
			if old.Version != c.ExpectedVersion {
				return errdef.New(409, "VERSION_CONFLICT")
			}
		} else if c.ExpectedVersion != 0 || c.ID != "" {
			return errdef.Invalid()
		}
		fields := map[string]any{"question": c.Question, "answer": c.Answer, "enabled": c.Enabled, "sort": c.Sort, "version": old.Version + 1, "updated_by": actor(ctx)}
		if id == 0 {
			fields["created_by"] = actor(ctx)
			if err := tx.Table("faq").Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&fields).Error; err != nil {
				return err
			}
			id = fields["id"].(int64)
		} else {
			fields["updated_at"] = gorm.Expr("now()")
			if err := tx.Table("faq").Where("id=?", id).Updates(fields).Error; err != nil {
				return err
			}
		}
		action := "faq.update"
		if old.ID == 0 {
			action = "faq.create"
		}
		return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,?,'faq',?)", actor(ctx), action, strconv.FormatInt(id, 10)).Error
	})
	if err != nil {
		return c, err
	}
	return r.FAQ(ctx, id)
}
func (r *Repo) DeleteFAQ(ctx context.Context, id, version int64) error {
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row faqRow
		if err := tx.Table("faq").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		if err := tx.Exec("UPDATE faq SET deleted_at=now(),updated_at=now(),updated_by=?,version=version+1 WHERE id=?", actor(ctx), id).Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,'faq.delete','faq',?)", actor(ctx), strconv.FormatInt(id, 10)).Error
	})
}

type Repo struct{ Data *data.Data }

func NewRepo(d *data.Data) *Repo { return &Repo{Data: d} }
func actor(ctx context.Context) int64 {
	if i := auth.Current(ctx); i != nil {
		return i.User.ID
	}
	return -1
}
