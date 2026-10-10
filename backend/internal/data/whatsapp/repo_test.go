package whatsapp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	auth "peptide-site/internal/biz/admin"
	ids "peptide-site/internal/biz/catalog"
	mediaBiz "peptide-site/internal/biz/media"
	biz "peptide-site/internal/biz/whatsapp"
	media "peptide-site/internal/data/media"
	routing "peptide-site/internal/data/whatsapp"
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/testutil"
	"strconv"
	"sync"
	"testing"
	"time"
)

func code(err error) string {
	var e *errdef.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
func stamp(g biz.Group) string { return g.UpdatedAt.Format(time.RFC3339Nano) }
func input(g biz.Group) biz.Input {
	in := biz.Input{Name: g.Name, CountryCodes: g.CountryCodes, Enabled: g.Enabled, ExpectedUpdatedAt: stamp(g), Accounts: []biz.AccountInput{}}
	for _, a := range g.Accounts {
		in.Accounts = append(in.Accounts, biz.AccountInput{ID: a.ID, Name: a.Name, PhoneNumber: a.PhoneNumber, QRCodeMediaID: a.QRCodeMediaID, Enabled: a.Enabled})
	}
	return in
}
func TestRoutingConcurrencyAndConfiguration(t *testing.T) {
	d, c := testutil.Postgres(t)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{User: auth.User{ID: 1}})
	r := routing.NewRepo(d)
	mr := media.NewRepo(d)
	_, _, root, commit, e := d.RuntimeMetadata(ctx, c.Data.Storage.UploadDir)
	if e != nil {
		t.Fatal(e)
	}
	if e = commit(); e != nil {
		t.Fatal(e)
	}
	asset := mediaBiz.Media{Name: "QR", URL: "/uploads/" + root + "/0123456789abcdef0123456789abcdef.png", MIME: "image/png", MediaType: "image", Available: true, StorageRootID: root, StorageKey: "0123456789abcdef0123456789abcdef.png", Size: 1}
	if e = mr.Save(ctx, &asset); e != nil {
		t.Fatal(e)
	}
	makeAccount := func(n int) biz.AccountInput {
		return biz.AccountInput{Name: fmt.Sprintf("Account %d", n), PhoneNumber: fmt.Sprintf("12025550%03d", n), QRCodeMediaID: str(asset.ID), Enabled: true}
	}
	list, e := r.List(ctx, 1, 20, "")
	if e != nil || len(list.Items) != 1 || list.Items[0].CountryCodes[0] != "US" || list.DefaultGroupID != list.Items[0].ID {
		t.Fatal("US seed", list, e)
	}
	if ok, e := r.Status(ctx); e != nil || ok {
		t.Fatal("empty seed advertised", e)
	}
	if _, e = r.Contact(ctx, "US"); code(e) != "WHATSAPP_UNAVAILABLE" {
		t.Fatal("empty seed contact", e)
	}
	g, e := r.Get(ctx, mustID(list.DefaultGroupID))
	if e != nil {
		t.Fatal(e)
	}
	in := input(g)
	for n := 1; n <= 4; n++ {
		in.Accounts = append(in.Accounts, makeAccount(n))
	}
	g, e = r.Save(ctx, mustID(g.ID), in)
	if e != nil {
		t.Fatal(e)
	}
	_, offset := g.UpdatedAt.Zone()
	if offset != 0 {
		t.Fatal("transport timestamp must be UTC")
	}
	id := mustID(g.ID)
	if e = mr.Delete(ctx, asset.ID); code(e) != "MEDIA_IN_USE" {
		t.Fatal("QR ref unprotected", e)
	}
	for n := 1; n <= 4; n++ {
		v, e := r.Contact(ctx, "US")
		if e != nil || v.PhoneNumber != makeAccount(n).PhoneNumber {
			t.Fatal("initial round", n, v, e)
		}
	}
	before := stamp(g)
	g, e = r.Get(ctx, id)
	if e != nil || before != stamp(g) {
		t.Fatal("runtime bumped edit timestamp", e)
	}
	in = input(g)
	in.Accounts = append(in.Accounts, makeAccount(5), makeAccount(6))
	g, e = r.Save(ctx, id, in)
	if e != nil {
		t.Fatal(e)
	}
	for _, n := range []int{5, 6, 1, 2, 3, 4} {
		v, e := r.Contact(ctx, "US")
		if e != nil || v.PhoneNumber != makeAccount(n).PhoneNumber {
			t.Fatal("append reset", n, v, e)
		}
	}
	counts := map[string]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for n := 0; n < 60; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, e := r.Contact(ctx, "ZZ")
			if e != nil {
				t.Error(e)
				return
			}
			mu.Lock()
			counts[v.PhoneNumber]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	for n := 1; n <= 6; n++ {
		if counts[makeAccount(n).PhoneNumber] != 10 {
			t.Fatal("not balanced", counts)
		}
	}
	// Two concurrent mutations using the same timestamp cannot both succeed.
	old := g
	results := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, e := r.Enabled(ctx, id, mustID(old.Accounts[n].ID), stamp(old), false)
			results <- e
		}(n)
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for e := range results {
		if e == nil {
			wins++
		} else if code(e) == "UPDATE_CONFLICT" {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatal("overwritten", wins, conflicts)
	}
	if _, e = r.Save(ctx, id, input(old)); code(e) != "UPDATE_CONFLICT" {
		t.Fatal("stale aggregate accepted", e)
	}
	g, e = r.Get(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	millis := g.UpdatedAt.Truncate(time.Millisecond).Format(time.RFC3339Nano)
	if millis != stamp(g) {
		if _, e = r.Enabled(ctx, id, 0, millis, true); code(e) != "UPDATE_CONFLICT" {
			t.Fatal("lost timestamp precision", e)
		}
	}
	// Disable until only one remains, then preserve the default's last account.
	for _, a := range g.Accounts {
		if a.Enabled && g.AvailableAccountCount > 1 {
			g, e = r.Enabled(ctx, id, mustID(a.ID), stamp(g), false)
			if e != nil {
				t.Fatal(e)
			}
		}
	}
	for _, a := range g.Accounts {
		if a.Enabled {
			if _, e = r.Enabled(ctx, id, mustID(a.ID), stamp(g), false); code(e) != "DEFAULT_GROUP_PROTECTED" {
				t.Fatal("disabled last", e)
			}
		}
	}
	if _, e = r.Enabled(ctx, id, 0, stamp(g), false); code(e) != "DEFAULT_GROUP_PROTECTED" {
		t.Fatal("disabled default", e)
	}
	if e = r.Delete(ctx, id, stamp(g)); code(e) != "DEFAULT_GROUP_PROTECTED" {
		t.Fatal("deleted default", e)
	}
	// Non-default disabled pools fall back to the shared default cursor.
	uk, e := r.Save(ctx, 0, biz.Input{Name: "UK", CountryCodes: []string{"GB"}, Enabled: true, Accounts: []biz.AccountInput{makeAccount(20)}})
	if e != nil {
		t.Fatal(e)
	}
	v, e := r.Contact(ctx, "GB")
	if e != nil || v.PhoneNumber != makeAccount(20).PhoneNumber {
		t.Fatal("country match", v, e)
	}
	uk, e = r.Enabled(ctx, mustID(uk.ID), mustID(uk.Accounts[0].ID), stamp(uk), false)
	if e != nil {
		t.Fatal(e)
	}
	v, e = r.Contact(ctx, "GB")
	if e != nil || v.PhoneNumber == makeAccount(20).PhoneNumber {
		t.Fatal("fallback", v, e)
	}
	list, e = r.List(ctx, 1, 20, "")
	if e != nil {
		t.Fatal(e)
	}
	din := biz.DefaultInput{GroupID: uk.ID, ExpectedUpdatedAt: list.RoutingUpdatedAt.Format(time.RFC3339Nano), ExpectedGroupUpdatedAt: stamp(uk)}
	if _, e = r.Default(ctx, din); code(e) != "DEFAULT_GROUP_UNAVAILABLE" {
		t.Fatal("empty default accepted", e)
	}
	uk, e = r.Enabled(ctx, mustID(uk.ID), mustID(uk.Accounts[0].ID), stamp(uk), true)
	if e != nil {
		t.Fatal(e)
	}
	ca, e := r.Save(ctx, 0, biz.Input{Name: "Canada", CountryCodes: []string{"CA"}, Enabled: true, Accounts: []biz.AccountInput{makeAccount(30)}})
	if e != nil {
		t.Fatal(e)
	}
	results = make(chan error, 2)
	for _, target := range []biz.Group{uk, ca} {
		wg.Add(1)
		go func(target biz.Group) {
			defer wg.Done()
			_, e := r.Default(ctx, biz.DefaultInput{GroupID: target.ID, ExpectedUpdatedAt: list.RoutingUpdatedAt.Format(time.RFC3339Nano), ExpectedGroupUpdatedAt: stamp(target)})
			results <- e
		}(target)
	}
	wg.Wait()
	close(results)
	wins, conflicts = 0, 0
	for e := range results {
		if e == nil {
			wins++
		} else if code(e) == "UPDATE_CONFLICT" {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatal("default race", wins, conflicts)
	}
	list, e = r.List(ctx, 1, 20, "")
	if e != nil {
		t.Fatal(e)
	}
	defaults := 0
	for _, row := range list.Items {
		if row.IsDefault {
			defaults++
		}
	}
	if defaults != 1 {
		t.Fatal("multiple defaults")
	}
	beforeList, _ := json.Marshal(list)
	if e = d.DB.WithContext(ctx).Transaction(routing.Seed); e != nil {
		t.Fatal(e)
	}
	list, e = r.List(ctx, 1, 20, "")
	if e != nil {
		t.Fatal(e)
	}
	afterList, _ := json.Marshal(list)
	if string(beforeList) != string(afterList) {
		t.Fatal("seed reset configuration")
	}
	// A conflicting country and invalid QR must not leave partial new rows.
	if _, e = r.Save(ctx, 0, biz.Input{Name: "Duplicate", CountryCodes: []string{"US"}, Enabled: true, Accounts: []biz.AccountInput{makeAccount(40)}}); code(e) != "COUNTRY_ALREADY_ASSIGNED" {
		t.Fatal("country duplicate", e)
	}
	bad := makeAccount(40)
	bad.QRCodeMediaID = "99999999"
	if _, e = r.Save(ctx, 0, biz.Input{Name: "Missing QR", CountryCodes: []string{"FR"}, Enabled: true, Accounts: []biz.AccountInput{bad}}); e == nil {
		t.Fatal("missing QR accepted")
	}
	latest, e := r.List(ctx, 1, 20, "")
	if e != nil || latest.Total != 3 {
		t.Fatal("partial creation", latest, e)
	}
	// Force a failure at cursor update; no account or cursor counter may advance.
	var beforeCount int64
	if e = d.DB.Table("whatsapp_cursor").Select("allocation_count").Where("group_id=?", mustID(list.DefaultGroupID)).Scan(&beforeCount).Error; e != nil {
		t.Fatal(e)
	}
	if e = d.DB.Exec("CREATE FUNCTION reject_account_update() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced failure'; END $$; CREATE TRIGGER test_reject BEFORE UPDATE ON whatsapp_account FOR EACH ROW EXECUTE FUNCTION reject_account_update()").Error; e != nil {
		t.Fatal(e)
	}
	if _, e = r.Contact(ctx, ""); e == nil {
		t.Fatal("forced failure absent")
	}
	var afterCount int64
	d.DB.Table("whatsapp_cursor").Select("allocation_count").Where("group_id=?", mustID(list.DefaultGroupID)).Scan(&afterCount)
	if beforeCount != afterCount {
		t.Fatal("failed tx advanced cursor")
	}
	if e = d.DB.Exec("DROP TRIGGER test_reject ON whatsapp_account; DROP FUNCTION reject_account_update()").Error; e != nil {
		t.Fatal(e)
	}
	// Releasing a non-default group releases every QR reference it owned.
	for _, row := range list.Items {
		if !row.IsDefault {
			if e = r.Delete(ctx, mustID(row.ID), stamp(row)); e != nil {
				t.Fatal(e)
			}
		}
	}
	var refs int64
	if e = d.DB.Table("media_reference").Where("owner_type='whatsapp_account'").Count(&refs).Error; e != nil {
		t.Fatal(e)
	}
	target, e := r.Get(ctx, mustID(list.DefaultGroupID))
	if e != nil || refs != int64(len(target.Accounts)) {
		t.Fatal("orphan references", refs, e)
	}
}
func mustID(s string) int64 {
	n, e := ids.ParseID(s)
	if e != nil {
		panic(e)
	}
	return n
}

func str(id int64) string { return strconv.FormatInt(id, 10) }
