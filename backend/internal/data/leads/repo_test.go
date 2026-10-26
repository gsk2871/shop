package leads_test

import (
	"context"
	auth "peptide-site/internal/biz/admin"
	biz "peptide-site/internal/biz/leads"
	repo "peptide-site/internal/data/leads"
	"peptide-site/internal/testutil"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestLeadLifecycleAndConfigSeed(t *testing.T) {
	d, _ := testutil.Postgres(t)
	r := repo.NewRepo(d)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}})
	c, err := r.Config(ctx)
	if err != nil || !c.Enabled || c.Title != "Get 10% OFF your order" {
		t.Fatal(c, err)
	}
	c.Title = "A configured offer"
	c.ExpectedVersion = c.Version
	saved, err := r.SaveConfig(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.SaveConfig(ctx, c); err == nil {
		t.Fatal("stale config accepted")
	}
	if err := repo.Seed(d.DB); err != nil {
		t.Fatal(err)
	}
	c, err = r.Config(ctx)
	if err != nil || c.Title != saved.Title || c.Version != saved.Version {
		t.Fatal("seed overwrote config", c, err)
	}
	if err := r.Submit(ctx, "Lead@Example.com"); err != nil {
		t.Fatal(err)
	}
	f := biz.Filter{Page: 1, Size: 20}
	rows, total, err := r.List(ctx, f)
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatal(rows, total, err)
	}
	first := rows[0]
	id, _ := strconv.ParseInt(first.ID, 10, 64)
	if err := r.Update(ctx, id, biz.Update{Status: "completed", Note: "Contacted", ExpectedVersion: first.Version}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.Submit(ctx, "lead@example.com"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	rows, total, err = r.List(ctx, f)
	if err != nil || total != 1 || rows[0].SubmissionCount != 5 || rows[0].Status != "pending" || rows[0].Note != "Contacted" || !rows[0].FirstSubmittedAt.Equal(first.FirstSubmittedAt) {
		t.Fatal(rows, total, err)
	}
	if err := r.Update(ctx, id, biz.Update{Status: "invalid", ExpectedVersion: first.Version}); err == nil {
		t.Fatal("stale lead update accepted")
	}
	if err := r.Submit(ctx, "+1 (202) 555-0123"); err != nil {
		t.Fatal(err)
	}
	if err := r.Submit(ctx, "+12025550123"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"email", "whatsapp", ""} {
		f.ContactType = kind
		rows, total, err = r.List(ctx, f)
		want := int64(1)
		if kind == "" {
			want = 2
		}
		if err != nil || total != want || len(rows) != int(want) {
			t.Fatal(rows, total, err)
		}
		for _, row := range rows {
			if kind != "" && row.ContactType != kind {
				t.Fatal("wrong contact type", row.ContactType)
			}
		}
	}
	f.Contact = "202555"
	rows, total, err = r.List(ctx, f)
	if err != nil || total != 1 || rows[0].ContactType != "whatsapp" || rows[0].SubmissionCount != 2 {
		t.Fatal(rows, total, err)
	}
	f.Contact = "' OR 1=1 --"
	_, total, err = r.List(ctx, f)
	if err != nil || total != 0 {
		t.Fatal("filter not literal", total, err)
	}
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	f.Contact = ""
	f.Start = &past
	f.End = &future
	_, total, err = r.List(ctx, f)
	if err != nil || total != 2 {
		t.Fatal("range filter", total, err)
	}
	f.Start = &future
	_, total, err = r.List(ctx, f)
	if err != nil || total != 0 {
		t.Fatal("start filter", total, err)
	}
	f.Start = nil
	f.End = &past
	_, total, err = r.List(ctx, f)
	if err != nil || total != 0 {
		t.Fatal("end filter", total, err)
	}
	c.Enabled = false
	c.ExpectedVersion = c.Version
	if _, err := r.SaveConfig(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := r.Submit(ctx, "new@example.com"); err == nil {
		t.Fatal("disabled popup accepted lead")
	}
}
