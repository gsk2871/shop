package leads

import (
	"context"
	"database/sql"
	"encoding/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/leads"
	"peptide-site/internal/data"
	"peptide-site/internal/pkg/errdef"
	"strconv"
)

type Repo struct{ Data *data.Data }

func NewRepo(d *data.Data) *Repo { return &Repo{d} }
func Seed(tx *gorm.DB) error {
	payload, err := json.Marshal(biz.Default())
	if err != nil {
		return err
	}
	return tx.Exec("INSERT INTO lead_popup_config(id,payload,version) VALUES(1,?::jsonb,1) ON CONFLICT(id) DO NOTHING", string(payload)).Error
}
func config(tx *gorm.DB) (biz.Config, error) {
	var row struct {
		Payload []byte
		Version int64
	}
	c := biz.Config{}
	err := tx.Table("lead_popup_config").Where("id=1").Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return c, nil
	} // Until explicitly seeded, remain disabled.
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(row.Payload, &c)
	c.Version = row.Version
	return c, err
}
func (r *Repo) Config(ctx context.Context) (biz.Config, error) {
	return config(r.Data.DB.WithContext(ctx))
}
func audit(tx *gorm.DB, ctx context.Context, action, kind, id string) error {
	identity := auth.Current(ctx)
	if identity == nil {
		return errdef.Denied()
	}
	return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,?,?,?)", identity.User.ID, action, kind, id).Error
}
func (r *Repo) SaveConfig(ctx context.Context, c biz.Config) (biz.Config, error) {
	if c.ExpectedVersion < 1 {
		return c, errdef.Invalid()
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	expected := c.ExpectedVersion
	c.ExpectedVersion = 0
	c.Version = expected + 1
	payload, err := json.Marshal(c)
	if err != nil {
		return c, err
	}
	err = r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("UPDATE lead_popup_config SET payload=?::jsonb,version=version+1 WHERE id=1 AND version=?", string(payload), expected)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		return audit(tx, ctx, "lead_popup.update", "lead_popup_config", "1")
	})
	return c, err
}
func (r *Repo) Submit(ctx context.Context, raw string) error {
	kind, contact, err := biz.Normalize(raw)
	if err != nil {
		return err
	}
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize with disabling the popup: no submissions after disable commits.
		c, err := config(tx.Clauses(clause.Locking{Strength: "SHARE"}))
		if err != nil {
			return err
		}
		if !c.Enabled {
			return errdef.New(409, "POPUP_DISABLED")
		}
		return tx.Exec(`INSERT INTO contact_lead(contact_type,contact) VALUES(?,?)
   ON CONFLICT(contact_type,contact) DO UPDATE SET last_submitted_at=now(),submission_count=contact_lead.submission_count+1,status='pending',version=contact_lead.version+1`, kind, contact).Error
	})
}
func (r *Repo) List(ctx context.Context, f biz.Filter) ([]biz.Lead, int64, error) {
	rows := []biz.Lead{}
	var total int64
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Table("contact_lead")
		if f.ContactType != "" {
			q = q.Where("contact_type = ?", f.ContactType)
		}
		if f.Contact != "" {
			q = q.Where("contact ILIKE ?", "%"+f.Contact+"%")
		}
		if f.Start != nil {
			q = q.Where("last_submitted_at>=?", *f.Start)
		}
		if f.End != nil {
			q = q.Where("last_submitted_at<=?", *f.End)
		}
		if err := q.Count(&total).Error; err != nil {
			return err
		}
		return q.Order("last_submitted_at DESC,id DESC").Limit(f.Size).Offset((f.Page - 1) * f.Size).Find(&rows).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return rows, total, err
}
func (r *Repo) Update(ctx context.Context, id int64, u biz.Update) error {
	if err := u.Validate(); err != nil {
		return err
	}
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("UPDATE contact_lead SET status=?,note=?,version=version+1 WHERE id=? AND version=?", u.Status, u.Note, id, u.ExpectedVersion)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		return audit(tx, ctx, "lead.update", "contact_lead", strconv.FormatInt(id, 10))
	})
}
