package leads

import (
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	for _, tt := range []struct{ raw, kind, value string }{
		{" Research@Example.com ", "email", "research@example.com"},
		{"+1 (202) 555-0123", "whatsapp", "+12025550123"},
		{"+8613800000000", "whatsapp", "+8613800000000"},
	} {
		kind, value, err := Normalize(tt.raw)
		if err != nil || kind != tt.kind || value != tt.value {
			t.Fatalf("normalization: %q %q %v", kind, value, err)
		}
	}
	for _, raw := range []string{"", "Name <user@example.com>", "user@localhost", "a@@example.com", "13800000000", "+01234567", "+1234567890123456", "+1abc2025550123", "a@example.com\r\nBcc:x@example.com", strings.Repeat("a", 255)} {
		if _, _, err := Normalize(raw); err == nil {
			t.Fatalf("accepted invalid input %q", raw)
		}
	}
}
func TestConfigAndStatusValidation(t *testing.T) {
	c := Default()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, exists := Public(c)["version"]; exists {
		t.Fatal("management version exposed")
	}
	c.Notice = strings.Repeat("a", 2001)
	if c.Validate() == nil {
		t.Fatal("unbounded notice")
	}
	for _, status := range []string{"pending", "completed", "invalid"} {
		if (Update{Status: status, ExpectedVersion: 1}).Validate() != nil {
			t.Fatal(status)
		}
	}
	for _, u := range []Update{{Status: "in_progress", ExpectedVersion: 1}, {Status: "pending"}, {Status: "invalid", ExpectedVersion: 1, Note: strings.Repeat("x", 2001)}} {
		if u.Validate() == nil {
			t.Fatal("invalid update accepted")
		}
	}
}
