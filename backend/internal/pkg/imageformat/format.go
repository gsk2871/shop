// Package imageformat identifies image formats without decoding pixels or dimensions.
// Upload permission is controlled separately by storage.allowed_types.
package imageformat

import (
	"mime"
	"slices"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

func Lookup(value string) *mimetype.MIME {
	if !strings.HasPrefix(value, "image/") {
		return nil
	}
	return mimetype.Lookup(value)
}

func Extension(value string) string {
	if m := Lookup(value); m != nil {
		return strings.TrimPrefix(m.Extension(), ".")
	}
	return ""
}

// Extensions includes common filename aliases, independent of the upload whitelist.
func Extensions(value string) []string {
	m := Lookup(value)
	if m == nil {
		return nil
	}
	out := []string{m.Extension()}
	aliases, _ := mime.ExtensionsByType(value)
	out = append(out, aliases...)
	switch m.String() {
	case "image/jpeg":
		out = append(out, ".jpeg", ".jpe")
	case "image/tiff":
		out = append(out, ".tif")
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func Matches(body []byte, expected string) bool {
	if Lookup(expected) == nil {
		return false
	}
	// Image subformats such as APNG remain valid for their image container (PNG).
	for detected := mimetype.Detect(body); detected != nil && strings.HasPrefix(detected.String(), "image/"); detected = detected.Parent() {
		if detected.Is(expected) {
			return true
		}
	}
	return false
}
