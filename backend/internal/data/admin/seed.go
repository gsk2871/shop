// Package admin implements transactional administrative persistence.
package admin

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	"peptide-site/internal/data/seed"
	whatsapp "peptide-site/internal/data/whatsapp"
	"time"
)

// Seed initializes only missing system records. Existing user credentials,
// states, MFA bindings and role assignment are never changed.
func Seed(ctx context.Context, d *data.Data, c conf.Seed) error {
	registry, err := seed.Load()
	if err != nil {
		return err
	}
	return d.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(784296316)").Error; err != nil {
			return err
		}
		if tx.Migrator().HasTable("currency") {
			if err := tx.Exec("INSERT INTO currency(code,minor_unit,symbol) VALUES('USD',2,'$') ON CONFLICT(code) DO NOTHING").Error; err != nil {
				return err
			}
		}
		if tx.Migrator().HasTable("banner_slot") {
			if err := tx.Exec("INSERT INTO banner_slot(code,name) VALUES('home_hero','首页主视觉'),('products_header','产品页顶部'),('research_header','文章页顶部') ON CONFLICT(code) DO NOTHING").Error; err != nil {
				return err
			}
		}
		if tx.Migrator().HasTable("whatsapp_group") {
			if err := whatsapp.Seed(tx); err != nil {
				return err
			}
		}
		ids := map[string]int64{}
		pending := append([]seed.Node(nil), registry.Nodes...)
		for len(pending) > 0 {
			next := []seed.Node{}
			for _, n := range pending {
				var parent any
				if n.ParentCode != nil {
					value, ok := ids[*n.ParentCode]
					if !ok {
						next = append(next, n)
						continue
					}
					parent = value
				}
				var id int64
				if err := tx.Raw("INSERT INTO admin_permission(code,parent_id,name,node_type,route,permission_code,sort,visibility) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(code) DO UPDATE SET parent_id=EXCLUDED.parent_id,name=EXCLUDED.name,node_type=EXCLUDED.node_type,route=EXCLUDED.route,permission_code=EXCLUDED.permission_code,sort=EXCLUDED.sort,visibility=EXCLUDED.visibility,updated_at=now() RETURNING id", n.Code, parent, n.Name, n.Type, n.Route, n.PermissionCode, n.Sort, n.Visibility).Scan(&id).Error; err != nil {
					return err
				}
				ids[n.Code] = id
			}
			if len(next) == len(pending) {
				return errors.New("permission tree cannot be ordered")
			}
			pending = next
		}
		var superID int64
		for _, role := range registry.Roles {
			var id int64
			result := tx.Raw("INSERT INTO admin_role(code,name,is_super) VALUES(?,?,?) ON CONFLICT(code) WHERE deleted_at IS NULL DO NOTHING RETURNING id", role.Code, role.Name, role.IsSuper).Scan(&id)
			if result.Error != nil {
				return result.Error
			}
			fresh := id != 0
			if !fresh {
				if err := tx.Table("admin_role").Select("id").Where("code=? AND deleted_at IS NULL", role.Code).Scan(&id).Error; err != nil {
					return err
				}
			}
			var storedSuper bool
			if err := tx.Table("admin_role").Select("is_super").Where("id=?", id).Scan(&storedSuper).Error; err != nil {
				return err
			}
			if storedSuper != role.IsSuper {
				return errors.New("reserved role kind conflicts with seed")
			}
			if role.IsSuper {
				superID = id
			}
			// Super administrators gain missing registered permissions; ordinary roles
			// retain their existing explicit grants. Neither path removes grants.
			if fresh || role.IsSuper {
				for _, permission := range role.Permissions {
					if err := tx.Exec("INSERT INTO admin_role_permission(role_id,permission_id) SELECT ?,id FROM admin_permission WHERE permission_code=? ON CONFLICT DO NOTHING", id, permission).Error; err != nil {
						return err
					}
				}
			}
		}
		var existing struct {
			ID        int64
			RoleID    int64
			DeletedAt *time.Time
		}
		result := tx.Table("admin_user").Where("username=?", c.SuperAdmin.Username).Order("id DESC").Limit(1).Scan(&existing)
		if result.Error != nil {
			return result.Error
		}
		if existing.ID != 0 {
			if existing.RoleID != superID || existing.DeletedAt != nil {
				return errors.New("seed administrator username conflicts with an existing account")
			}
			return nil
		}
		if len(c.SuperAdmin.Password) < 12 || len(c.SuperAdmin.Password) > 72 {
			return errors.New("SEED_ADMIN_PASSWORD or seed.super_admin.password must contain 12 to 72 bytes")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(c.SuperAdmin.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		return tx.Exec("INSERT INTO admin_user(username,display_name,password_hash,role_id,enabled,must_change_pwd) VALUES(?,?,?,?,true,true)", c.SuperAdmin.Username, "超级管理员", string(hash), superID).Error
	})
}
