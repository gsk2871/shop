package media

import "testing"

func TestInspectImageFormatAndSizeOnly(t *testing.T) {
	// A PNG signature has no dimensions or decodable pixels, but meets the
	// upload contract: a recognized file header within the size limit.
	pngHeader := []byte("\x89PNG\r\n\x1a\n")
	for _, tc := range []struct {
		name    string
		body    []byte
		mime    string
		max     int64
		wantErr bool
	}{
		{"header without pixels or dimensions", pngHeader, "image/png", 8, false},
		{"empty", nil, "image/png", 8, true},
		{"oversized", pngHeader, "image/png", 7, true},
		{"mismatched format", pngHeader, "image/jpeg", 8, true},
		{"non-image content", []byte("not an image"), "image/png", 50, true},
		{"unsupported type", []byte("text"), "text/plain", 50, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := InspectImage(tc.body, tc.mime, tc.max); (err != nil) != tc.wantErr {
				t.Fatalf("InspectImage error = %v, want error = %v", err, tc.wantErr)
			}
		})
	}
}

func TestCommonImageHeaders(t *testing.T) {
	for _, tc := range []struct{ name, mime, header string }{
		{"photo.jpeg", "image/jpeg", "\xff\xd8\xff"},
		{"image.png", "image/png", "\x89PNG\r\n\x1a\n"},
		{"image.webp", "image/webp", "RIFF\x00\x00\x00\x00WEBPVP8 "},
		{"image.gif", "image/gif", "GIF89a"},
		{"image.bmp", "image/bmp", "BM"},
		{"scan.tif", "image/tiff", "II\x2a\x00"},
		{"photo.avif", "image/avif", "\x00\x00\x00\x18ftypavif\x00\x00\x00\x00avifmif1"},
		{"photo.heic", "image/heic", "\x00\x00\x00\x18ftypheic\x00\x00\x00\x00heicmif1"},
		{"photo.heif", "image/heif", "\x00\x00\x00\x18ftypmif1\x00\x00\x00\x00mif1heif"},
		{"favicon.ico", "image/x-icon", "\x00\x00\x01\x00"},
		{"logo.svg", "image/svg+xml", `<svg xmlns="http://www.w3.org/2000/svg"></svg>`},
	} {
		t.Run(tc.mime, func(t *testing.T) {
			if !ValidImageName(tc.name, tc.mime) {
				t.Fatal("extension rejected")
			}
			if err := InspectImage([]byte(tc.header), tc.mime, 1024); err != nil {
				t.Fatal(err)
			}
			if err := InspectImage([]byte("not an image"), tc.mime, 1024); err == nil {
				t.Fatal("fake content accepted")
			}
		})
	}
}

func TestAnimatedPNGUsesPNGPolicy(t *testing.T) {
	body := make([]byte, 41)
	copy(body, "\x89PNG\r\n\x1a\n")
	copy(body[37:], "acTL")
	if err := InspectImage(body, "image/png", 100); err != nil {
		t.Fatal(err)
	}
}
