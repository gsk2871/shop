package news

import (
	"context"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/news"
	"peptide-site/internal/pkg/richtext"
	"peptide-site/internal/testutil"
	"strconv"
	"testing"
)

func TestPublicationVersionAndPreview(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"article.publish"}})
	a, e := r.Save(ctx, 0, biz.Article{Title: "Research", Slug: "research", Status: "draft", Body: richtext.Document{HTML: "<p>Safe</p><script>bad</script>"}})
	if e != nil {
		t.Fatal(e)
	}
	id, _ := strconv.ParseInt(a.ID, 10, 64)
	if _, e = r.Public(ctx, a.Slug); e == nil {
		t.Fatal("draft public")
	}
	a, e = r.Status(ctx, id, a.Version, "published")
	if e != nil || a.FirstPublishedAt == nil {
		t.Fatal(e, a)
	}
	first := *a.FirstPublishedAt
	a, e = r.Status(ctx, id, a.Version, "offline")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.Public(ctx, a.Slug); e == nil {
		t.Fatal("offline public")
	}
	a, e = r.Status(ctx, id, a.Version, "published")
	if e != nil || *a.FirstPublishedAt != first {
		t.Fatal("first publication changed", e)
	}
	public, e := r.Public(ctx, a.Slug)
	if e != nil || public["status"] != nil || a.Body.HTML != "<p>Safe</p>" {
		t.Fatal(e, public)
	}
	a.ExpectedVersion = a.Version
	a.Title = "Preview"
	if _, e = r.Preview(ctx, a); e != nil {
		t.Fatal(e)
	}
	saved, e := r.Get(ctx, id)
	if e != nil || saved.Title == a.Title || saved.Version != a.Version {
		t.Fatal("preview persisted", e)
	}
	a.ExpectedVersion--
	if _, e = r.Save(ctx, id, a); e == nil {
		t.Fatal("stale save")
	}
	if e = r.Delete(ctx, id, saved.Version); e != nil {
		t.Fatal(e)
	}
	if _, e = r.Public(ctx, a.Slug); e == nil {
		t.Fatal("deleted public")
	}
}
