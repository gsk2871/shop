package admin

import (
	"encoding/json"
	"gorm.io/gorm"
	biz "peptide-site/internal/biz/branding"
)

// SeedMarketing adds only missing fields; an intentionally empty list is preserved.
// The same advisory lock as SaveSettings serializes seeding with first-time edits.
func SeedMarketing(tx *gorm.DB) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(784296317)").Error; err != nil {
		return err
	}
	defaults := biz.Default()
	payload, err := json.Marshal(defaults)
	if err != nil {
		return err
	}
	var initial map[string]json.RawMessage
	if err = json.Unmarshal(payload, &initial); err != nil {
		return err
	}
	for _, key := range []string{"headerBackground", "announcement", "brandIntro"} {
		delete(initial, key)
	}
	payload, err = json.Marshal(initial)
	if err != nil {
		return err
	}
	if err = tx.Exec("INSERT INTO site_config(id,payload,version) VALUES(1,?::jsonb,1) ON CONFLICT(id) DO NOTHING", string(payload)).Error; err != nil {
		return err
	}
	for _, field := range []struct {
		Key   string
		Value any
	}{
		{"headerBackground", "#eaf4fc"}, {"announcement", biz.DefaultAnnouncement()}, {"brandIntro", biz.DefaultBrandIntro()},
	} {
		value, err := json.Marshal(field.Value)
		if err != nil {
			return err
		}
		if err = tx.Exec("UPDATE site_config SET payload=payload || jsonb_build_object(?::text,?::jsonb),version=version+1 WHERE id=1 AND NOT jsonb_exists(payload,?::text)", field.Key, string(value), field.Key).Error; err != nil {
			return err
		}
	}
	return nil
}
