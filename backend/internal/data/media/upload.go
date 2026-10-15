package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/media"
	"peptide-site/internal/conf"
	"peptide-site/internal/pkg/cosstore"
	"peptide-site/internal/pkg/errdef"
	"strings"
	"time"
)

type Batch struct {
	ID                                      string
	UploaderID                              int64
	Bucket, Region, PublicDomain, KeyPrefix string
	MaxBytes                                int64
	ExpiresAt, CleanupAfter                 time.Time
	CleanedAt                               *time.Time
	CreatedAt                               time.Time
}

func (b Batch) Location() cosstore.Location {
	return cosstore.Location{Bucket: b.Bucket, Region: b.Region}
}
func (b Batch) Temporary(key string) string {
	return b.KeyPrefix + "/temporary/" + b.ID + "/" + key[strings.LastIndex(key, "/")+1:]
}

type UploadFile struct {
	Name string `json:"name"`
	MIME string `json:"mime"`
	Size int64  `json:"size"`
}
type UploadInput struct {
	GroupID *int64       `json:"groupId,string"`
	Files   []UploadFile `json:"files"`
}
type UploadItem struct {
	ID   int64  `json:"id,string"`
	Key  string `json:"key"`
	Name string `json:"name"`
	MIME string `json:"mime"`
}
type UploadResult struct {
	BatchID    string              `json:"batchId"`
	Bucket     string              `json:"bucket"`
	Region     string              `json:"region"`
	Credential cosstore.Credential `json:"credential"`
	Items      []UploadItem        `json:"items"`
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func (r *Repo) Batch(ctx context.Context, id string) (Batch, error) {
	var b Batch
	err := r.Data.DB.WithContext(ctx).Table("media_upload_batch").Where("id=?", id).Take(&b).Error
	return b, err
}
func uploader(ctx context.Context) int64 {
	if i := auth.Current(ctx); i != nil {
		return i.User.ID
	}
	return 0
}
func (r *Repo) InitUpload(ctx context.Context, c conf.Storage, in UploadInput) (UploadResult, error) {
	if !c.COS.Enabled || r.Store == nil {
		return UploadResult{}, errdef.New(503, "COS_NOT_CONFIGURED")
	}
	if uploader(ctx) <= 0 {
		return UploadResult{}, errdef.New(401, "AUTHENTICATION_REQUIRED")
	}
	if len(in.Files) == 0 || len(in.Files) > 20 {
		return UploadResult{}, errdef.Invalid()
	}
	batchID, err := randomID()
	if err != nil {
		return UploadResult{}, err
	}
	now := time.Now().UTC()
	b := Batch{ID: batchID, UploaderID: uploader(ctx), Bucket: c.COS.Bucket, Region: c.COS.Region, PublicDomain: strings.TrimRight(c.COS.PublicDomain, "/"), KeyPrefix: c.COS.KeyPrefix, MaxBytes: c.MaxImageBytes, ExpiresAt: now.Add(24 * time.Hour), CleanupAfter: now.Add(25 * time.Hour), CreatedAt: now}
	rows := make([]biz.Media, 0, len(in.Files))
	items := make([]UploadItem, 0, len(in.Files))
	keys := []string{}
	for _, file := range in.Files {
		allowed := false
		for _, mime := range c.AllowedTypes {
			if mime == file.MIME {
				allowed = true
			}
		}
		if !allowed || !biz.ValidImageName(file.Name, file.MIME) || biz.ValidateName(file.Name) != nil || file.Size <= 0 || file.Size > c.MaxImageBytes {
			return UploadResult{}, errdef.Invalid()
		}
		token, err := randomID()
		if err != nil {
			return UploadResult{}, err
		}
		key := b.KeyPrefix + "/images/" + token + "." + biz.ImageExtension(file.MIME)
		rows = append(rows, biz.Media{Name: file.Name, URL: "", MIME: file.MIME, MediaType: "image", Size: file.Size, StorageProvider: "cos", StorageRootID: "cos:" + b.Region + ":" + b.Bucket, StorageKey: key, Status: "pending", BatchID: &b.ID, GroupID: in.GroupID, UploadedBy: &b.UploaderID, CreatedAt: now, UpdatedAt: now})
		keys = append(keys, b.Temporary(key))
	}
	credential, err := r.Store.Authorize(ctx, b.Location(), keys, b.MaxBytes, 30*time.Minute)
	if err != nil {
		return UploadResult{}, err
	}
	err = r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, in.GroupID); err != nil {
			return err
		}
		if err := tx.Table("media_upload_batch").Create(&b).Error; err != nil {
			return err
		}
		// Select all explicitly keeps available=false despite the database default.
		for i := range rows {
			if err := tx.Table("media").Select("*").Create(&rows[i]).Error; err != nil {
				return err
			}
			items = append(items, UploadItem{rows[i].ID, keys[i], rows[i].Name, rows[i].MIME})
		}
		return audit(tx, ctx, "media.upload.init", rows[0].ID)
	})
	return UploadResult{b.ID, b.Bucket, b.Region, credential, items}, err
}
func (r *Repo) RefreshUpload(ctx context.Context, id string) (cosstore.Credential, error) {
	if r.Store == nil {
		return cosstore.Credential{}, errdef.New(503, "COS_NOT_CONFIGURED")
	}
	b, err := r.Batch(ctx, id)
	if err != nil {
		return cosstore.Credential{}, err
	}
	if b.UploaderID != uploader(ctx) {
		return cosstore.Credential{}, gorm.ErrRecordNotFound
	}
	if time.Until(b.ExpiresAt) < 30*time.Minute {
		return cosstore.Credential{}, errdef.New(409, "MEDIA_UPLOAD_EXPIRED")
	}
	var rows []biz.Media
	if err := r.Data.DB.WithContext(ctx).Table("media").Where("batch_id=? AND status='pending' AND deleted_at IS NULL", id).Find(&rows).Error; err != nil {
		return cosstore.Credential{}, err
	}
	if len(rows) == 0 {
		return cosstore.Credential{}, errdef.New(409, "MEDIA_UPLOAD_STATE")
	}
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, b.Temporary(row.StorageKey))
	}
	return r.Store.Authorize(ctx, b.Location(), keys, b.MaxBytes, 30*time.Minute)
}
func (r *Repo) CompleteUpload(ctx context.Context, id int64) (biz.Media, error) {
	if r.Store == nil {
		return biz.Media{}, errdef.New(503, "COS_NOT_CONFIGURED")
	}
	var row biz.Media
	var invalid error
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("media").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND storage_provider='cos' AND uploaded_by=? AND deleted_at IS NULL", id, uploader(ctx)).Take(&row).Error; err != nil {
			return err
		}
		if row.Status == "ready" {
			return nil
		}
		if row.Status != "pending" || row.BatchID == nil {
			return errdef.New(409, "MEDIA_UPLOAD_STATE")
		}
		var b Batch
		if err := tx.Table("media_upload_batch").Where("id=?", *row.BatchID).Take(&b).Error; err != nil {
			return err
		}
		if !time.Now().Before(b.ExpiresAt) {
			return errdef.New(409, "MEDIA_UPLOAD_EXPIRED")
		}
		body, etag, err := r.Store.Read(ctx, b.Location(), b.Temporary(row.StorageKey), b.MaxBytes)
		if err != nil && !errors.Is(err, cosstore.ErrInvalidObject) {
			return err
		}
		inspectErr := biz.InspectImage(body, row.MIME, b.MaxBytes)
		if err != nil || inspectErr != nil || int64(len(body)) != row.Size {
			invalid = errdef.New(400, "MEDIA_FILE_INVALID")
			if err := tx.Table("media").Where("id=?", id).Updates(map[string]any{"status": "failed", "updated_at": time.Now()}).Error; err != nil {
				return err
			}
			return audit(tx, ctx, "media.upload.failed", id)
		}
		if err := r.Store.Promote(ctx, b.Location(), b.Temporary(row.StorageKey), row.StorageKey, etag, row.MIME); err != nil {
			return err
		}
		row.URL = b.PublicDomain + "/" + row.StorageKey
		row.Available = true
		row.Status = "ready"
		if err := tx.Table("media").Where("id=?", id).Updates(map[string]any{"url": row.URL, "available": true, "status": "ready", "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "media.upload", id)
	})
	if err != nil {
		return biz.Media{}, err
	}
	if invalid != nil {
		return biz.Media{}, invalid
	}
	return row, nil
}
func (r *Repo) CancelUpload(ctx context.Context, id int64) error {
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row biz.Media
		if err := tx.Table("media").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND uploaded_by=? AND storage_provider='cos' AND deleted_at IS NULL", id, uploader(ctx)).Take(&row).Error; err != nil {
			return err
		}
		if row.Status == "canceled" {
			return nil
		}
		if row.Status != "pending" {
			return errdef.New(409, "MEDIA_UPLOAD_STATE")
		}
		if err := tx.Table("media").Where("id=?", id).Updates(map[string]any{"status": "canceled", "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		return audit(tx, ctx, "media.upload.cancel", id)
	})
}

// CleanupUploads checkpoints each remote delete separately. A deadline can never
// roll back progress for objects that have already been removed.
func (r *Repo) CleanupUploads(ctx context.Context) error {
	if r.Store == nil {
		return nil
	}
	var batches []Batch
	if err := r.Data.DB.WithContext(ctx).Table("media_upload_batch").Where("cleaned_at IS NULL AND cleanup_after < now()").Order("cleanup_after").Limit(10).Find(&batches).Error; err != nil {
		return err
	}
	var cleanupErr error
	for _, b := range batches {
		if ctx.Err() != nil {
			return errors.Join(cleanupErr, ctx.Err())
		}
		// An expired upload cannot complete or refresh. Formal-object cleanup uses
		// the same durable queue as deletion of a ready image.
		if err := r.Data.DB.WithContext(ctx).Exec("UPDATE media SET status=CASE WHEN status='pending' THEN 'canceled' ELSE status END,deleted_at=now(),cleanup_pending=true,updated_at=now() WHERE batch_id=? AND status!='ready' AND deleted_at IS NULL", b.ID).Error; err != nil {
			return err
		}
		var rows []biz.Media
		if err := r.Data.DB.WithContext(ctx).Table("media").Where("batch_id=? AND NOT temporary_cleaned", b.ID).Order("id").Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if ctx.Err() != nil {
				return errors.Join(cleanupErr, ctx.Err())
			}
			if err := r.Store.Delete(ctx, b.Location(), b.Temporary(row.StorageKey)); err != nil {
				cleanupErr = errors.Join(cleanupErr, err)
				continue
			}
			if err := r.Data.DB.WithContext(ctx).Exec("UPDATE media SET temporary_cleaned=true WHERE id=?", row.ID).Error; err != nil {
				return err
			}
		}
		if err := r.Data.DB.WithContext(ctx).Exec("UPDATE media_upload_batch SET cleaned_at=now() WHERE id=? AND NOT EXISTS(SELECT 1 FROM media WHERE batch_id=? AND (NOT temporary_cleaned OR cleanup_pending))", b.ID, b.ID).Error; err != nil {
			return err
		}
	}
	return cleanupErr
}
