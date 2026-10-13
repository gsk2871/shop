// Package richtext sanitizes the one HTML protocol used by products and news.
package richtext

import (
	"bytes"
	"context"
	"golang.org/x/net/html"
	"net/url"
	"peptide-site/internal/pkg/errdef"
	"sort"
	"strconv"
	"strings"
)

type Document struct {
	HTML     string   `json:"html"`
	MediaIDs []string `json:"mediaIds"`
}
type Media struct {
	Type string
	URL  string
}
type Resolver func(context.Context, int64) (Media, error)

// Sanitize resolves controlled nodes without storing content or references.
func Sanitize(ctx context.Context, input Document, resolve Resolver) (Document, error) {
	if len(input.HTML) > 200000 {
		return Document{}, errdef.Invalid()
	}
	parsed, err := html.Parse(strings.NewReader(input.HTML))
	if err != nil {
		return Document{}, errdef.Invalid()
	}
	root := &html.Node{Type: html.ElementNode, Data: "div"}
	refs := map[string]bool{}
	allowed := map[string]bool{"p": true, "h2": true, "h3": true, "h4": true, "strong": true, "em": true, "u": true, "ul": true, "ol": true, "li": true, "blockquote": true, "br": true, "a": true, "img": true}
	blocked := map[string]bool{"script": true, "style": true, "iframe": true, "object": true, "embed": true, "svg": true, "math": true, "template": true}
	var clean func(*html.Node, *html.Node) error
	clean = func(source, destination *html.Node) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if source.Type == html.TextNode {
			destination.AppendChild(&html.Node{Type: html.TextNode, Data: source.Data})
			return nil
		}
		if source.Type != html.ElementNode && source.Type != html.DocumentNode {
			return nil
		}
		tag := source.Data
		if tag == "b" {
			tag = "strong"
		}
		if tag == "i" {
			tag = "em"
		}
		if blocked[tag] {
			return nil
		}
		if tag == "video" {
			return errdef.Invalid()
		}
		if !allowed[tag] {
			for child := source.FirstChild; child != nil; child = child.NextSibling {
				if err := clean(child, destination); err != nil {
					return err
				}
			}
			return nil
		}
		target := &html.Node{Type: html.ElementNode, Data: tag}
		attributes := map[string]string{}
		for _, a := range source.Attr {
			if _, duplicate := attributes[a.Key]; duplicate && a.Key == "data-media-id" {
				return errdef.Invalid()
			}
			attributes[a.Key] = a.Val
		}
		if tag == "a" {
			href := attributes["href"]
			if safeLink(href) {
				target.Attr = append(target.Attr, html.Attribute{Key: "href", Val: href})
				if strings.HasPrefix(href, "https://") {
					target.Attr = append(target.Attr, html.Attribute{Key: "target", Val: "_blank"}, html.Attribute{Key: "rel", Val: "noopener noreferrer"})
				}
			}
		}
		if tag == "img" {
			raw := attributes["data-media-id"]
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id <= 0 || strconv.FormatInt(id, 10) != raw {
				return errdef.Invalid()
			}
			media, err := resolve(ctx, id)
			if err != nil {
				return err
			}
			required := "image"
			if media.Type != required || !safeMediaURL(media.URL) {
				return errdef.Invalid()
			}
			if src := attributes["src"]; src != "" && src != media.URL {
				return errdef.Invalid()
			}
			refs[raw] = true
			target.Attr = append(target.Attr, html.Attribute{Key: "data-media-id", Val: raw}, html.Attribute{Key: "src", Val: media.URL})
			alt := attributes["alt"]
			if len(alt) > 500 {
				return errdef.Invalid()
			}
			target.Attr = append(target.Attr, html.Attribute{Key: "alt", Val: alt}, html.Attribute{Key: "loading", Val: "lazy"})
		}
		destination.AppendChild(target)
		if tag != "img" && tag != "br" {
			for child := source.FirstChild; child != nil; child = child.NextSibling {
				if err := clean(child, target); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := clean(parsed, root); err != nil {
		return Document{}, err
	}
	declared := map[string]bool{}
	for _, id := range input.MediaIDs {
		declared[id] = true
	}
	if len(declared) != len(refs) {
		return Document{}, errdef.Invalid()
	}
	for id := range refs {
		if !declared[id] {
			return Document{}, errdef.Invalid()
		}
	}
	var output bytes.Buffer
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&output, child); err != nil {
			return Document{}, err
		}
	}
	ids := make([]string, 0, len(refs))
	for id := range refs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, _ := strconv.ParseInt(ids[i], 10, 64)
		b, _ := strconv.ParseInt(ids[j], 10, 64)
		return a < b
	})
	return Document{HTML: output.String(), MediaIDs: ids}, nil
}
func safeLink(raw string) bool {
	if raw == "" || strings.ContainsAny(raw, "\r\n\\") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") && u.Host == "") || (u.Scheme == "https" && u.Host != "" && u.User == nil)
}
func IDs(d Document) []int64 {
	result := make([]int64, 0, len(d.MediaIDs))
	for _, id := range d.MediaIDs {
		value, _ := strconv.ParseInt(id, 10, 64)
		result = append(result, value)
	}
	return result
}

// The resolver supplies a persisted, verified media URL; submitted src must still match it.
func safeMediaURL(raw string) bool {
	if strings.HasPrefix(raw, "/uploads/") {
		return true
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}
