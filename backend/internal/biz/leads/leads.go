package leads

import (
	"net/mail"
	"peptide-site/internal/pkg/errdef"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Enabled            bool   `json:"enabled"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	Placeholder        string `json:"placeholder"`
	SubmitText         string `json:"submitText"`
	DismissText        string `json:"dismissText"`
	Notice             string `json:"notice"`
	SuccessTitle       string `json:"successTitle"`
	SuccessDescription string `json:"successDescription"`
	Version            int64  `json:"version"`
	ExpectedVersion    int64  `json:"expectedVersion,omitempty"`
}

func Default() Config {
	return Config{Enabled: true, Title: "Get 10% OFF your order", Description: "Sign up and unlock your discount.", Placeholder: "Email or WhatsApp (include country code)", SubmitText: "Claim discount", DismissText: "No, thanks", Notice: "By submitting your contact details, you agree to be contacted by email or WhatsApp about your inquiry and discount. You can ask us to stop contacting you at any time.", SuccessTitle: "Thank you!", SuccessDescription: "We've received your contact details. Our team will contact you by email or WhatsApp to help with your inquiry and confirm your 10% discount.", Version: 1}
}
func (c Config) Validate() error {
	for _, s := range []string{c.Title, c.Placeholder, c.SubmitText, c.DismissText, c.SuccessTitle} {
		if strings.TrimSpace(s) == "" || len(s) > 200 {
			return errdef.Invalid()
		}
	}
	for _, s := range []string{c.Description, c.Notice, c.SuccessDescription} {
		if strings.TrimSpace(s) == "" || len(s) > 2000 {
			return errdef.Invalid()
		}
	}
	return nil
}
func Public(c Config) map[string]any {
	return map[string]any{"enabled": c.Enabled, "title": c.Title, "description": c.Description, "placeholder": c.Placeholder, "submitText": c.SubmitText, "dismissText": c.DismissText, "notice": c.Notice, "successTitle": c.SuccessTitle, "successDescription": c.SuccessDescription}
}

var phone = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
var separators = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")

func Normalize(raw string) (kind, contact string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 254 || strings.ContainsAny(raw, "\r\n\x00\t") {
		return "", "", errdef.Invalid()
	}
	if strings.Contains(raw, "@") {
		address, e := mail.ParseAddress(raw)
		if e != nil || address.Address != raw || !strings.Contains(strings.SplitN(raw, "@", 2)[1], ".") {
			return "", "", errdef.Invalid()
		}
		return "email", strings.ToLower(raw), nil
	}
	normalized := separators.Replace(raw)
	if !phone.MatchString(normalized) {
		return "", "", errdef.Invalid()
	}
	return "whatsapp", normalized, nil
}

type Lead struct {
	ID               string    `json:"id"`
	ContactType      string    `json:"contactType"`
	Contact          string    `json:"contact"`
	FirstSubmittedAt time.Time `json:"firstSubmittedAt"`
	LastSubmittedAt  time.Time `json:"lastSubmittedAt"`
	SubmissionCount  int64     `json:"submissionCount"`
	Status           string    `json:"status"`
	Note             string    `json:"note"`
	Version          int64     `json:"version"`
}
type Update struct {
	Status          string `json:"status"`
	Note            string `json:"note"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

func (u Update) Validate() error {
	if (u.Status != "pending" && u.Status != "completed" && u.Status != "invalid") || len(u.Note) > 2000 || u.ExpectedVersion < 1 {
		return errdef.Invalid()
	}
	return nil
}

type Filter struct {
	Contact    string
	Start, End *time.Time
	Page, Size int
}
