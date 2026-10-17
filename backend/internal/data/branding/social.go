package branding

import (
	"context"
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	biz "peptide-site/internal/biz/branding"
	ids "peptide-site/internal/biz/catalog"
	media "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/errdef"
)

type socialRow struct {
	ID, IconMediaID, Version   int64
	Name, Platform, Link, Icon string
	Enabled                    bool
	Sort                       int
}

func socialDTO(row socialRow) biz.SocialAccount {
	return biz.SocialAccount{ID: strconv.FormatInt(row.ID, 10), Name: row.Name, Platform: row.Platform,
		Link: row.Link, IconMediaID: strconv.FormatInt(row.IconMediaID, 10), Icon: row.Icon,
		Enabled: row.Enabled, Sort: row.Sort, Version: row.Version}
}

func socialQuery(tx *gorm.DB) *gorm.DB {
	return tx.Table("social_account s").Select("s.*,m.url AS icon").
		Joins("LEFT JOIN media m ON m.id=s.icon_media_id AND m.available AND m.deleted_at IS NULL").Where("s.deleted_at IS NULL")
}

func (r *Repo) SocialAccounts(ctx context.Context, public bool) ([]biz.SocialAccount, error) {
	q := socialQuery(r.Data.DB.WithContext(ctx))
	if public {
		q = q.Where("s.enabled AND m.id IS NOT NULL")
	}
	var rows []socialRow
	err := q.Order("s.sort,s.id").Find(&rows).Error
	items := make([]biz.SocialAccount, 0, len(rows))
	for _, row := range rows {
		items = append(items, socialDTO(row))
	}
	return items, err
}

func (r *Repo) SaveSocialAccount(ctx context.Context, id int64, account biz.SocialAccount) (biz.SocialAccount, error) {
	if err := biz.ValidateSocialAccounts([]biz.SocialAccount{account}); err != nil {
		return account, err
	}
	iconID, err := ids.ParseID(account.IconMediaID)
	if err != nil || iconID == 0 {
		return account, errdef.Invalid()
	}
	if id == 0 && (account.ID != "" || account.ExpectedVersion != 0) {
		return account, errdef.Invalid()
	}
	if id != 0 && account.ID != "" && account.ID != strconv.FormatInt(id, 10) {
		return account, errdef.Invalid()
	}
	err = r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old socialRow
		if id == 0 {
			// Serialize creates so concurrent requests cannot exceed the account limit.
			if err := tx.Exec("SELECT pg_advisory_xact_lock(784296318)").Error; err != nil {
				return err
			}
			var count int64
			if err := tx.Table("social_account").Where("deleted_at IS NULL").Count(&count).Error; err != nil {
				return err
			}
			if count >= biz.MaxSocialAccounts {
				return errdef.Invalid()
			}
		} else {
			if err := tx.Table("social_account").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&old).Error; err != nil {
				return err
			}
			if old.Version != account.ExpectedVersion {
				return errdef.New(409, "VERSION_CONFLICT")
			}
		}
		fields := map[string]any{"name": account.Name, "platform": account.Platform, "link": account.Link,
			"icon_media_id": iconID, "enabled": account.Enabled, "sort": account.Sort, "version": old.Version + 1, "updated_by": actor(ctx)}
		if id == 0 {
			fields["created_by"] = actor(ctx)
			if err := tx.Table("social_account").Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&fields).Error; err != nil {
				return err
			}
			id = fields["id"].(int64)
		} else {
			fields["updated_at"] = gorm.Expr("now()")
			if err := tx.Table("social_account").Where("id=?", id).Updates(fields).Error; err != nil {
				return err
			}
		}
		if err := media.SyncReferences(tx, "social_account", strconv.FormatInt(id, 10), map[string][]int64{"icon": {iconID}}); err != nil {
			return err
		}
		action := "social.update"
		if old.ID == 0 {
			action = "social.create"
		}
		if err := audit(tx, ctx, action, "social_account", strconv.FormatInt(id, 10)); err != nil {
			return err
		}
		var saved socialRow
		if err := socialQuery(tx).Where("s.id=?", id).Take(&saved).Error; err != nil {
			return err
		}
		account = socialDTO(saved)
		return nil
	})
	return account, err
}

// ChangeSocialAccount updates only enabled state or soft-deletes one versioned record.
func (r *Repo) ChangeSocialAccount(ctx context.Context, id, version int64, enabled bool, remove bool) (biz.SocialAccount, error) {
	var result biz.SocialAccount
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row socialRow
		if err := tx.Table("social_account").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			return errdef.New(409, "VERSION_CONFLICT")
		}
		fields := map[string]any{"version": row.Version + 1, "updated_at": gorm.Expr("now()"), "updated_by": actor(ctx)}
		action := "social.enabled"
		if remove {
			fields["deleted_at"] = gorm.Expr("now()")
			action = "social.delete"
			if err := media.SyncReferences(tx, "social_account", strconv.FormatInt(id, 10), nil); err != nil {
				return err
			}
		} else {
			fields["enabled"] = enabled
		}
		if err := tx.Table("social_account").Where("id=?", id).Updates(fields).Error; err != nil {
			return err
		}
		if err := audit(tx, ctx, action, "social_account", strconv.FormatInt(id, 10)); err != nil {
			return err
		}
		if !remove {
			var saved socialRow
			if err := socialQuery(tx).Where("s.id=?", id).Take(&saved).Error; err != nil {
				return err
			}
			result = socialDTO(saved)
		}
		return nil
	})
	return result, err
}
