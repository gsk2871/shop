package branding

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	biz "peptide-site/internal/biz/branding"
	mediaBiz "peptide-site/internal/biz/media"
	"peptide-site/internal/testutil"
)

func socialFixture(t *testing.T) (*Repo, mediaBiz.Media) {
	t.Helper()
	d, c := testutil.Postgres(t)
	_, _, root, commit, err := d.RuntimeMetadata(context.Background(), c.Data.Storage.UploadDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := commit(); err != nil {
		t.Fatal(err)
	}
	asset := mediaBiz.Media{Status: "ready", StorageProvider: "local", Name: "Icon", URL: "/uploads/" + root + "/icon.png", MediaType: "image", MIME: "image/png", Size: 1, Available: true, StorageRootID: root, StorageKey: "icon.png"}
	if err := d.DB.Table("media").Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	return NewRepo(d), asset
}

func TestSocialAccountPersistenceVersionsAndReferences(t *testing.T) {
	r, asset := socialFixture(t)
	ctx := context.Background()
	account := biz.SocialAccount{Name: "Channel", Platform: "youtube", Link: "https://www.youtube.com/@channel", IconMediaID: strconv.FormatInt(asset.ID, 10), Icon: "https://untrusted.example/icon", Enabled: false}
	first, err := r.SaveSocialAccount(ctx, 0, account)
	if err != nil || first.Icon != asset.URL || first.Enabled || first.ID == "" {
		t.Fatal(first, err)
	}
	second, err := r.SaveSocialAccount(ctx, 0, account)
	if err != nil || second.ID == first.ID {
		t.Fatal("same-platform accounts must have independent IDs", second, err)
	}
	items, err := r.SocialAccounts(ctx, true)
	if err != nil || len(items) != 0 {
		t.Fatal("disabled account leaked", items, err)
	}
	firstID, _ := strconv.ParseInt(first.ID, 10, 64)
	updated, err := r.ChangeSocialAccount(ctx, firstID, first.Version, true, false)
	if err != nil || !updated.Enabled || updated.Version != first.Version+1 {
		t.Fatal(updated, err)
	}
	if _, err := r.ChangeSocialAccount(ctx, firstID, first.Version, false, false); err == nil {
		t.Fatal("stale toggle accepted")
	}
	first.ExpectedVersion = first.Version
	if _, err := r.SaveSocialAccount(ctx, firstID, first); err == nil {
		t.Fatal("stale edit accepted")
	}
	// Editing a different account does not conflict with the first account's version.
	secondID, _ := strconv.ParseInt(second.ID, 10, 64)
	second.ExpectedVersion = second.Version
	second.Name = "Second channel"
	second, err = r.SaveSocialAccount(ctx, secondID, second)
	if err != nil {
		t.Fatal(err)
	}
	s := biz.Default()
	s.Brand = "Brand"
	s.ExpectedVersion = s.Version
	if _, err := r.SaveSettings(ctx, s); err != nil {
		t.Fatal(err)
	}
	items, err = r.SocialAccounts(ctx, true)
	if err != nil || len(items) != 1 || items[0].ID != first.ID {
		t.Fatal("brand save affected accounts", items, err)
	}
	if err := r.Media.Delete(ctx, asset.ID); err == nil {
		t.Fatal("referenced icon deleted")
	}
	if _, err := r.ChangeSocialAccount(ctx, firstID, first.Version, false, true); err == nil {
		t.Fatal("stale delete accepted")
	}
	if _, err := r.ChangeSocialAccount(ctx, firstID, updated.Version, false, true); err != nil {
		t.Fatal(err)
	}
	if err := r.Media.Delete(ctx, asset.ID); err == nil {
		t.Fatal("disabled account reference lost")
	}
	if _, err := r.ChangeSocialAccount(ctx, secondID, second.Version, false, true); err != nil {
		t.Fatal(err)
	}
	if err := r.Media.Delete(ctx, asset.ID); err != nil {
		t.Fatal("deleted accounts retained media references", err)
	}
	var deleted int64
	if err := r.Data.DB.Table("social_account").Where("deleted_at IS NOT NULL").Count(&deleted).Error; err != nil || deleted != 2 {
		t.Fatal("not soft-deleted", deleted, err)
	}
	account.IconMediaID = "999999999"
	if _, err := r.SaveSocialAccount(ctx, 0, account); err == nil {
		t.Fatal("missing icon accepted")
	}
	var live int64
	if err := r.Data.DB.Table("social_account").Where("deleted_at IS NULL").Count(&live).Error; err != nil || live != 0 {
		t.Fatal("failed save was not rolled back", live, err)
	}
}

func TestSocialAccountMigrationRoundTrip(t *testing.T) {
	r, asset := socialFixture(t)
	ctx := context.Background()
	down, err := os.ReadFile("../migration/000005_social_account.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../migration/000005_social_account.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Data.DB.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	legacy := []biz.SocialAccount{
		{Name: "first", Platform: "x", Link: "https://x.com/first", IconMediaID: strconv.FormatInt(asset.ID, 10), Enabled: true},
		{Name: "second", Platform: "x", Link: "https://x.com/second", IconMediaID: strconv.FormatInt(asset.ID, 10), Enabled: false},
	}
	body, _ := json.Marshal(map[string]any{"brand": "Brand", "socialAccounts": legacy})
	if err := r.Data.DB.Exec("UPDATE site_config SET payload=?::jsonb WHERE id=1", string(body)).Error; err != nil {
		t.Fatal(err)
	}
	if err := r.Data.DB.Exec("INSERT INTO media_reference(media_id,owner_type,owner_id,field) VALUES(?,'site_config','1','social_icon')", asset.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := r.Data.DB.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	items, err := r.SocialAccounts(ctx, false)
	if err != nil || len(items) != 2 || items[0].Name != "first" || items[1].Name != "second" || items[1].Enabled {
		t.Fatal(items, err)
	}
	var oldReferences int64
	if err := r.Data.DB.Table("media_reference").Where("owner_type='site_config' AND field='social_icon'").Count(&oldReferences).Error; err != nil || oldReferences != 0 {
		t.Fatal(oldReferences, err)
	}
	if err := r.Media.Delete(ctx, asset.ID); err == nil {
		t.Fatal("migrated media reference missing")
	}
	if err := r.Data.DB.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	var payload string
	if err := r.Data.DB.Raw("SELECT payload::text FROM site_config WHERE id=1").Scan(&payload).Error; err != nil {
		t.Fatal(err)
	}
	var restored struct {
		Brand    string              `json:"brand"`
		Accounts []biz.SocialAccount `json:"socialAccounts"`
	}
	if err := json.Unmarshal([]byte(payload), &restored); err != nil || restored.Brand != "Brand" || len(restored.Accounts) != 2 || restored.Accounts[0].Name != "first" || restored.Accounts[1].Enabled || restored.Accounts[0].Icon != asset.URL {
		t.Fatal(restored, err)
	}
	if err := r.Media.Delete(ctx, asset.ID); err == nil {
		t.Fatal("rollback media reference missing")
	}
}
