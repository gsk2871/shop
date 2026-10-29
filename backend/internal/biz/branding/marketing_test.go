package branding

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarketingValidation(t *testing.T) {
	a, b := DefaultAnnouncement(), DefaultBrandIntro()
	s := Default()
	s.Brand = "Test"
	s.HeaderBackground = "#eaf4fc"
	s.Announcement = &a
	s.BrandIntro = &b
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Settings){
		func(s *Settings) { s.HeaderBackground = "url(https://example.invalid)" },
		func(s *Settings) { s.Announcement.Speed = "unbounded" },
		func(s *Settings) { s.Announcement.Items = []AnnouncementItem{{Text: " "}} },
		func(s *Settings) { s.BrandIntro.Parts[0].Color = "red;display:none" },
		func(s *Settings) { s.BrandIntro.Metrics = []BrandMetric{{Value: "1"}} },
	} {
		a, b = DefaultAnnouncement(), DefaultBrandIntro()
		candidate := s
		candidate.Announcement = &a
		candidate.BrandIntro = &b
		mutate(&candidate)
		if candidate.Validate() == nil {
			t.Fatal("invalid marketing input accepted")
		}
	}
}

func TestMarketingEmptyCollections(t *testing.T) {
	s := Default()
	if err := json.Unmarshal([]byte(`{"announcement":{"items":null},"brandIntro":{}}`), &s); err != nil {
		t.Fatal(err)
	}
	s.NormalizeMarketing()
	body, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"items":[]`, `"parts":[]`, `"metrics":[]`} {
		if !strings.Contains(string(body), expected) {
			t.Fatal("unstable collections", string(body))
		}
	}
}
