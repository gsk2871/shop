package catalog

import (
	"context"
	"database/sql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	biz "peptide-site/internal/biz/catalog"
	"peptide-site/internal/pkg/errdef"
	"strconv"
)

type categoryRow struct {
	ID      int64
	Name    string
	Sort    int
	Version int64
}

func categoryDTO(row categoryRow) biz.Category {
	return biz.Category{ID: strconv.FormatInt(row.ID, 10), Name: row.Name, Sort: row.Sort, Version: row.Version}
}
func category(tx *gorm.DB, id int64, lock bool) (categoryRow, error) {
	var row categoryRow
	q := tx.Table("product_category").Where("id=? AND deleted_at IS NULL", id)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "SHARE"})
	}
	err := q.Take(&row).Error
	return row, err
}
func (r *Repo) Category(ctx context.Context, id int64) (biz.Category, error) {
	row, err := category(r.Data.DB.WithContext(ctx), id, false)
	return categoryDTO(row), err
}
func (r *Repo) Categories(ctx context.Context, page, size int, search string, all bool) ([]biz.Category, int64, error) {
	items := []biz.Category{}
	var total int64
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Table("product_category").Where("deleted_at IS NULL")
		if search != "" {
			q = q.Where("strpos(lower(name),lower(?))>0", search)
		}
		if err := q.Count(&total).Error; err != nil {
			return err
		}
		var rows []categoryRow
		q = q.Order("sort,id")
		if !all {
			q = q.Offset((page - 1) * size).Limit(size)
		}
		if err := q.Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			items = append(items, categoryDTO(row))
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return items, total, err
}
func (r *Repo) SaveCategory(ctx context.Context, id int64, c biz.Category) (biz.Category, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old categoryRow
		if id != 0 {
			if err := tx.Table("product_category").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&old).Error; err != nil {
				return err
			}
			if old.Version != c.ExpectedVersion {
				return errdef.New(409, "VERSION_CONFLICT")
			}
		} else if c.ExpectedVersion != 0 || c.ID != "" {
			return errdef.Invalid()
		}
		fields := map[string]any{"name": c.Name, "sort": c.Sort, "version": old.Version + 1, "updated_by": actor(ctx)}
		if id == 0 {
			fields["created_by"] = actor(ctx)
			if err := tx.Table("product_category").Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&fields).Error; err != nil {
				return err
			}
			id = fields["id"].(int64)
		} else {
			fields["updated_at"] = gorm.Expr("now()")
			if err := tx.Table("product_category").Where("id=?", id).Updates(fields).Error; err != nil {
				return err
			}
		}
		action := "category.update"
		if old.ID == 0 {
			action = "category.create"
		}
		return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,?,'product_category',?)", actor(ctx), action, strconv.FormatInt(id, 10)).Error
	})
	if err != nil {
		return c, err
	}
	return r.Category(ctx, id)
}
func (r *Repo) DeleteCategory(ctx context.Context, id, version int64) error {
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row categoryRow
		if err := tx.Table("product_category").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		var count int64
		if err := tx.Table("product").Where("category_id=? AND deleted_at IS NULL", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errdef.New(409, "CATEGORY_IN_USE")
		}
		if err := tx.Exec("UPDATE product_category SET deleted_at=now(),updated_at=now(),updated_by=?,version=version+1 WHERE id=?", actor(ctx), id).Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,'category.delete','product_category',?)", actor(ctx), strconv.FormatInt(id, 10)).Error
	})
}
