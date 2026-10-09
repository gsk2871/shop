// Package cms defines the independent site's content contracts.
package cms

import (
	"context"
	"errors"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid content")

// Banner configures a homepage slide.
type Banner struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Image    string `json:"image"`
	Link     string `json:"link"`
}

// Settings contains public website branding and contact settings.
type Settings struct {
	Brand           string   `json:"brand"`
	Logo            string   `json:"logo"`
	PrimaryColor    string   `json:"primaryColor"`
	BackgroundColor string   `json:"backgroundColor"`
	Banners         []Banner `json:"banners"`
	Description     string   `json:"description"`
	Email           string   `json:"email"`
	WhatsApp        string   `json:"whatsapp"`
	Disclaimer      string   `json:"disclaimer"`
}

// DefaultSettings provides neutral content without unverified product claims.
func DefaultSettings() Settings {
	return Settings{Brand: "Research Supply", PrimaryColor: "#174e42", BackgroundColor: "#f6f5ef", Banners: []Banner{{Title: "Research starts with the right materials.", Subtitle: "Explore our laboratory research catalogue and request documentation.", Link: "/contact"}}, Description: "Materials and documentation for laboratory research.", Disclaimer: "For laboratory and research use only. Not for human or clinical consumption."}
}

// Article stores an English editorial entry. Body is plain text, never executable HTML.
type Article struct {
	ID             int64     `json:"id"`
	Title          string    `json:"title"`
	Slug           string    `json:"slug"`
	Summary        string    `json:"summary"`
	Body           string    `json:"body"`
	Cover          string    `json:"cover"`
	Status         string    `json:"status"`
	SEOTitle       string    `json:"seoTitle"`
	SEODescription string    `json:"seoDescription"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Media describes an uploaded public asset.
type Media struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	MIME      string    `json:"mime"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

// Admin is the single site's administrator.
type Admin struct {
	ID           int64
	Username     string
	PasswordHash string
}

// Audit records an administrator mutation without sensitive payloads.
type Audit struct {
	ID        int64     `json:"id"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"createdAt"`
}

// Repository abstracts PostgreSQL persistence for content and authentication.
type Repository interface {
	Settings(context.Context) (Settings, error)
	SaveSettings(context.Context, Settings) error
	Articles(context.Context, bool) ([]Article, error)
	Article(context.Context, string) (Article, error)
	SaveArticle(context.Context, *Article) error
	DeleteArticle(context.Context, int64) error
	Products(context.Context, bool, string) ([]Product, error)
	Product(context.Context, string) (Product, error)
	SaveProduct(context.Context, *Product) error
	DeleteProduct(context.Context, int64) error
	Media(context.Context) ([]Media, error)
	SaveMedia(context.Context, *Media) error
	Admin(context.Context) (Admin, error)
	Password(context.Context, string) error
	CreateSession(context.Context, string, time.Time) error
	Session(context.Context, string) (bool, error)
	DeleteSession(context.Context, string) error
	Audit(context.Context, string) error
	Audits(context.Context) ([]Audit, error)
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// SafeURL accepts same-origin paths and HTTPS addresses only.
func SafeURL(s string) bool {
	if s == "" {
		return true
	}
	if strings.ContainsAny(s, "\r\n\\") {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if strings.HasPrefix(s, "/") {
		return !strings.HasPrefix(s, "//") && u.Host == ""
	}
	return u.Scheme == "https" && u.Host != "" && u.User == nil
}

// ValidateSettings prevents unsafe links and CSS injection.
func ValidateSettings(s Settings) error {
	if strings.TrimSpace(s.Brand) == "" || len(s.Brand) > 100 || !colorPattern.MatchString(s.PrimaryColor) || !colorPattern.MatchString(s.BackgroundColor) || !SafeURL(s.Logo) || len(s.Banners) > 3 || len(s.Disclaimer) > 500 {
		return ErrInvalid
	}
	if s.Email != "" {
		address, err := mail.ParseAddress(s.Email)
		if err != nil || address.Address != s.Email {
			return ErrInvalid
		}
	}
	if len(s.Description) > 5000 {
		return ErrInvalid
	}
	if s.WhatsApp != "" {
		for _, c := range s.WhatsApp {
			if c < '0' || c > '9' {
				return ErrInvalid
			}
		}
		if len(s.WhatsApp) < 7 || len(s.WhatsApp) > 15 {
			return ErrInvalid
		}
	}
	for _, b := range s.Banners {
		if strings.TrimSpace(b.Title) == "" || len(b.Title) > 200 || len(b.Subtitle) > 1000 || !SafeURL(b.Image) || !SafeURL(b.Link) {
			return ErrInvalid
		}
	}
	return nil
}

// ValidateArticle validates publication fields independently of transport.
func ValidateArticle(a Article) error {
	if strings.TrimSpace(a.Title) == "" || len(a.Title) > 200 || len(a.Slug) > 120 || !slugPattern.MatchString(a.Slug) || len(a.Body) > 100000 || len(a.Summary) > 2000 || !SafeURL(a.Cover) {
		return ErrInvalid
	}
	switch a.Status {
	case "draft", "published", "offline":
		return nil
	}
	return ErrInvalid
}
