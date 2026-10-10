package branding

import (
	"net/mail"
	"net/url"
	"peptide-site/internal/pkg/errdef"
	"regexp"
	"strings"
	"time"
)

type Theme struct {
	Primary    string `json:"primary"`
	Background string `json:"background"`
	Surface    string `json:"surface"`
	Text       string `json:"text"`
	Muted      string `json:"muted"`
	Border     string `json:"border"`
	Accent     string `json:"accent"`
	OnPrimary  string `json:"onPrimary"`
}
type Settings struct {
	Brand           string `json:"brand"`
	LogoMediaID     string `json:"logoMediaId"`
	Logo            string `json:"logo"`
	Theme           Theme  `json:"theme"`
	Description     string `json:"description"`
	Email           string `json:"email"`
	WhatsApp        string `json:"whatsapp"`
	Disclaimer      string `json:"disclaimer"`
	Version         int64  `json:"version"`
	ExpectedVersion int64  `json:"expectedVersion,omitempty"`
}

func Default() Settings {
	return Settings{Brand: "Research Supply", Theme: Theme{"#174e42", "#f6f5ef", "#ffffff", "#1c2825", "#65706b", "#d9ddd4", "#d39d49", "#ffffff"}, Description: "Materials and documentation for laboratory research.", Disclaimer: "For laboratory and research use only. Not for human or clinical consumption.", Version: 1}
}

var color = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var phone = regexp.MustCompile(`^$|^[0-9]{7,15}$`)

func (s Settings) Validate() error {
	if strings.TrimSpace(s.Brand) == "" || len(s.Brand) > 100 || len(s.Description) > 5000 || len(s.Disclaimer) > 500 || !phone.MatchString(s.WhatsApp) {
		return errdef.Invalid()
	}
	if s.Email != "" {
		a, e := mail.ParseAddress(s.Email)
		if e != nil || a.Address != s.Email {
			return errdef.Invalid()
		}
	}
	for _, c := range []string{s.Theme.Primary, s.Theme.Background, s.Theme.Surface, s.Theme.Text, s.Theme.Muted, s.Theme.Border, s.Theme.Accent, s.Theme.OnPrimary} {
		if !color.MatchString(c) {
			return errdef.Invalid()
		}
	}
	return nil
}

type Banner struct {
	ID                 string     `json:"id,omitempty"`
	SlotCode           string     `json:"slotCode"`
	Title              string     `json:"title"`
	Subtitle           string     `json:"subtitle"`
	ImageMediaID       string     `json:"imageMediaId"`
	MobileImageMediaID string     `json:"mobileImageMediaId"`
	Image              string     `json:"image"`
	MobileImage        string     `json:"mobileImage"`
	CTAText            string     `json:"ctaText"`
	Link               string     `json:"link"`
	Sort               int        `json:"sort"`
	Enabled            bool       `json:"enabled"`
	StartsAt           *time.Time `json:"startsAt"`
	EndsAt             *time.Time `json:"endsAt"`
	Version            int64      `json:"version"`
	ExpectedVersion    int64      `json:"expectedVersion,omitempty"`
}

func (b Banner) Validate() error {
	if strings.TrimSpace(b.Title) == "" || len(b.Title) > 200 || len(b.Subtitle) > 2000 || len(b.CTAText) > 100 || len(b.Link) > 2000 || !safeURL(b.Link) || b.ImageMediaID == "" || b.StartsAt != nil && b.EndsAt != nil && !b.EndsAt.After(*b.StartsAt) {
		return errdef.Invalid()
	}
	return nil
}
func Public(b Banner) map[string]any {
	return map[string]any{"id": b.ID, "title": b.Title, "subtitle": b.Subtitle, "image": b.Image, "mobileImage": b.MobileImage, "ctaText": b.CTAText, "link": b.Link}
}

func safeURL(s string) bool {
	if s == "" {
		return true
	}
	if strings.ContainsAny(s, "\r\n\\") {
		return false
	}
	u, e := url.Parse(s)
	return e == nil && u.User == nil && ((strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "//") && u.Host == "") || (u.Scheme == "https" && u.Host != ""))
}
