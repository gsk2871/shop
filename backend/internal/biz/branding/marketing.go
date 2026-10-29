package branding

import (
	"peptide-site/internal/pkg/errdef"
	"strings"
)

type AnnouncementItem struct {
	Text    string `json:"text"`
	Enabled bool   `json:"enabled"`
}
type Announcement struct {
	Enabled    bool               `json:"enabled"`
	Background string             `json:"background"`
	TextColor  string             `json:"textColor"`
	Speed      string             `json:"speed"`
	Items      []AnnouncementItem `json:"items"`
}
type BrandPart struct {
	Text  string `json:"text"`
	Color string `json:"color"`
}
type BrandMetric struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}
type BrandIntro struct {
	Tagline     string        `json:"tagline"`
	Parts       []BrandPart   `json:"parts"`
	Description string        `json:"description"`
	Metrics     []BrandMetric `json:"metrics"`
}

func DefaultAnnouncement() Announcement {
	return Announcement{Background: "#2563eb", TextColor: "#ffffff", Speed: "standard", Items: []AnnouncementItem{
		{"99% Purity", true}, {"10% off for the first order", true}, {"Super affordable price", true},
		{"150+ Countries Served", true}, {"24H Response Time", true}, {"Contact us right now", true},
	}}
}
func DefaultBrandIntro() BrandIntro {
	return BrandIntro{Tagline: "Manufacturer Direct • Research Supply", Parts: []BrandPart{{"NOVA", "#2563eb"}, {"PEPT", "#111827"}},
		Description: "Premium Research Peptides for laboratories and research institutions worldwide",
		Metrics:     []BrandMetric{{"≥99%", "Purity Guaranteed"}, {"24h", "Response Time"}, {"150+", "Countries Served"}, {"COA", "Every Batch Verified"}}}
}
func (s Settings) validateMarketing() error {
	if s.HeaderBackground != "" && !color.MatchString(s.HeaderBackground) {
		return errdef.Invalid()
	}
	if a := s.Announcement; a != nil {
		if !color.MatchString(a.Background) || !color.MatchString(a.TextColor) || (a.Speed != "slow" && a.Speed != "standard" && a.Speed != "fast") || len(a.Items) > 30 {
			return errdef.Invalid()
		}
		for _, item := range a.Items {
			if strings.TrimSpace(item.Text) == "" || len(item.Text) > 500 {
				return errdef.Invalid()
			}
		}
	}
	if b := s.BrandIntro; b != nil {
		if len(b.Tagline) > 200 || len(b.Description) > 2000 || len(b.Parts) > 10 || len(b.Metrics) > 20 {
			return errdef.Invalid()
		}
		for _, part := range b.Parts {
			if strings.TrimSpace(part.Text) == "" || len(part.Text) > 100 || !color.MatchString(part.Color) {
				return errdef.Invalid()
			}
		}
		for _, metric := range b.Metrics {
			if strings.TrimSpace(metric.Value) == "" || len(metric.Value) > 100 || strings.TrimSpace(metric.Description) == "" || len(metric.Description) > 200 {
				return errdef.Invalid()
			}
		}
	}
	return nil
}

// NormalizeMarketing keeps empty collections stable across JSON round trips.
func (s *Settings) NormalizeMarketing() {
	if s.Announcement != nil && s.Announcement.Items == nil {
		s.Announcement.Items = []AnnouncementItem{}
	}
	if s.BrandIntro != nil {
		if s.BrandIntro.Parts == nil {
			s.BrandIntro.Parts = []BrandPart{}
		}
		if s.BrandIntro.Metrics == nil {
			s.BrandIntro.Metrics = []BrandMetric{}
		}
	}
}
