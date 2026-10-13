package media

import (
	"bytes"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path/filepath"
	"peptide-site/internal/pkg/errdef"
	"strings"
	"unicode/utf8"
)

var imageExtensions = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp", "image/gif": "gif"}

func ImageExtension(mime string) string { return imageExtensions[mime] }
func ValidImageName(name, mime string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	return ImageExtension(mime) != "" && (ext == ImageExtension(mime) || (mime == "image/jpeg" && ext == "jpeg"))
}
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 200 {
		return errdef.Invalid()
	}
	return nil
}
func InspectImage(body []byte, expectedMIME string, maxBytes int64) (int, int, error) {
	if len(body) == 0 || int64(len(body)) > maxBytes || ImageExtension(expectedMIME) == "" || http.DetectContentType(body) != expectedMIME {
		return 0, 0, errdef.New(400, "MEDIA_FILE_INVALID")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 20000000 {
		return 0, 0, errdef.New(400, "MEDIA_FILE_INVALID")
	}
	if _, _, err := image.Decode(bytes.NewReader(body)); err != nil {
		return 0, 0, errdef.New(400, "MEDIA_FILE_INVALID")
	}
	return cfg.Width, cfg.Height, nil
}
