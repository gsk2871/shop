// Package admin defines administrative identity and safe transport models.
package admin

import (
	"context"
	"time"
)

type User struct {
	TOTPBound     bool       `json:"totpBound" gorm:"-"`
	ID            int64      `json:"id,string"`
	Username      string     `json:"username"`
	DisplayName   string     `json:"displayName"`
	RoleID        int64      `json:"roleId,string"`
	Enabled       bool       `json:"enabled"`
	MustChangePwd bool       `json:"mustChangePwd"`
	AuthVersion   int64      `json:"-"`
	PasswordHash  string     `json:"-"`
	TotpKeyID     *string    `json:"-"`
	TotpCipher    *string    `json:"-"`
	LastLoginAt   *time.Time `json:"lastLoginAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"-"`
}
type Role struct {
	ID        int64      `json:"id,string"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	IsSuper   bool       `json:"isSuper"`
	Enabled   bool       `json:"enabled"`
	DeletedAt *time.Time `json:"-"`
}
type Permission struct {
	ID             int64        `json:"id,string"`
	Code           string       `json:"code"`
	ParentID       *int64       `json:"-"`
	Name           string       `json:"name"`
	NodeType       string       `json:"type"`
	Route          *string      `json:"route"`
	PermissionCode *string      `json:"permissionCode"`
	Sort           int          `json:"sort"`
	Visibility     *string      `json:"-"`
	Children       []Permission `json:"children" gorm:"-"`
}
type Identity struct {
	TOTPEnabled bool         `json:"totpEnabled"`
	User        User         `json:"user"`
	Role        Role         `json:"role"`
	Scope       string       `json:"scope"`
	Permissions []string     `json:"permissions"`
	Menu        []Permission `json:"menu"`
	VerifiedAt  *time.Time   `json:"-"`
	SessionHash string       `json:"-"`
}
type key struct{}

func WithIdentity(ctx context.Context, i *Identity) context.Context {
	return context.WithValue(ctx, key{}, i)
}
func Current(ctx context.Context) *Identity { i, _ := ctx.Value(key{}).(*Identity); return i }
func (i *Identity) Allows(p string) bool {
	for _, v := range i.Permissions {
		if v == p {
			return true
		}
	}
	return false
}
func (i *Identity) Recent(now time.Time) bool {
	return i.VerifiedAt != nil && now.Sub(*i.VerifiedAt) >= 0 && now.Sub(*i.VerifiedAt) < 5*time.Minute
}
