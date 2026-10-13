// Package media implements locked references and retryable physical cleanup.
package media

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"os"
	"path/filepath"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/media"
	"peptide-site/internal/data"
	"peptide-site/internal/pkg/cosstore"
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/filestaging"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Repo struct {
	Data  *data.Data
	Store cosstore.Store
}

func NewRepo(d *data.Data) *Repo { return &Repo{Data: d} }
func audit(tx *gorm.DB, ctx context.Context, action string, id int64) error {
	i := auth.Current(ctx)
	var actor any
	if i != nil {
		actor = i.User.ID
	}
	return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,?,'media',?::bigint::text)", actor, action, id).Error
}
func (r *Repo) List(ctx context.Context, page, size int, kind, search string, group ...string) ([]biz.Media, int64, error) {
	rows := []biz.Media{}
	var total int64
	q := r.Data.DB.WithContext(ctx).Table("media").Where("available AND deleted_at IS NULL AND media_type IN ? AND status='ready'", []string{"image"})
	if kind != "" {
		if kind != "image" {
			return nil, 0, errdef.Invalid()
		}
		q = q.Where("media_type=?", kind)
	}
	if len(group) > 0 && group[0] != "" {
		if group[0] == "ungrouped" {
			q = q.Where("group_id IS NULL")
		} else {
			q = q.Where("group_id=?", group[0])
		}
	}
	if search != "" {
		q = q.Where("strpos(lower(name),lower(?))>0", search)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}
func (r *Repo) Get(ctx context.Context, id int64) (biz.Media, error) {
	var row biz.Media
	err := r.Data.DB.WithContext(ctx).Table("media").Where("id=? AND available AND status='ready' AND media_type='image' AND deleted_at IS NULL", id).Take(&row).Error
	return row, err
}
func (r *Repo) FindFile(ctx context.Context, root, key string) (biz.Media, string, error) {
	var row biz.Media
	err := r.Data.DB.WithContext(ctx).Table("media").Where("((storage_provider='local' AND storage_root_id=? AND storage_key=?) OR url=?) AND available AND deleted_at IS NULL", root, key, "/uploads/"+root+"/"+key).Take(&row).Error
	if err != nil {
		return row, "", err
	}
	if row.StorageProvider == "cos" && row.BatchID != nil {
		batch, err := r.Batch(ctx, *row.BatchID)
		return row, strings.TrimRight(batch.PublicDomain, "/") + "/" + row.StorageKey, err
	}
	path, err := r.Data.StoragePath(ctx, root)
	return row, filepath.Join(path, key), err
}

// Save imports a local image for trusted seed tools. HTTP uploads use InitUpload.
func (r *Repo) Save(ctx context.Context, row *biz.Media) error {
	if row.MediaType != "image" || (row.StorageProvider != "" && row.StorageProvider != "local") {
		return errdef.Invalid()
	}
	row.StorageProvider = "local"
	row.Status = "ready"
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("media").Create(row).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "media.upload", row.ID)
	})
}
func (r *Repo) Rename(ctx context.Context, id int64, name string) (biz.Media, error) {
	name = strings.TrimSpace(name)
	if err := biz.ValidateName(name); err != nil {
		return biz.Media{}, err
	}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row biz.Media
		if err := tx.Table("media").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND available AND status='ready' AND media_type='image' AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE media SET name=?,updated_at=now() WHERE id=?", name, id).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "media.rename", id)
	})
	if err != nil {
		return biz.Media{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repo) References(ctx context.Context, id int64) ([]biz.Reference, error) {
	rows := []biz.Reference{}
	if _, err := r.Get(ctx, id); err != nil {
		return rows, err
	}
	err := r.Data.DB.WithContext(ctx).Table("media_reference").Where("media_id=?", id).Order("id").Find(&rows).Error
	return rows, err
}
func (r *Repo) Delete(ctx context.Context, id int64) error { return r.DeleteBatch(ctx, []int64{id}) }
func (r *Repo) Cleanup(ctx context.Context) error {
	rows := []biz.Media{}
	if err := r.Data.DB.WithContext(ctx).Table("media").Where("NOT available AND deleted_at IS NOT NULL AND cleanup_pending").Limit(100).Find(&rows).Error; err != nil {
		return err
	}
	var cleanupErr error
	for _, row := range rows {
		if ctx.Err() != nil {
			return errors.Join(cleanupErr, ctx.Err())
		}
		if row.StorageProvider == "cos" {
			if r.Store == nil || row.BatchID == nil {
				continue
			}
			batch, err := r.Batch(ctx, *row.BatchID)
			if err != nil {
				return err
			}
			if err = r.Store.Delete(ctx, batch.Location(), row.StorageKey); err != nil {
				cleanupErr = errors.Join(cleanupErr, err)
				continue
			}
			if err = r.Data.DB.WithContext(ctx).Exec("UPDATE media SET cleanup_pending=false WHERE id=? AND NOT available", row.ID).Error; err != nil {
				return err
			}
			continue
		}
		path, err := r.Data.StoragePath(ctx, row.StorageRootID)
		if err != nil {
			continue
		}
		if filepath.Base(row.StorageKey) != row.StorageKey || strings.HasPrefix(row.StorageKey, ".") {
			continue
		}
		err = os.Remove(filepath.Join(path, row.StorageKey))
		if err == nil || errors.Is(err, os.ErrNotExist) {
			if err := r.Data.DB.WithContext(ctx).Exec("UPDATE media SET cleanup_pending=false WHERE id=? AND NOT available", row.ID).Error; err != nil {
				return err
			}
		}
	}
	cleanupErr = errors.Join(cleanupErr, r.CleanupUploads(ctx))
	return errors.Join(cleanupErr, r.cleanupOrphans(ctx))
}

// SyncReferences locks all new IDs in ascending order before removing old
// references. The same locks protect deletion, preventing dangling references.
func SyncReferences(tx *gorm.DB, owner string, ownerID string, fields map[string][]int64) error {
	ids := map[int64]bool{}
	for _, list := range fields {
		for _, id := range list {
			if id <= 0 {
				return errdef.Invalid()
			}
			ids[id] = true
		}
	}
	ordered := make([]int64, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	for _, id := range ordered {
		var row biz.Media
		if err := tx.Table("media").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND available AND status='ready' AND media_type='image' AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errdef.Invalid()
			}
			return err
		}
	}
	if err := tx.Exec("DELETE FROM media_reference WHERE owner_type=? AND owner_id=?", owner, ownerID).Error; err != nil {
		return err
	}
	for field, list := range fields {
		for _, id := range list {
			if err := tx.Exec("INSERT INTO media_reference(media_id,owner_type,owner_id,field) VALUES(?,?,?,?) ON CONFLICT DO NOTHING", id, owner, ownerID, field).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// Crash leftovers are only removed after a 24-hour grace period. Running uploads
// protect both their temporary and final paths, including retired generations.
func (r *Repo) cleanupOrphans(ctx context.Context) error {
	var roots []struct{ ID, Path string }
	if err := r.Data.DB.WithContext(ctx).Table("storage_root").Find(&roots).Error; err != nil {
		return err
	}
	key := regexp.MustCompile(`^[0-9a-f]{32}\.(png|jpg|webp|gif|pdf|mp4|webm)$`)
	for _, root := range roots {
		entries, err := os.ReadDir(root.Path)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, ".upload-") && !key.MatchString(name) {
				continue
			}
			path := filepath.Join(root.Path, name)
			if filestaging.Active(path) {
				continue
			}
			info, err := entry.Info()
			if err != nil || time.Since(info.ModTime()) < 24*time.Hour {
				continue
			}
			if !strings.HasPrefix(name, ".upload-") {
				var count int64
				if err := r.Data.DB.WithContext(ctx).Table("media").Where("(storage_root_id=? AND storage_key=?) OR url=?", root.ID, name, "/uploads/"+root.ID+"/"+name).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					continue
				}
			}
			if filestaging.Active(path) {
				continue
			}
			_ = os.Remove(path)
		}
	}
	return nil
}
