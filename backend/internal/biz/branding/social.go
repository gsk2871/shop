package branding

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"peptide-site/internal/pkg/errdef"
)

const MaxSocialAccounts = 50

type SocialAccount struct {
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Link        string `json:"link"`
	IconMediaID string `json:"iconMediaId"`
	Icon        string `json:"icon"`
	Enabled     bool   `json:"enabled"`
}

type PublicSocialAccount struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Link     string `json:"link"`
	Icon     string `json:"icon"`
}

func ValidateSocialAccounts(accounts []SocialAccount) error {
	if len(accounts) > MaxSocialAccounts {
		return errdef.Invalid()
	}
	for _, account := range accounts {
		if strings.TrimSpace(account.Name) == "" || utf8.RuneCountInString(account.Name) > 100 || account.IconMediaID == "" {
			return errdef.Invalid()
		}
		switch account.Platform {
		case "facebook", "tiktok", "youtube", "x", "instagram", "linkedin", "other":
		default:
			return errdef.Invalid()
		}
		u, err := url.Parse(account.Link)
		if err != nil || len(account.Link) > 2000 || strings.ContainsAny(account.Link, "\r\n\t\\ ") || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return errdef.Invalid()
		}
	}
	return nil
}

func PublicSocialAccounts(accounts []SocialAccount) []PublicSocialAccount {
	result := make([]PublicSocialAccount, 0, len(accounts))
	for _, account := range accounts {
		if account.Enabled {
			result = append(result, PublicSocialAccount{Name: account.Name, Platform: account.Platform, Link: account.Link, Icon: account.Icon})
		}
	}
	return result
}
