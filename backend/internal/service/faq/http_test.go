package faq

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	biz "peptide-site/internal/biz/faq"
	repo "peptide-site/internal/data/faq"
	"peptide-site/internal/testutil"
)

func TestFAQVisibilityOrderingVersionAndAudit(t *testing.T) {
	d, _ := testutil.Postgres(t)
	r := repo.NewRepo(d)
	ctx := context.Background()
	create := func(question string, sort int, enabled bool) biz.FAQ {
		t.Helper()
		row, err := r.SaveFAQ(ctx, 0, biz.FAQ{Question: question, Answer: "Line one\n<script>plain text</script>", Sort: sort, Enabled: enabled})
		if err != nil {
			t.Fatal(err)
		}
		return row
	}
	hidden := create("Hidden", -10, false)
	later := create("Later", 10, true)
	first := create("First", 0, true)
	tied := create("Tied", 0, true)
	h := NewHandler(r)
	public := func(want ...biz.FAQ) {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/faqs?enabled=false&pageSize=1", nil))
		var result struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 {
			t.Fatal(w.Code, w.Body.String(), err)
		}
		if len(result.Items) != len(want) {
			t.Fatalf("got %v want %v", result, want)
		}
		for i, row := range result.Items {
			if len(row) != 3 || row["id"] != want[i].ID || row["answer"] != want[i].Answer {
				t.Fatal("visibility, order or public contract failed", result)
			}
		}
	}
	public(first, tied, later)
	all, total, err := r.List(ctx, 1, 2, "", false)
	if err != nil || total != 4 || len(all) != 2 || all[0].ID != hidden.ID {
		t.Fatal(all, total, err)
	}
	id, _ := strconv.ParseInt(first.ID, 10, 64)
	first.ExpectedVersion = first.Version
	first.Enabled = false
	updated, err := r.SaveFAQ(ctx, id, first)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Answer != first.Answer || updated.Enabled {
		t.Fatal("round trip failed", updated)
	}
	if _, err = r.SaveFAQ(ctx, id, first); err == nil {
		t.Fatal("stale update accepted")
	}
	if err = r.DeleteFAQ(ctx, id, first.Version); err == nil {
		t.Fatal("stale delete accepted")
	}
	public(tied, later)
	tiedID, _ := strconv.ParseInt(tied.ID, 10, 64)
	if err = r.DeleteFAQ(ctx, tiedID, tied.Version); err != nil {
		t.Fatal(err)
	}
	public(later)
	if _, err = r.FAQ(ctx, tiedID); err == nil {
		t.Fatal("deleted FAQ returned")
	}
	var audits int64
	if err = d.DB.Table("audit_log").Where("target_type='faq'").Count(&audits).Error; err != nil || audits != 6 {
		t.Fatal("audit mismatch", audits, err)
	}
	var deleted int64
	d.DB.Table("faq").Where("id=? AND deleted_at IS NOT NULL", tiedID).Count(&deleted)
	if deleted != 1 {
		t.Fatal("not soft deleted")
	}
}

func TestFAQRejectsInvalidInput(t *testing.T) {
	h := NewHandler(nil)
	for _, body := range []string{`{"question":"Q","answer":"A","extra":1}`, `{} {}`, `{"sort":1.5}`, `{"enabled":"true"}`} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "/admin/api/v1/faqs", bytes.NewBufferString(body)))
		if w.Code != 400 {
			t.Fatal(body, w.Code, w.Body.String())
		}
	}
}
