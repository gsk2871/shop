package admin

import (
	"context"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	biz "peptide-site/internal/biz/admin"
	"peptide-site/internal/pkg/errdef"
	"regexp"
	"strings"
	"time"
)

type UserWrite struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	RoleID      int64  `json:"roleId,string"`
	Password    string `json:"password"`
	Enabled     bool   `json:"enabled"`
}
type RoleWrite struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func (r *Repo) RoleGrants(ctx context.Context, id int64) ([]string, error) {
	values := []string{}
	err := r.DB.WithContext(ctx).Table("admin_role_permission AS rp").Select("p.permission_code").Joins("JOIN admin_permission p ON p.id=rp.permission_id").Where("rp.role_id=? AND p.permission_code IS NOT NULL", id).Order("p.permission_code").Scan(&values).Error
	return values, err
}
func manageRole(tx *gorm.DB, i *biz.Identity, id int64) (biz.Role, error) {
	var role biz.Role
	if err := tx.Table("admin_role").Where("id=? AND deleted_at IS NULL", id).Take(&role).Error; err != nil {
		return role, errdef.New(404, "NOT_FOUND")
	}
	if i.Role.IsSuper {
		return role, nil
	}
	if role.IsSuper {
		return role, errdef.Denied()
	}
	var permissions []string
	if err := tx.Table("admin_role_permission rp").Select("p.permission_code").Joins("JOIN admin_permission p ON p.id=rp.permission_id").Where("rp.role_id=? AND p.permission_code IS NOT NULL", id).Scan(&permissions).Error; err != nil {
		return role, err
	}
	for _, p := range permissions {
		if !i.Allows(p) {
			return role, errdef.Denied()
		}
	}
	return role, nil
}
func protectSuper(tx *gorm.DB, target biz.User, role biz.Role, remaining bool) error {
	if !target.Enabled || !role.Enabled || !role.IsSuper || remaining {
		return nil
	}
	var count int64
	if err := tx.Table("admin_user u").Joins("JOIN admin_role r ON r.id=u.role_id").Where("u.enabled AND u.deleted_at IS NULL AND r.enabled AND r.is_super AND r.deleted_at IS NULL").Count(&count).Error; err != nil {
		return err
	}
	if count <= 1 {
		return errdef.New(409, "LAST_SUPER_ADMIN")
	}
	return nil
}
func manageLock(tx *gorm.DB) error { return tx.Exec("SELECT pg_advisory_xact_lock(784296316)").Error }
func (r *Repo) Users(ctx context.Context, page, size int) ([]biz.User, int64, error) {
	rows := []biz.User{}
	var count int64
	q := r.DB.WithContext(ctx).Table("admin_user").Where("deleted_at IS NULL")
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	for j := range rows {
		rows[j].TOTPBound = rows[j].TotpCipher != nil
	}
	return rows, count, err
}
func (r *Repo) User(ctx context.Context, id int64) (biz.User, error) {
	var row biz.User
	err := r.DB.WithContext(ctx).Table("admin_user").Where("id=? AND deleted_at IS NULL", id).Take(&row).Error
	row.TOTPBound = row.TotpCipher != nil
	return row, err
}
func (r *Repo) Roles(ctx context.Context, page, size int) ([]biz.Role, int64, error) {
	rows := []biz.Role{}
	var count int64
	q := r.DB.WithContext(ctx).Table("admin_role").Where("deleted_at IS NULL")
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, count, err
}
func (r *Repo) WriteUser(ctx context.Context, i *biz.Identity, id int64, in UserWrite, m Meta) (biz.User, error) {
	if strings.TrimSpace(in.DisplayName) == "" || len(in.DisplayName) > 100 || in.RoleID <= 0 {
		return biz.User{}, errdef.Invalid()
	}
	if id == 0 && (!regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,100}$`).MatchString(in.Username) || len(in.Password) < 12 || len(in.Password) > 72) {
		return biz.User{}, errdef.Invalid()
	}
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := manageLock(tx); err != nil {
			return err
		}
		role, err := manageRole(tx, i, in.RoleID)
		if err != nil {
			return err
		}
		if !role.Enabled {
			return errdef.Conflict()
		}
		if id == 0 {
			if role.IsSuper && !i.Role.IsSuper {
				return errdef.Denied()
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			if err := tx.Raw("INSERT INTO admin_user(username,display_name,password_hash,role_id,enabled,must_change_pwd,created_by,updated_by) VALUES(?,?,?,?,?,true,?,?) RETURNING id", in.Username, in.DisplayName, string(hash), in.RoleID, in.Enabled, i.User.ID, i.User.ID).Scan(&id).Error; err != nil {
				return err
			}
			return r.audit(tx, m, "user.create", "user", id, "success")
		}
		var target biz.User
		if err := tx.Table("admin_user").Where("id=? AND deleted_at IS NULL", id).Take(&target).Error; err != nil {
			return errdef.New(404, "NOT_FOUND")
		}
		oldRole, err := manageRole(tx, i, target.RoleID)
		if err != nil {
			return err
		}
		if (in.Username != "" && in.Username != target.Username) || in.Password != "" {
			return errdef.Invalid()
		}
		if target.ID == i.User.ID && in.RoleID != target.RoleID {
			return errdef.Denied()
		}
		if target.Enabled != in.Enabled && !i.Allows("user.disable") {
			return errdef.Denied()
		}
		if err := protectSuper(tx, target, oldRole, in.Enabled && role.IsSuper && role.Enabled); err != nil {
			return err
		}
		changed := target.RoleID != in.RoleID || target.Enabled != in.Enabled
		if err := tx.Exec("UPDATE admin_user SET display_name=?,role_id=?,enabled=?,auth_version=auth_version+?,updated_at=now(),updated_by=? WHERE id=?", in.DisplayName, in.RoleID, in.Enabled, boolInt(changed), i.User.ID, id).Error; err != nil {
			return err
		}
		if changed {
			if err := revoke(tx, id); err != nil {
				return err
			}
		}
		return r.audit(tx, m, "user.update", "user", id, "success")
	})
	if err != nil {
		return biz.User{}, err
	}
	return r.User(ctx, id)
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func (r *Repo) UserAction(ctx context.Context, i *biz.Identity, id int64, action, password, reason string, enabled bool, m Meta) error {
	if len(reason) > 500 {
		return errdef.Invalid()
	}
	if (action == "reset-password" || action == "reset-totp") && (!i.Recent(r.Now()) || strings.TrimSpace(reason) == "") {
		return errdef.New(403, "REAUTHENTICATION_REQUIRED")
	}
	if action == "reset-password" && (len(password) < 12 || len(password) > 72) {
		return errdef.Invalid()
	}
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := manageLock(tx); err != nil {
			return err
		}
		var u biz.User
		if err := tx.Table("admin_user").Where("id=? AND deleted_at IS NULL", id).Take(&u).Error; err != nil {
			return errdef.New(404, "NOT_FOUND")
		}
		role, err := manageRole(tx, i, u.RoleID)
		if err != nil {
			return err
		}
		if action == "delete" || action == "status" {
			if err := protectSuper(tx, u, role, action == "status" && enabled); err != nil {
				return err
			}
		}
		switch action {
		case "delete":
			err = tx.Exec("UPDATE admin_user SET deleted_at=now(),auth_version=auth_version+1,updated_at=now(),updated_by=? WHERE id=?", i.User.ID, id).Error
		case "status":
			err = tx.Exec("UPDATE admin_user SET enabled=?,auth_version=auth_version+1,updated_at=now(),updated_by=? WHERE id=?", enabled, i.User.ID, id).Error
		case "reset-password":
			hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if e != nil {
				return e
			}
			err = tx.Exec("UPDATE admin_user SET password_hash=?,must_change_pwd=true,auth_version=auth_version+1,updated_at=now(),updated_by=? WHERE id=?", string(hash), i.User.ID, id).Error
		case "reset-totp":
			err = tx.Exec("UPDATE admin_user SET totp_key_id=NULL,totp_cipher=NULL,auth_version=auth_version+1,updated_at=now(),updated_by=? WHERE id=?", i.User.ID, id).Error
		default:
			return errdef.Invalid()
		}
		if err != nil {
			return err
		}
		if err := revoke(tx, id); err != nil {
			return err
		}
		if err := r.audit(tx, m, "user."+action, "user", id, "success"); err != nil {
			return err
		}
		if reason != "" {
			changes, _ := jsonReason(reason)
			return tx.Exec("UPDATE audit_log SET changes=?::jsonb WHERE id=currval(pg_get_serial_sequence('audit_log','id'))", changes).Error
		}
		return nil
	})
}
func jsonReason(reason string) (string, error) {
	b, err := json.Marshal(map[string]string{"reason": reason})
	return string(b), err
}
func (r *Repo) WriteRole(ctx context.Context, i *biz.Identity, id int64, in RoleWrite, m Meta) (biz.Role, error) {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 {
		return biz.Role{}, errdef.Invalid()
	}
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := manageLock(tx); err != nil {
			return err
		}
		if id == 0 {
			if !regexp.MustCompile(`^[a-z][a-z0-9_]{0,99}$`).MatchString(in.Code) {
				return errdef.Invalid()
			}
			if err := tx.Raw("INSERT INTO admin_role(code,name,is_super,enabled,created_by,updated_by) VALUES(?,?,false,?,?,?) RETURNING id", in.Code, in.Name, in.Enabled, i.User.ID, i.User.ID).Scan(&id).Error; err != nil {
				return err
			}
			return r.audit(tx, m, "role.create", "role", id, "success")
		}
		role, err := manageRole(tx, i, id)
		if err != nil {
			return err
		}
		if role.IsSuper && !in.Enabled {
			return errdef.New(409, "LAST_SUPER_ADMIN")
		}
		if in.Code != "" && in.Code != role.Code {
			return errdef.Invalid()
		}
		if err := tx.Exec("UPDATE admin_role SET name=?,enabled=?,updated_at=now(),updated_by=? WHERE id=?", in.Name, in.Enabled, i.User.ID, id).Error; err != nil {
			return err
		}
		if role.Enabled != in.Enabled {
			if err := revokeRole(tx, id); err != nil {
				return err
			}
		}
		return r.audit(tx, m, "role.update", "role", id, "success")
	})
	var role biz.Role
	if err == nil {
		err = r.DB.WithContext(ctx).Table("admin_role").Where("id=?", id).Take(&role).Error
	}
	return role, err
}
func revokeRole(tx *gorm.DB, id int64) error {
	if err := tx.Exec("UPDATE admin_user SET auth_version=auth_version+1 WHERE role_id=? AND deleted_at IS NULL", id).Error; err != nil {
		return err
	}
	for _, table := range []string{"admin_session", "admin_mfa_ticket", "admin_totp_pending"} {
		if err := tx.Exec("DELETE FROM "+table+" WHERE user_id IN (SELECT id FROM admin_user WHERE role_id=?)", id).Error; err != nil {
			return err
		}
	}
	return nil
}
func (r *Repo) DeleteRole(ctx context.Context, i *biz.Identity, id int64, m Meta) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := manageLock(tx); err != nil {
			return err
		}
		role, err := manageRole(tx, i, id)
		if err != nil {
			return err
		}
		if role.IsSuper {
			return errdef.Denied()
		}
		var count int64
		if err := tx.Table("admin_user").Where("role_id=? AND deleted_at IS NULL", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errdef.Conflict()
		}
		if err := tx.Exec("UPDATE admin_role SET deleted_at=now(),updated_at=now(),updated_by=? WHERE id=?", i.User.ID, id).Error; err != nil {
			return err
		}
		return r.audit(tx, m, "role.delete", "role", id, "success")
	})
}
func (r *Repo) Grant(ctx context.Context, i *biz.Identity, id int64, permissions []string, m Meta) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := manageLock(tx); err != nil {
			return err
		}
		role, err := manageRole(tx, i, id)
		if err != nil {
			return err
		}
		if role.IsSuper {
			return errdef.Denied()
		}
		seen := map[string]bool{}
		for _, p := range permissions {
			if seen[p] || !i.Allows(p) {
				return errdef.Denied()
			}
			seen[p] = true
			var count int64
			if err := tx.Table("admin_permission").Where("permission_code=?", p).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errdef.Invalid()
			}
		}
		if err := tx.Exec("DELETE FROM admin_role_permission WHERE role_id=?", id).Error; err != nil {
			return err
		}
		for _, p := range permissions {
			if err := tx.Exec("INSERT INTO admin_role_permission(role_id,permission_id) SELECT ?,id FROM admin_permission WHERE permission_code=?", id, p).Error; err != nil {
				return err
			}
		}
		if err := revokeRole(tx, id); err != nil {
			return err
		}
		return r.audit(tx, m, "role.grant", "role", id, "success")
	})
}

type Audit struct {
	ID         int64     `json:"id,string"`
	UserID     *int64    `json:"userId,string"`
	Action     string    `json:"action"`
	TargetType string    `json:"targetType"`
	TargetID   string    `json:"targetId"`
	Result     string    `json:"result"`
	RequestID  string    `json:"requestId"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (r *Repo) Audits(ctx context.Context, page, size int) ([]Audit, int64, error) {
	rows := []Audit{}
	var total int64
	q := r.DB.WithContext(ctx).Table("audit_log")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

// Recovery is available only to the server command and writes a system actor audit.
func (r *Repo) Recover(ctx context.Context, username, reason string) error {
	if strings.TrimSpace(reason) == "" || len(reason) > 500 {
		return errdef.Invalid()
	}
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := manageLock(tx); err != nil {
			return err
		}
		var u biz.User
		if err := tx.Table("admin_user").Where("username=? AND deleted_at IS NULL", username).Take(&u).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE admin_user SET totp_key_id=NULL,totp_cipher=NULL,auth_version=auth_version+1,updated_at=now(),updated_by=-1 WHERE id=?", u.ID).Error; err != nil {
			return err
		}
		if err := revoke(tx, u.ID); err != nil {
			return err
		}
		if err := r.audit(tx, Meta{ActorID: -1}, "user.totp.recover", "user", u.ID, "success"); err != nil {
			return err
		}
		changes, _ := jsonReason(reason)
		return tx.Exec("UPDATE audit_log SET changes=?::jsonb WHERE id=currval(pg_get_serial_sequence('audit_log','id'))", changes).Error
	})
}

func (r *Repo) Role(ctx context.Context, id int64) (biz.Role, error) {
	var role biz.Role
	err := r.DB.WithContext(ctx).Table("admin_role").Where("id=? AND deleted_at IS NULL", id).Take(&role).Error
	return role, err
}
