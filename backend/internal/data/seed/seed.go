// Package seed owns the versioned permission registry, shared by seeding and routing.
package seed

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed admin_permissions.json
var body []byte

type API struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}
type Node struct {
	Code           string  `json:"code"`
	ParentCode     *string `json:"parentCode"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Sort           int     `json:"sort"`
	Route          *string `json:"route"`
	PermissionCode *string `json:"permissionCode"`
	Visibility     string  `json:"visibility"`
	APIs           []API   `json:"apis"`
}
type Role struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	IsSuper     bool     `json:"isSuper"`
	Permissions []string `json:"permissions"`
}
type SelfService struct {
	Code          string   `json:"code"`
	APIs          []API    `json:"apis"`
	AllowedScopes []string `json:"allowedScopes"`
}
type Registry struct {
	SchemaVersion int           `json:"schemaVersion"`
	Nodes         []Node        `json:"nodes"`
	Roles         []Role        `json:"roles"`
	SelfService   []SelfService `json:"selfService"`
	Anonymous     []API         `json:"anonymousApis"`
}

func Load() (Registry, error) {
	var r Registry
	if err := json.Unmarshal(body, &r); err != nil {
		return r, err
	}
	return r, Validate(r)
}
func Validate(r Registry) error {
	if r.SchemaVersion != 1 {
		return fmt.Errorf("unsupported permission schema")
	}
	nodes := map[string]Node{}
	permissions := map[string]bool{}
	apis := map[string]bool{}
	register := func(a API) error {
		k := a.Method + " " + a.Path
		if apis[k] {
			return fmt.Errorf("duplicate registered route")
		}
		apis[k] = true
		return nil
	}
	for _, n := range r.Nodes {
		if n.Code == "" || nodes[n.Code].Code != "" {
			return fmt.Errorf("duplicate permission node")
		}
		nodes[n.Code] = n
		if n.PermissionCode != nil {
			if permissions[*n.PermissionCode] {
				return fmt.Errorf("duplicate permission")
			}
			permissions[*n.PermissionCode] = true
		}
		for _, a := range n.APIs {
			if err := register(a); err != nil {
				return err
			}
		}
	}
	for _, n := range r.Nodes {
		seen := map[string]bool{n.Code: true}
		parent := n.ParentCode
		for parent != nil {
			node, ok := nodes[*parent]
			if !ok || seen[*parent] {
				return fmt.Errorf("invalid permission tree")
			}
			seen[*parent] = true
			parent = node.ParentCode
		}
	}
	for _, role := range r.Roles {
		for _, p := range role.Permissions {
			if !permissions[p] {
				return fmt.Errorf("unknown role permission")
			}
		}
	}
	for _, s := range r.SelfService {
		for _, a := range s.APIs {
			if err := register(a); err != nil {
				return err
			}
		}
	}
	for _, a := range r.Anonymous {
		if err := register(a); err != nil {
			return err
		}
	}
	return nil
}
