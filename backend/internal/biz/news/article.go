// Package news defines versioned English editorial content.
package news

import (
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/richtext"
	"regexp"
	"strings"
)

type Article struct {
	ID               string            `json:"id,omitempty"`
	Title            string            `json:"title"`
	Slug             string            `json:"slug"`
	Summary          string            `json:"summary"`
	Body             richtext.Document `json:"body"`
	CoverMediaID     string            `json:"coverMediaId"`
	Cover            string            `json:"cover"`
	Status           string            `json:"status"`
	SEOTitle         string            `json:"seoTitle"`
	SEODescription   string            `json:"seoDescription"`
	Pinned           bool              `json:"pinned"`
	Sort             int               `json:"sort"`
	Version          int64             `json:"version"`
	ExpectedVersion  int64             `json:"expectedVersion,omitempty"`
	FirstPublishedAt *string           `json:"firstPublishedAt"`
	UpdatedAt        string            `json:"updatedAt"`
}

var slug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func Validate(a Article) error {
	if strings.TrimSpace(a.Title) == "" || len(a.Title) > 200 || !slug.MatchString(a.Slug) || len(a.Slug) > 120 || len(a.Summary) > 2000 || len(a.SEOTitle) > 200 || len(a.SEODescription) > 5000 {
		return errdef.Invalid()
	}
	if a.Status != "draft" && a.Status != "published" && a.Status != "offline" {
		return errdef.Invalid()
	}
	return nil
}
func Public(a Article) map[string]any {
	return map[string]any{"id": a.ID, "title": a.Title, "slug": a.Slug, "summary": a.Summary, "body": a.Body, "cover": a.Cover, "seoTitle": a.SEOTitle, "seoDescription": a.SEODescription, "firstPublishedAt": a.FirstPublishedAt}
}
