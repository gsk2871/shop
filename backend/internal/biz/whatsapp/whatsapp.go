// Package whatsapp defines country routing configuration and transport contracts.
package whatsapp

import (
	"peptide-site/internal/pkg/errdef"
	"regexp"
	"strings"
	"time"
)

type Account struct {
	ID              string    `json:"id,omitempty"`
	Name            string    `json:"name"`
	PhoneNumber     string    `json:"phoneNumber"`
	QRCodeMediaID   string    `json:"qrCodeMediaId"`
	QRCodeURL       string    `json:"qrCodeUrl,omitempty"`
	Enabled         bool      `json:"enabled"`
	AllocationCount int64     `json:"allocationCount"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
type AccountInput struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name"`
	PhoneNumber   string `json:"phoneNumber"`
	QRCodeMediaID string `json:"qrCodeMediaId"`
	Enabled       bool   `json:"enabled"`
}
type Input struct {
	Name              string         `json:"name"`
	CountryCodes      []string       `json:"countryCodes"`
	Enabled           bool           `json:"enabled"`
	ExpectedUpdatedAt string         `json:"expectedUpdatedAt,omitempty"`
	Accounts          []AccountInput `json:"accounts"`
}
type Group struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	CountryCodes          []string  `json:"countryCodes"`
	Enabled               bool      `json:"enabled"`
	UpdatedAt             time.Time `json:"updatedAt"`
	IsDefault             bool      `json:"isDefault"`
	PendingSetup          bool      `json:"pendingSetup"`
	AccountCount          int       `json:"accountCount"`
	AvailableAccountCount int       `json:"availableAccountCount"`
	Accounts              []Account `json:"accounts,omitempty"`
}
type List struct {
	Items            []Group   `json:"items"`
	Total            int64     `json:"total"`
	Page             int       `json:"page"`
	PageSize         int       `json:"pageSize"`
	DefaultGroupID   string    `json:"defaultGroupId"`
	DefaultGroupName string    `json:"defaultGroupName"`
	RoutingUpdatedAt time.Time `json:"routingUpdatedAt"`
	GeoIPAvailable   bool      `json:"geoipAvailable"`
}
type DefaultInput struct {
	GroupID                string `json:"groupId"`
	ExpectedUpdatedAt      string `json:"expectedUpdatedAt"`
	ExpectedGroupUpdatedAt string `json:"expectedGroupUpdatedAt"`
}
type DefaultResult struct {
	DefaultGroupID   string    `json:"defaultGroupId"`
	RoutingUpdatedAt time.Time `json:"routingUpdatedAt"`
}
type Contact struct {
	AccountName string `json:"accountName"`
	PhoneNumber string `json:"phoneNumber"`
	QRCodeURL   string `json:"qrCodeUrl"`
	URL         string `json:"url"`
}

var Phone = regexp.MustCompile(`^[1-9][0-9]{6,14}$`)
var Country = regexp.MustCompile(`^[A-Z]{2}$`)

const MaxAccounts = 100
const MaxCountries = 250

func (v *Input) Validate() error {
	v.Name = strings.TrimSpace(v.Name)
	if v.Name == "" || len(v.Name) > 200 || len(v.CountryCodes) == 0 || len(v.CountryCodes) > MaxCountries || len(v.Accounts) > MaxAccounts {
		return errdef.Invalid()
	}
	countries := map[string]bool{}
	phones := map[string]bool{}
	ids := map[string]bool{}
	for _, c := range v.CountryCodes {
		if !Country.MatchString(c) || countries[c] {
			return errdef.Invalid()
		}
		countries[c] = true
	}
	for j := range v.Accounts {
		a := &v.Accounts[j]
		a.Name = strings.TrimSpace(a.Name)
		if a.Name == "" || len(a.Name) > 100 || !Phone.MatchString(a.PhoneNumber) || a.QRCodeMediaID == "" {
			return errdef.Invalid()
		}
		if phones[a.PhoneNumber] {
			return errdef.New(409, "DUPLICATE_WHATSAPP_ACCOUNT")
		}
		phones[a.PhoneNumber] = true
		if a.ID != "" {
			if ids[a.ID] {
				return errdef.Invalid()
			}
			ids[a.ID] = true
		}
	}
	return nil
}

// MatchTimestamp requires the exact database microsecond value. It never accepts
// a missing timestamp or lets a millisecond-truncated client bypass the guard.
func MatchTimestamp(raw string, actual time.Time) error {
	expected, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return errdef.Invalid()
	}
	if !expected.Equal(actual) {
		return errdef.New(409, "UPDATE_CONFLICT")
	}
	return nil
}
