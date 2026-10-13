// Package contentslug generates stable website paths for newly created content.
package contentslug

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

var separators = regexp.MustCompile(`[^a-z0-9]+`)

func New(title string) (string, error) {
	base := strings.Trim(separators.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if len(base) > 80 {
		base = strings.TrimRight(base[:80], "-")
	}
	if base == "" {
		base = "content"
	}
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	return base + "-" + hex.EncodeToString(suffix[:]), nil
}
