package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	biz "peptide-site/internal/biz/admin"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	"peptide-site/internal/pkg/admintotp"
	"peptide-site/internal/pkg/errdef"
	"strings"
	"time"
)

type Repo struct {
	DB     *gorm.DB
	Config conf.Config
	Codec  *admintotp.Codec
	Now    func() time.Time
}
type Meta struct {
	ActorID   int64
	RequestID string
	IP        string
}

func NewRepo(d *data.Data, c conf.Config) (*Repo, error) {
	codec, err := admintotp.New(c.Auth.TOTP.ActiveKeyID, c.Auth.TOTP.Keys)
	if err != nil {
		return nil, err
	}
	r := &Repo{DB: d.DB, Config: c, Codec: codec, Now: time.Now}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r, r.CheckKeys(ctx)
}
func (r *Repo) CheckKeys(ctx context.Context) error {
	var rows []struct {
		KeyID  string
		Cipher string
	}
	if err := r.DB.WithContext(ctx).Raw("SELECT totp_key_id AS key_id,totp_cipher AS cipher FROM admin_user WHERE totp_cipher IS NOT NULL UNION ALL SELECT key_id,cipher FROM admin_totp_pending").Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := r.Codec.Decrypt(row.KeyID, row.Cipher); err != nil {
			return err
		}
	}
	return nil
}

// Password-only tokens are tagged before hashing so clients cannot remove the
// marker to keep a session valid after TOTP is re-enabled.
const passwordOnlyPrefix = "p:"

func Hash(value string) string { s := sha256.Sum256([]byte(value)); return hex.EncodeToString(s[:]) }
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func (r *Repo) audit(tx *gorm.DB, m Meta, action, target string, id any, result string) error {
	b, _ := json.Marshal(map[string]any{})
	return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id,result,request_id,ip,changes) VALUES(?,?,?,?,?,?,?,?::jsonb)", m.ActorID, action, target, fmt.Sprint(id), result, m.RequestID, m.IP, string(b)).Error
}
func (r *Repo) AuditFailure(ctx context.Context, m Meta, action string) error {
	return r.audit(r.DB.WithContext(ctx), m, action, "auth", "", "failure")
}
func (r *Repo) Rate(ctx context.Context, key string) error {
	now := r.Now().UTC()
	var count int
	err := r.DB.WithContext(ctx).Raw("INSERT INTO admin_rate_limit(key_hash,window_start,attempts,updated_at) VALUES(?,?,1,?) ON CONFLICT(key_hash) DO UPDATE SET attempts=CASE WHEN admin_rate_limit.window_start <= ? THEN 1 ELSE admin_rate_limit.attempts+1 END,window_start=CASE WHEN admin_rate_limit.window_start <= ? THEN EXCLUDED.window_start ELSE admin_rate_limit.window_start END,updated_at=EXCLUDED.updated_at RETURNING attempts", Hash(key), now, now, now.Add(-time.Minute), now.Add(-time.Minute)).Scan(&count).Error
	if err != nil {
		return err
	}
	if count > r.Config.Auth.LoginLimit {
		return errdef.New(429, "RATE_LIMITED")
	}
	return nil
}
func activeUser(tx *gorm.DB, id int64) (biz.User, biz.Role, error) {
	var u biz.User
	err := tx.Table("admin_user").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND enabled AND deleted_at IS NULL", id).Take(&u).Error
	if err != nil {
		return u, biz.Role{}, err
	}
	var role biz.Role
	err = tx.Table("admin_role").Where("id=? AND enabled AND deleted_at IS NULL", u.RoleID).Take(&role).Error
	return u, role, err
}
func (r *Repo) Identity(ctx context.Context, token string) (*biz.Identity, error) {
	if r.Config.Auth.TOTP.Enabled && strings.HasPrefix(token, passwordOnlyPrefix) {
		return nil, errdef.New(401, "AUTHENTICATION_REQUIRED")
	}
	var session struct {
		UserID, AuthVersion int64
		Scope               string
		VerifiedAt          *time.Time
	}
	err := r.DB.WithContext(ctx).Table("admin_session").Where("token_hash=? AND expires_at>?", Hash(token), r.Now()).Take(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errdef.New(401, "AUTHENTICATION_REQUIRED")
	}
	if err != nil {
		return nil, err
	}
	var u biz.User
	err = r.DB.WithContext(ctx).Table("admin_user").Where("id=? AND auth_version=? AND enabled AND deleted_at IS NULL", session.UserID, session.AuthVersion).Take(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errdef.New(401, "AUTHENTICATION_REQUIRED")
	}
	if err != nil {
		return nil, err
	}
	var role biz.Role
	err = r.DB.WithContext(ctx).Table("admin_role").Where("id=? AND enabled AND deleted_at IS NULL", u.RoleID).Take(&role).Error
	if err != nil {
		return nil, errdef.New(401, "AUTHENTICATION_REQUIRED")
	}
	permissions := []string{}
	err = r.DB.WithContext(ctx).Table("admin_role_permission AS rp").Select("p.permission_code").Joins("JOIN admin_permission p ON p.id=rp.permission_id").Where("rp.role_id=? AND p.permission_code IS NOT NULL", role.ID).Order("p.permission_code").Scan(&permissions).Error
	if err != nil {
		return nil, err
	}
	u.TOTPBound = u.TotpCipher != nil
	scope := session.Scope
	if !r.Config.Auth.TOTP.Enabled {
		scope = "full"
		if u.MustChangePwd {
			scope = "password_change"
		}
	}
	identity := &biz.Identity{TOTPEnabled: r.Config.Auth.TOTP.Enabled, User: u, Role: role, Scope: scope, VerifiedAt: session.VerifiedAt, SessionHash: Hash(token), Permissions: permissions}
	identity.Menu, err = r.Tree(ctx, permissions, false)
	return identity, err
}
func (r *Repo) Tree(ctx context.Context, permissions []string, all bool) ([]biz.Permission, error) {
	nodes := []biz.Permission{}
	if err := r.DB.WithContext(ctx).Table("admin_permission").Order("sort,id").Find(&nodes).Error; err != nil {
		return nil, err
	}
	grant := map[string]bool{}
	for _, p := range permissions {
		grant[p] = true
	}
	var build func(*int64) []biz.Permission
	build = func(parent *int64) []biz.Permission {
		children := []biz.Permission{}
		for _, n := range nodes {
			if n.Code == "menu.overview" {
				continue
			}
			if n.Code == "menu.whatsapp" {
				n.Name = "客户配置"
			}
			if (parent == nil && n.ParentID != nil) || (parent != nil && (n.ParentID == nil || *n.ParentID != *parent)) {
				continue
			}
			n.Children = build(&n.ID)
			allowed := all || (n.PermissionCode != nil && grant[*n.PermissionCode]) || (n.Visibility != nil && strings.HasPrefix(*n.Visibility, "authenticated")) || len(n.Children) > 0
			if allowed {
				children = append(children, n)
			}
		}
		return children
	}
	return build(nil), nil
}

type LoginResult struct {
	Step      string `json:"step"`
	Ticket    string `json:"ticket,omitempty"`
	Token     string `json:"-"`
	Scope     string `json:"scope,omitempty"`
	ExpiresIn int    `json:"expiresIn"`
}

func (r *Repo) createSession(tx *gorm.DB, u biz.User, scope string, verified *time.Time) (LoginResult, error) {
	token, err := randomToken()
	if err != nil {
		return LoginResult{}, err
	}
	if !r.Config.Auth.TOTP.Enabled && verified == nil {
		token = passwordOnlyPrefix + token[len(passwordOnlyPrefix):]
	}
	ttl := r.Config.Auth.SessionTTL
	if scope != "full" {
		ttl = r.Config.Auth.OnboardingTTL
	}
	err = tx.Exec("INSERT INTO admin_session(token_hash,user_id,scope,auth_version,expires_at,verified_at) VALUES(?,?,?,?,?,?)", Hash(token), u.ID, scope, u.AuthVersion, r.Now().Add(ttl), verified).Error
	return LoginResult{Step: "session", Token: token, Scope: scope, ExpiresIn: int(ttl.Seconds())}, err
}
func (r *Repo) Login(ctx context.Context, username, password string, m Meta) (LoginResult, error) {
	var result LoginResult
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct{ ID int64 }
		if err := tx.Table("admin_user").Select("id").Where("username=? AND deleted_at IS NULL", username).Take(&row).Error; err != nil {
			return errdef.New(401, "INVALID_CREDENTIALS")
		}
		u, _, err := activeUser(tx, row.ID)
		if err != nil {
			return errdef.New(401, "INVALID_CREDENTIALS")
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
			return errdef.New(401, "INVALID_CREDENTIALS")
		}
		m.ActorID = u.ID
		if !r.Config.Auth.TOTP.Enabled {
			scope := "full"
			if u.MustChangePwd {
				scope = "password_change"
			}
			result, err = r.createSession(tx, u, scope, nil)
			if err == nil {
				err = tx.Exec("UPDATE admin_user SET last_login_at=? WHERE id=?", r.Now().UTC(), u.ID).Error
			}
		} else if u.TotpCipher == nil {
			result, err = r.createSession(tx, u, "onboarding", nil)
		} else {
			var token string
			token, err = randomToken()
			if err == nil {
				err = tx.Exec("INSERT INTO admin_mfa_ticket(token_hash,user_id,auth_version,expires_at) VALUES(?,?,?,?)", Hash(token), u.ID, u.AuthVersion, r.Now().Add(r.Config.Auth.MFATicketTTL)).Error
				result = LoginResult{Step: "totp", Ticket: token, ExpiresIn: int(r.Config.Auth.MFATicketTTL.Seconds())}
			}
		}
		if err != nil {
			return err
		}
		return r.audit(tx, m, "auth.password", "user", u.ID, "success")
	})
	return result, err
}
func (r *Repo) useCode(tx *gorm.DB, u biz.User, code string) (bool, error) {
	if u.TotpCipher == nil || u.TotpKeyID == nil {
		return false, nil
	}
	secret, err := r.Codec.Decrypt(*u.TotpKeyID, *u.TotpCipher)
	if err != nil {
		return false, err
	}
	step, ok := admintotp.Match(secret, code, r.Now())
	if !ok {
		return false, nil
	}
	res := tx.Exec("INSERT INTO admin_totp_used(user_id,timestep) VALUES(?,?) ON CONFLICT DO NOTHING", u.ID, step)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}
func (r *Repo) LoginTOTP(ctx context.Context, ticket, code string, m Meta) (LoginResult, error) {
	var result LoginResult
	var failure error
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidate struct{ UserID int64 }
		if err := tx.Table("admin_mfa_ticket").Select("user_id").Where("token_hash=?", Hash(ticket)).Take(&candidate).Error; err != nil {
			return errdef.New(401, "MFA_TICKET_INVALID")
		}
		u, _, err := activeUser(tx, candidate.UserID)
		if err != nil {
			return errdef.New(401, "MFA_TICKET_INVALID")
		}
		var row struct {
			AuthVersion int64
			Attempts    int
			ExpiresAt   time.Time
			ConsumedAt  *time.Time
		}
		if err := tx.Table("admin_mfa_ticket").Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash=?", Hash(ticket)).Take(&row).Error; err != nil {
			return err
		}
		if row.ConsumedAt != nil || !row.ExpiresAt.After(r.Now()) || row.AuthVersion != u.AuthVersion || row.Attempts >= r.Config.Auth.MFAMaxAttempts {
			return errdef.New(401, "MFA_TICKET_INVALID")
		}
		ok, err := r.useCode(tx, u, code)
		if err != nil {
			return err
		}
		m.ActorID = u.ID
		if !ok {
			if err := tx.Exec("UPDATE admin_mfa_ticket SET attempts=attempts+1 WHERE token_hash=?", Hash(ticket)).Error; err != nil {
				return err
			}
			failure = errdef.New(401, "MFA_INVALID")
			return r.audit(tx, m, "auth.totp", "user", u.ID, "failure")
		}
		now := r.Now().UTC()
		if err := tx.Exec("UPDATE admin_mfa_ticket SET consumed_at=? WHERE token_hash=?", now, Hash(ticket)).Error; err != nil {
			return err
		}
		scope := "full"
		if u.MustChangePwd {
			scope = "password_change"
		}
		result, err = r.createSession(tx, u, scope, &now)
		if err != nil {
			return err
		}
		if err := tx.Exec("UPDATE admin_user SET last_login_at=? WHERE id=?", now, u.ID).Error; err != nil {
			return err
		}
		return r.audit(tx, m, "auth.login", "user", u.ID, "success")
	})
	if err == nil {
		err = failure
	}
	return result, err
}
func revoke(tx *gorm.DB, id int64) error {
	if err := tx.Exec("DELETE FROM admin_session WHERE user_id=?", id).Error; err != nil {
		return err
	}
	if err := tx.Exec("DELETE FROM admin_mfa_ticket WHERE user_id=?", id).Error; err != nil {
		return err
	}
	return tx.Exec("DELETE FROM admin_totp_pending WHERE user_id=?", id).Error
}
func (r *Repo) Password(ctx context.Context, i *biz.Identity, current, next string, m Meta) error {
	if len(next) < 12 || len(next) > 72 {
		return errdef.Invalid()
	}
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u, _, err := activeUser(tx, i.User.ID)
		if err != nil || u.AuthVersion != i.User.AuthVersion {
			return errdef.New(401, "AUTHENTICATION_REQUIRED")
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)) != nil {
			return errdef.New(401, "INVALID_CREDENTIALS")
		}
		if current == next {
			return errdef.Invalid()
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := tx.Exec("UPDATE admin_user SET password_hash=?,must_change_pwd=false,auth_version=auth_version+1,updated_at=now(),updated_by=? WHERE id=?", string(hash), u.ID, u.ID).Error; err != nil {
			return err
		}
		if err := revoke(tx, u.ID); err != nil {
			return err
		}
		return r.audit(tx, m, "user.password", "user", u.ID, "success")
	})
}
func (r *Repo) Setup(ctx context.Context, i *biz.Identity, m Meta) (string, string, error) {
	secret, err := admintotp.Secret()
	if err != nil {
		return "", "", err
	}
	id, encrypted, err := r.Codec.Encrypt(secret)
	if err != nil {
		return "", "", err
	}
	err = r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u, _, err := activeUser(tx, i.User.ID)
		if err != nil || u.AuthVersion != i.User.AuthVersion {
			return errdef.New(401, "AUTHENTICATION_REQUIRED")
		}
		if u.MustChangePwd || u.TotpCipher != nil {
			return errdef.Conflict()
		}
		if err := tx.Exec("INSERT INTO admin_totp_pending(user_id,auth_version,key_id,cipher,expires_at) VALUES(?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET auth_version=EXCLUDED.auth_version,key_id=EXCLUDED.key_id,cipher=EXCLUDED.cipher,expires_at=EXCLUDED.expires_at", u.ID, u.AuthVersion, id, encrypted, r.Now().Add(r.Config.Auth.OnboardingTTL)).Error; err != nil {
			return err
		}
		return r.audit(tx, m, "auth.totp.setup", "user", u.ID, "success")
	})
	return secret, admintotp.URI(r.Config.Auth.TOTP.Issuer, i.User.Username, secret), err
}
func (r *Repo) Activate(ctx context.Context, i *biz.Identity, code string, m Meta) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u, _, err := activeUser(tx, i.User.ID)
		if err != nil || u.AuthVersion != i.User.AuthVersion {
			return errdef.New(401, "AUTHENTICATION_REQUIRED")
		}
		if u.MustChangePwd || u.TotpCipher != nil {
			return errdef.Conflict()
		}
		var pending struct {
			AuthVersion int64
			KeyID       string
			Cipher      string
			ExpiresAt   time.Time
		}
		if err := tx.Table("admin_totp_pending").Where("user_id=?", u.ID).Take(&pending).Error; err != nil {
			return errdef.New(401, "MFA_INVALID")
		}
		if pending.AuthVersion != u.AuthVersion || !pending.ExpiresAt.After(r.Now()) {
			return errdef.New(401, "MFA_INVALID")
		}
		temporary := u
		temporary.TotpKeyID = &pending.KeyID
		temporary.TotpCipher = &pending.Cipher
		ok, err := r.useCode(tx, temporary, code)
		if err != nil {
			return err
		}
		if !ok {
			return errdef.New(401, "MFA_INVALID")
		}
		if err := tx.Exec("UPDATE admin_user SET totp_key_id=?,totp_cipher=?,auth_version=auth_version+1,updated_at=now(),updated_by=? WHERE id=?", pending.KeyID, pending.Cipher, u.ID, u.ID).Error; err != nil {
			return err
		}
		if err := revoke(tx, u.ID); err != nil {
			return err
		}
		return r.audit(tx, m, "auth.totp.activate", "user", u.ID, "success")
	})
}
func (r *Repo) Reauthenticate(ctx context.Context, i *biz.Identity, password, code string, m Meta) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u, _, err := activeUser(tx, i.User.ID)
		if err != nil || u.AuthVersion != i.User.AuthVersion {
			return errdef.New(401, "AUTHENTICATION_REQUIRED")
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
			return errdef.New(401, "INVALID_CREDENTIALS")
		}
		if r.Config.Auth.TOTP.Enabled {
			ok, err := r.useCode(tx, u, code)
			if err != nil {
				return err
			}
			if !ok {
				return errdef.New(401, "MFA_INVALID")
			}
		}
		if err := tx.Exec("UPDATE admin_session SET verified_at=? WHERE token_hash=?", r.Now(), i.SessionHash).Error; err != nil {
			return err
		}
		return r.audit(tx, m, "auth.reauthenticate", "user", u.ID, "success")
	})
}
func (r *Repo) Logout(ctx context.Context, i *biz.Identity) error {
	return r.DB.WithContext(ctx).Exec("DELETE FROM admin_session WHERE token_hash=?", i.SessionHash).Error
}

// Cleanup uses conservative retention and never removes records within the replay window.
func (r *Repo) Cleanup(ctx context.Context) error {
	now := r.Now()
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, q := range []struct {
			SQL    string
			Before time.Time
		}{{"DELETE FROM admin_mfa_ticket WHERE expires_at < ?", now.Add(-time.Hour)}, {"DELETE FROM admin_totp_pending WHERE expires_at < ?", now.Add(-time.Hour)}, {"DELETE FROM admin_totp_used WHERE created_at < ?", now.Add(-24 * time.Hour)}, {"DELETE FROM admin_session WHERE expires_at < ?", now}, {"DELETE FROM admin_rate_limit WHERE updated_at < ?", now.Add(-24 * time.Hour)}} {
			if err := tx.Exec(q.SQL, q.Before).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
