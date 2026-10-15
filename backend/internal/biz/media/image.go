package media

import (
	"path/filepath"
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/imageformat"
	"slices"
	"strings"
	"unicode/utf8"
)

func ImageExtension(value string) string { return imageformat.Extension(value) }
func ValidImageName(name, value string) bool {
	return slices.Contains(imageformat.Extensions(value), strings.ToLower(filepath.Ext(name)))
}
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 200 {
		return errdef.Invalid()
	}
	return nil
}

// InspectImage checks size and content type only; it does not decode image pixels or dimensions.
func InspectImage(body []byte, expectedMIME string, maxBytes int64) error {
	if len(body) == 0 || int64(len(body)) > maxBytes || !imageformat.Matches(body, expectedMIME) {
		return errdef.New(400, "MEDIA_FILE_INVALID")
	}
	return nil
}
