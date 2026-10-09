package cms

import "testing"

func TestSafeURL(t *testing.T) {
	for _, tc := range []struct {
		url string
		ok  bool
	}{{"", true}, {"/contact", true}, {"https://example.com/a", true}, {"javascript:alert(1)", false}, {"//example.com", false}, {"/\\example.com", false}, {"https://u:p@example.com", false}, {"data:text/html,x", false}} {
		t.Run(tc.url, func(t *testing.T) {
			if SafeURL(tc.url) != tc.ok {
				t.Fatalf("unexpected result for %q", tc.url)
			}
		})
	}
}
func TestValidateSettings(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Settings)
		ok     bool
	}{{"default", func(*Settings) {}, true}, {"blank brand", func(s *Settings) { s.Brand = "" }, false}, {"css injection", func(s *Settings) { s.PrimaryColor = "red;display:none" }, false}, {"unsafe link", func(s *Settings) { s.Banners[0].Link = "javascript:x" }, false}, {"too many", func(s *Settings) { s.Banners = make([]Banner, 4) }, false}, {"bad phone", func(s *Settings) { s.WhatsApp = "+1?x=y" }, false}} {
		t.Run(tc.name, func(t *testing.T) {
			s := DefaultSettings()
			tc.change(&s)
			if (ValidateSettings(s) == nil) != tc.ok {
				t.Fatal("unexpected validation")
			}
		})
	}
}
func TestValidateArticle(t *testing.T) {
	for _, tc := range []struct {
		slug, status string
		ok           bool
	}{{"lab-notes", "draft", true}, {"lab-notes", "published", true}, {"lab-notes", "offline", true}, {"../admin", "draft", false}, {"lab-notes", "review", false}} {
		t.Run(tc.slug+tc.status, func(t *testing.T) {
			a := Article{Title: "Notes", Slug: tc.slug, Status: tc.status}
			if (ValidateArticle(a) == nil) != tc.ok {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestInvalidEmail(t *testing.T) {
	s := DefaultSettings()
	s.Email = "lab@example.com?bcc=other@example.com"
	if ValidateSettings(s) == nil {
		t.Fatal("unsafe email accepted")
	}
	s.Email = "lab@example.com"
	if err := ValidateSettings(s); err != nil {
		t.Fatal(err)
	}
}
