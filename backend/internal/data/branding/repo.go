package branding

import (
	"context"
	"database/sql"
	"encoding/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/branding"
	ids "peptide-site/internal/biz/catalog"
	"peptide-site/internal/data"
	media "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/errdef"
	"strconv"
	"time"
)

type Repo struct {
	Data  *data.Data
	Media *media.Repo
}

func NewRepo(d *data.Data) *Repo { return &Repo{d, media.NewRepo(d)} }
func actor(ctx context.Context) int64 {
	if i := auth.Current(ctx); i != nil {
		return i.User.ID
	}
	return -1
}
func can(ctx context.Context, p string) bool { i := auth.Current(ctx); return i != nil && i.Allows(p) }
func audit(tx *gorm.DB, ctx context.Context, action, kind, id string) error {
	return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,?,?,?)", actor(ctx), action, kind, id).Error
}
func (r *Repo) image(ctx context.Context, id string) (int64, string, error) {
	if id == "" {
		return 0, "", nil
	}
	n, e := ids.ParseID(id)
	if e != nil {
		return 0, "", e
	}
	m, e := r.Media.Get(ctx, n)
	if e != nil && e != gorm.ErrRecordNotFound {
		return 0, "", e
	}
	if e != nil || m.MediaType != "image" {
		return 0, "", errdef.Invalid()
	}
	return n, m.URL, nil
}
func (r *Repo) Settings(ctx context.Context) (biz.Settings, error) {
	s := biz.Default()
	var rec struct {
		Payload []byte
		Version int64
	}
	e := r.Data.DB.WithContext(ctx).Table("site_config").Where("id=1").Take(&rec).Error
	if e == gorm.ErrRecordNotFound {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	if e = json.Unmarshal(rec.Payload, &s); e != nil {
		return s, e
	}
	s.Version = rec.Version
	return s, nil
}
func (r *Repo) SaveSettings(ctx context.Context, s biz.Settings) (biz.Settings, error) {
	if e := s.Validate(); e != nil {
		return s, e
	}
	id, url, e := r.image(ctx, s.LogoMediaID)
	if e != nil {
		return s, e
	}
	s.Logo = url
	iconIDs := make([]int64, 0, len(s.SocialAccounts))
	for i := range s.SocialAccounts {
		iconID, iconURL, err := r.image(ctx, s.SocialAccounts[i].IconMediaID)
		if err != nil {
			return s, err
		}
		s.SocialAccounts[i].Icon = iconURL
		iconIDs = append(iconIDs, iconID)
	}
	e = r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SELECT pg_advisory_xact_lock(784296317)").Error; e != nil {
			return e
		}
		var rec struct{ Version int64 }
		e := tx.Table("site_config").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=1").Take(&rec).Error
		if e != nil && e != gorm.ErrRecordNotFound {
			return e
		}
		v := rec.Version
		if v == 0 {
			v = 1
		}
		if s.ExpectedVersion != v {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		s.Version = v + 1
		s.ExpectedVersion = 0
		b, e := json.Marshal(s)
		if e != nil {
			return e
		}
		if e = tx.Exec("INSERT INTO site_config(id,payload,version) VALUES(1,?::jsonb,?) ON CONFLICT(id) DO UPDATE SET payload=EXCLUDED.payload,version=EXCLUDED.version", string(b), s.Version).Error; e != nil {
			return e
		}
		refs := map[string][]int64{"social_icon": iconIDs}
		if id != 0 {
			refs["logo"] = []int64{id}
		}
		if e = media.SyncReferences(tx, "site_config", "1", refs); e != nil {
			return e
		}
		return audit(tx, ctx, "site.update", "site_config", "1")
	})
	return s, e
}

type Slot struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (r *Repo) Slots(ctx context.Context) ([]Slot, error) {
	rows := []Slot{}
	e := r.Data.DB.WithContext(ctx).Table("banner_slot").Order("id").Find(&rows).Error
	return rows, e
}

type row struct {
	ID                                       int64
	SlotCode, Title, Subtitle, CTAText, Link string
	ImageMediaID                             int64
	MobileImageMediaID                       *int64
	Sort                                     int
	Enabled                                  bool
	StartsAt, EndsAt                         *time.Time
	Version                                  int64
}

func (r *Repo) get(tx *gorm.DB, id int64) (biz.Banner, error) {
	var rec row
	e := tx.Table("banner").Where("id=? AND deleted_at IS NULL", id).Take(&rec).Error
	if e != nil {
		return biz.Banner{}, e
	}
	b := biz.Banner{ID: strconv.FormatInt(id, 10), SlotCode: rec.SlotCode, Title: rec.Title, Subtitle: rec.Subtitle, CTAText: rec.CTAText, Link: rec.Link, ImageMediaID: strconv.FormatInt(rec.ImageMediaID, 10), Sort: rec.Sort, Enabled: rec.Enabled, StartsAt: rec.StartsAt, EndsAt: rec.EndsAt, Version: rec.Version}
	var url string
	e = tx.Table("media").Select("url").Where("id=? AND available AND deleted_at IS NULL", rec.ImageMediaID).Take(&url).Error
	if e != nil {
		return b, e
	}
	b.Image = url
	if rec.MobileImageMediaID != nil {
		b.MobileImageMediaID = strconv.FormatInt(*rec.MobileImageMediaID, 10)
		e = tx.Table("media").Select("url").Where("id=? AND available AND deleted_at IS NULL", *rec.MobileImageMediaID).Take(&url).Error
		b.MobileImage = url
	}
	return b, e
}
func (r *Repo) Get(ctx context.Context, id int64) (biz.Banner, error) {
	var b biz.Banner
	e := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { var e error; b, e = r.get(tx, id); return e }, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return b, e
}
func (r *Repo) List(ctx context.Context, slot string, page, size int, public bool) ([]any, int64, error) {
	items := []any{}
	var total int64
	e := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Table("banner").Where("deleted_at IS NULL")
		if slot != "" {
			q = q.Where("slot_code=?", slot)
		}
		if public {
			q = q.Where("enabled AND (starts_at IS NULL OR starts_at<=now()) AND (ends_at IS NULL OR ends_at>now())")
		}
		if e := q.Count(&total).Error; e != nil {
			return e
		}
		var records []row
		query := q.Order("sort,id")
		if !public {
			query = query.Offset((page - 1) * size).Limit(size)
		}
		if e := query.Find(&records).Error; e != nil {
			return e
		}
		for _, rec := range records {
			b, e := r.get(tx, rec.ID)
			if e != nil {
				return e
			}
			if public {
				items = append(items, biz.Public(b))
			} else {
				items = append(items, b)
			}
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return items, total, e
}
func (r *Repo) Save(ctx context.Context, id int64, b biz.Banner) (biz.Banner, error) {
	if e := b.Validate(); e != nil {
		return b, e
	}
	image, _, e := r.image(ctx, b.ImageMediaID)
	if e != nil {
		return b, e
	}
	mobile, _, e := r.image(ctx, b.MobileImageMediaID)
	if e != nil {
		return b, e
	}
	e = r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if e := tx.Table("banner_slot").Where("code=?", b.SlotCode).Count(&count).Error; e != nil {
			return e
		}
		if count != 1 {
			return errdef.Invalid()
		}
		var old row
		if id != 0 {
			if e := tx.Table("banner").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&old).Error; e != nil {
				return e
			}
			if old.Version != b.ExpectedVersion {
				return errdef.New(409, "VERSION_CONFLICT")
			}
		} else if b.ID != "" || b.ExpectedVersion != 0 {
			return errdef.Invalid()
		}
		if b.Enabled != old.Enabled && !can(ctx, "banner.publish") {
			return errdef.Denied()
		}
		var mobileValue any
		if mobile != 0 {
			mobileValue = mobile
		}
		fields := map[string]any{"slot_code": b.SlotCode, "title": b.Title, "subtitle": b.Subtitle, "image_media_id": image, "mobile_image_media_id": mobileValue, "cta_text": b.CTAText, "link": b.Link, "sort": b.Sort, "enabled": b.Enabled, "starts_at": b.StartsAt, "ends_at": b.EndsAt, "version": old.Version + 1, "updated_by": actor(ctx), "updated_at": time.Now().UTC()}
		if id == 0 {
			fields["created_by"] = actor(ctx)
			if e := tx.Table("banner").Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&fields).Error; e != nil {
				return e
			}
			id = fields["id"].(int64)
		} else if e := tx.Table("banner").Where("id=?", id).Updates(fields).Error; e != nil {
			return e
		}
		refs := map[string][]int64{"image": {image}}
		if mobile != 0 {
			refs["mobile_image"] = []int64{mobile}
		}
		if e := media.SyncReferences(tx, "banner", strconv.FormatInt(id, 10), refs); e != nil {
			return e
		}
		action := "banner.update"
		if old.ID == 0 {
			action = "banner.create"
		}
		return audit(tx, ctx, action, "banner", strconv.FormatInt(id, 10))
	})
	if e != nil {
		return b, e
	}
	return r.Get(ctx, id)
}
func (r *Repo) Enabled(ctx context.Context, id, version int64, enabled bool) (biz.Banner, error) {
	e := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rec row
		if e := tx.Table("banner").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&rec).Error; e != nil {
			return e
		}
		if rec.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		if e := tx.Exec("UPDATE banner SET enabled=?,version=version+1,updated_by=?,updated_at=now() WHERE id=?", enabled, actor(ctx), id).Error; e != nil {
			return e
		}
		return audit(tx, ctx, "banner.publish", "banner", strconv.FormatInt(id, 10))
	})
	if e != nil {
		return biz.Banner{}, e
	}
	return r.Get(ctx, id)
}
func (r *Repo) Delete(ctx context.Context, id, version int64) error {
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rec row
		if e := tx.Table("banner").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&rec).Error; e != nil {
			return e
		}
		if rec.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		if e := tx.Exec("UPDATE banner SET deleted_at=now(),version=version+1,updated_by=?,updated_at=now() WHERE id=?", actor(ctx), id).Error; e != nil {
			return e
		}
		if e := media.SyncReferences(tx, "banner", strconv.FormatInt(id, 10), nil); e != nil {
			return e
		}
		return audit(tx, ctx, "banner.delete", "banner", strconv.FormatInt(id, 10))
	})
}
