package whatsapp

import (
	"gorm.io/gorm"
	"log/slog"
)

// Seed only fills missing records. It never overwrites operational edits or an
// existing default. The initial US group intentionally has no fictitious account.
func Seed(tx *gorm.DB) error {
	if err := tx.Exec("INSERT INTO whatsapp_routing_setting(singleton_key,updated_by) VALUES(true,-1) ON CONFLICT(singleton_key) DO NOTHING").Error; err != nil {
		return err
	}
	s, e := getSetting(tx, "UPDATE")
	if e != nil {
		return e
	}
	var id int64
	if e = tx.Table("whatsapp_group_country").Select("group_id").Where("country_code='US' AND deleted_at IS NULL").Scan(&id).Error; e != nil {
		return e
	}
	fresh := id == 0
	if fresh {
		if e = tx.Raw("INSERT INTO whatsapp_group(name,enabled,created_by,updated_by) VALUES('美国客服',true,-1,-1) RETURNING id").Scan(&id).Error; e != nil {
			return e
		}
		if e = tx.Exec("INSERT INTO whatsapp_group_country(group_id,country_code,created_by,updated_by) VALUES(?,'US',-1,-1)", id).Error; e != nil {
			return e
		}
	}
	if e = tx.Exec("INSERT INTO whatsapp_cursor(group_id) VALUES(?) ON CONFLICT(group_id) DO NOTHING", id).Error; e != nil {
		return e
	}
	if s.DefaultGroupID == nil {
		g, e := getRow(tx, id, false)
		if e != nil {
			return e
		}
		pool, e := accounts(tx, id, true)
		if e != nil {
			return e
		}
		// Re-running seed after the initial empty bootstrap remains harmless.
		var total int64
		if e = tx.Table("whatsapp_account").Where("group_id=? AND deleted_at IS NULL", id).Count(&total).Error; e != nil {
			return e
		}
		if g.Enabled && (fresh || len(pool) > 0 || total == 0) {
			return tx.Exec("UPDATE whatsapp_routing_setting SET default_group_id=?,updated_at=GREATEST(clock_timestamp(),updated_at+interval '1 microsecond') WHERE id=? AND default_group_id IS NULL", id, s.ID).Error
		}
		slog.WarnContext(tx.Statement.Context, "US客服配置不可用，请在后台完善并设置默认")
	}
	return nil
}
