package richtext

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestControlledMediaAndSanitizing(t *testing.T) {
	resolve := func(_ context.Context, id int64) (Media, error) {
		if id == 1 {
			return Media{"image", "/uploads/root/one.png"}, nil
		}
		if id == 2 {
			return Media{"video", "/uploads/root/two.mp4"}, nil
		}
		return Media{}, errors.New("unknown")
	}
	input := Document{HTML: `<p onclick="evil()">Hello<script>alert(1)</script><strong>World</strong></p><img data-media-id="1"><video data-media-id="2" autoplay></video><a href="javascript:evil()">bad</a>`, MediaIDs: []string{"1", "2"}}
	clean, err := Sanitize(context.Background(), input, resolve)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"onclick", "script", "alert", "javascript", "autoplay"} {
		if strings.Contains(clean.HTML, bad) {
			t.Fatal("unsafe HTML retained", bad)
		}
	}
	if !strings.Contains(clean.HTML, `controls=""`) || !strings.Contains(clean.HTML, `src="/uploads/root/one.png"`) {
		t.Fatal("media not resolved")
	}
	again, err := Sanitize(context.Background(), clean, resolve)
	if err != nil || again.HTML != clean.HTML {
		t.Fatal("sanitization is not stable", err)
	}
	for _, bad := range []Document{{HTML: `<img src="https://example.com/a.png">`}, {HTML: `<img data-media-id="1" src="https://example.com/a.png">`, MediaIDs: []string{"1"}}, {HTML: `<video data-media-id="1"></video>`, MediaIDs: []string{"1"}}, {HTML: `<img data-media-id="3">`, MediaIDs: []string{"3"}}, {HTML: `<img data-media-id="1">`}, {HTML: `<p>text</p>`, MediaIDs: []string{"1"}}} {
		if _, err := Sanitize(context.Background(), bad, resolve); err == nil {
			t.Fatal("invalid controlled node accepted", bad)
		}
	}
}
