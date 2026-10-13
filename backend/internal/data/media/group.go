package media

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	biz "peptide-site/internal/biz/media"
	"peptide-site/internal/pkg/errdef"
	"sort"
	"strings"
)

type Group struct {
	ID    int64  `json:"id,string"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

func (r *Repo) Groups(ctx context.Context) ([]Group, error) {
	rows := []Group{}
	err := r.Data.DB.WithContext(ctx).Raw(`SELECT g.id,g.name,count(m.id) AS count FROM media_group g LEFT JOIN media m ON m.group_id=g.id AND m.available AND m.deleted_at IS NULL AND m.media_type='image' AND m.status='ready' WHERE g.deleted_at IS NULL GROUP BY g.id ORDER BY g.id`).Scan(&rows).Error
	return rows, err
}
func (r *Repo) SaveGroup(ctx context.Context, id int64, name string) (Group, error) {
	name = strings.TrimSpace(name)
	if err := biz.ValidateName(name); err != nil {
		return Group{}, err
	}
	if name == "全部图片" || name == "未分组" {
		return Group{}, errdef.Invalid()
	}
	row := Group{ID: id, Name: name}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id == 0 {
			if err := tx.Raw("INSERT INTO media_group(name) VALUES(?) RETURNING id", name).Scan(&row.ID).Error; err != nil {
				return err
			}
		} else {
			result := tx.Exec("UPDATE media_group SET name=?,updated_at=now() WHERE id=? AND deleted_at IS NULL", name, id)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return gorm.ErrRecordNotFound
			}
		}
		return audit(tx, ctx, "media.group.save", row.ID)
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return Group{}, errdef.New(409, "MEDIA_GROUP_EXISTS")
	}
	return row, err
}
func lockGroup(tx *gorm.DB, id *int64) error {
	if id == nil {
		return nil
	}
	if *id <= 0 {
		return errdef.Invalid()
	}
	var row Group
	return tx.Table("media_group").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", *id).Take(&row).Error
}
func (r *Repo) DeleteGroup(ctx context.Context, id int64) error {
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, &id); err != nil {
			return err
		}
		if err := tx.Exec("UPDATE media SET group_id=NULL,updated_at=now() WHERE group_id=?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE media_group SET deleted_at=now(),updated_at=now() WHERE id=?", id).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "media.group.delete", id)
	})
}
func sortedIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return nil, errdef.Invalid()
	}
	out := append([]int64(nil), ids...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	for i, id := range out {
		if id <= 0 || (i > 0 && id == out[i-1]) {
			return nil, errdef.Invalid()
		}
	}
	return out, nil
}
func lockImages(tx *gorm.DB, ids []int64) error {
	var rows []biz.Media
	if err := tx.Table("media").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ? AND available AND status='ready' AND deleted_at IS NULL AND media_type='image'", ids).Order("id").Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) != len(ids) {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *Repo) Move(ctx context.Context, ids []int64, group *int64) error {
	ids, err := sortedIDs(ids)
	if err != nil {
		return err
	}
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, group); err != nil {
			return err
		}
		if err := lockImages(tx, ids); err != nil {
			return err
		}
		if err := tx.Exec("UPDATE media SET group_id=?,updated_at=now() WHERE id IN ?", group, ids).Error; err != nil {
			return err
		}
		for _, id := range ids {
			if err := audit(tx, ctx, "media.move", id); err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *Repo) DeleteBatch(ctx context.Context, ids []int64) error {
	ids, err := sortedIDs(ids)
	if err != nil {
		return err
	}
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockImages(tx, ids); err != nil {
			return err
		}
		var count int64
		if err := tx.Table("media_reference").Where("media_id IN ?", ids).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errdef.New(409, "MEDIA_IN_USE")
		}
		if err := tx.Exec("UPDATE media SET available=false,deleted_at=now(),cleanup_pending=true,updated_at=now() WHERE id IN ?", ids).Error; err != nil {
			return err
		}
		for _, id := range ids {
			if err := audit(tx, ctx, "media.delete", id); err != nil {
				return err
			}
		}
		return nil
	})
}
