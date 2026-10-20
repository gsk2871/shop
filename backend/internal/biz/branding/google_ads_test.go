package branding

import (
	"strings"
	"testing"
)

func TestGoogleAdsValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config GoogleAds
		valid  bool
	}{
		{"disabled empty", GoogleAds{}, true},
		{"disabled draft", GoogleAds{ID: "AW-18370098818"}, true},
		{"enabled complete", GoogleAds{true, "AW-18370098818", "AbC-D_efGhIjKL"}, true},
		{"missing label", GoogleAds{true, "AW-18370098818", ""}, false},
		{"missing id", GoogleAds{true, "", "abc"}, false},
		{"script injection", GoogleAds{true, "AW-1\" onload=alert(1)", "abc"}, false},
		{"invalid disabled config", GoogleAds{ID: "https://evil.example"}, false},
		{"full destination as label", GoogleAds{true, "AW-1", "AW-1/abc"}, false},
		{"oversize label", GoogleAds{true, "AW-1", strings.Repeat("a", 101)}, false},
		{"newline", GoogleAds{true, "AW-1", "abc\n"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.config.Validate(); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}
