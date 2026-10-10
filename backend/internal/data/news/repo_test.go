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

func TestManagementFiltersAndPublicIsolation(t *testing.T) {
	d, c := testutil.Postgres(t)
	r := NewRepo(d, c)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}, Permissions: []string{"article.publish"}})
	for _, input := range []biz.Article{
		{Title: "Alpha 100%_", Slug: "alpha", Status: "published", Pinned: true},
		{Title: "Alpha draft", Slug: "alpha-draft", Status: "draft"},
		{Title: "Beta", Slug: "beta", Status: "offline", Pinned: true},
		{Title: "Deleted", Slug: "deleted", Status: "draft"},
	} {
		a, err := r.Save(ctx, 0, input)
		if err != nil {
			t.Fatal(err)
		}
		if input.Slug == "deleted" {
			id, _ := strconv.ParseInt(a.ID, 10, 64)
			if err = r.Delete(ctx, id, a.Version); err != nil {
				t.Fatal(err)
			}
		}
	}
	yes, no := true, false
	for _, tt := range []struct {
		name, search, status string
		pinned               *bool
		public               bool
		total                int64
	}{
		{name: "all", total: 3},
		{name: "case insensitive", search: "ALPHA", total: 2},
		{name: "literal percent", search: "%", total: 1},
		{name: "literal underscore", search: "_", total: 1},
		{name: "literal quote", search: "' OR 1=1 --", total: 0},
		{name: "combined", search: "alpha", status: "published", pinned: &yes, total: 1},
		{name: "unpinned", pinned: &no, total: 1},
		{name: "offline", status: "offline", total: 1},
		{name: "no match", search: "alpha", status: "offline", total: 0},
		{name: "public ignores management filters", public: true, status: "draft", pinned: &no, search: "missing", total: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rows, total, err := r.List(ctx, tt.public, 1, 1, tt.search, tt.status, tt.pinned)
			if err != nil || total != tt.total || len(rows) != min(1, int(tt.total)) {
				t.Fatalf("%v total=%d rows=%v", err, total, rows)
			}
			if tt.public && len(rows) > 0 {
				if rows[0]["status"] != nil || rows[0]["updatedAt"] != nil {
					t.Fatal("management fields leaked")
				}
			}
		})
	}
	rows, total, err := r.List(ctx, false, 2, 1, "Alpha", "", nil)
	if err != nil || total != 2 || len(rows) != 1 || rows[0]["slug"] != "alpha-draft" {
		t.Fatal(rows, total, err)
	}
	if rows[0]["updatedAt"] == nil {
		t.Fatal("management timestamp missing")
	}
}
