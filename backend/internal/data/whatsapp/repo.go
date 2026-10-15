// Package whatsapp implements transactional routing and timestamp-guarded administration.
package whatsapp

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/url"
	auth "peptide-site/internal/biz/admin"
	ids "peptide-site/internal/biz/catalog"
	biz "peptide-site/internal/biz/whatsapp"
	"peptide-site/internal/data"
	media "peptide-site/internal/data/media"
	"peptide-site/internal/pkg/errdef"
	"sort"
	"strconv"
	"time"
)

type Repo struct{ Data *data.Data }

func NewRepo(d *data.Data) *Repo { return &Repo{d} }

type setting struct {
	ID             int64
	DefaultGroupID *int64
	UpdatedAt      time.Time
}
type groupRow struct {
	ID        int64
	Name      string
	Enabled   bool
	UpdatedAt time.Time
}
type accountRow struct {
	Greeting                     string
	ID, GroupID, QRCodeMediaID   int64
	Name, PhoneNumber, QRCodeURL string
	Enabled                      bool
	AllocationCount              int64
	UpdatedAt                    time.Time
}

func str(id int64) string { return strconv.FormatInt(id, 10) }
func actor(ctx context.Context) int64 {
	if i := auth.Current(ctx); i != nil {
		return i.User.ID
	}
	return -1
}
func audit(tx *gorm.DB, ctx context.Context, action string, id int64) error {
	return tx.Exec("INSERT INTO audit_log(user_id,action,target_type,target_id) VALUES(?,?,'whatsapp_group',?)", actor(ctx), action, str(id)).Error
}
func getSetting(tx *gorm.DB, strength string) (setting, error) {
	var s setting
	q := tx.Table("whatsapp_routing_setting").Where("singleton_key")
	if strength != "" {
		q = q.Clauses(clause.Locking{Strength: strength})
	}
	err := q.Take(&s).Error
	return s, err
}
func getRow(tx *gorm.DB, id int64, lock bool) (groupRow, error) {
	var g groupRow
	q := tx.Table("whatsapp_group").Where("id=? AND deleted_at IS NULL", id)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.Take(&g).Error
	return g, err
}
func accounts(tx *gorm.DB, id int64, usable bool) ([]accountRow, error) {
	rows := []accountRow{}
	q := tx.Table("whatsapp_account a").Select("a.*,COALESCE(m.url,'') AS qr_code_url").Joins("LEFT JOIN media m ON m.id=a.qr_code_media_id AND m.available AND m.deleted_at IS NULL AND m.media_type='image'").Where("a.group_id=? AND a.deleted_at IS NULL", id)
	if usable {
		q = q.Where("a.enabled AND m.id IS NOT NULL AND a.phone_number ~ '^[1-9][0-9]{6,14}$'")
	}
	err := q.Order("a.id").Find(&rows).Error
	return rows, err
}
func detail(tx *gorm.DB, g groupRow, s setting, withAccounts bool) (biz.Group, error) {
	b := biz.Group{ID: str(g.ID), Name: g.Name, Enabled: g.Enabled, UpdatedAt: g.UpdatedAt.UTC(), CountryCodes: []string{}, Accounts: []biz.Account{}}
	if err := tx.Table("whatsapp_group_country").Where("group_id=? AND deleted_at IS NULL", g.ID).Order("country_code").Pluck("country_code", &b.CountryCodes).Error; err != nil {
		return b, err
	}
	rows, err := accounts(tx, g.ID, false)
	if err != nil {
		return b, err
	}
	b.IsDefault = s.DefaultGroupID != nil && *s.DefaultGroupID == g.ID
	b.AccountCount = len(rows)
	b.PendingSetup = len(rows) == 0
	for _, a := range rows {
		if a.Enabled && a.QRCodeURL != "" && biz.Phone.MatchString(a.PhoneNumber) {
			b.AvailableAccountCount++
		}
		if withAccounts {
			b.Accounts = append(b.Accounts, biz.Account{ID: str(a.ID), Name: a.Name, Greeting: a.Greeting, PhoneNumber: a.PhoneNumber, QRCodeMediaID: str(a.QRCodeMediaID), QRCodeURL: a.QRCodeURL, Enabled: a.Enabled, AllocationCount: a.AllocationCount, UpdatedAt: a.UpdatedAt.UTC()})
		}
	}
	return b, nil
}
func (r *Repo) Get(ctx context.Context, id int64) (biz.Group, error) {
	var b biz.Group
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, e := getSetting(tx, "")
		if e != nil {
			return e
		}
		g, e := getRow(tx, id, false)
		if e != nil {
			return e
		}
		b, e = detail(tx, g, s, true)
		return e
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return b, err
}
func (r *Repo) List(ctx context.Context, page, size int, q string) (biz.List, error) {
	out := biz.List{Items: []biz.Group{}, Page: page, PageSize: size}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, e := getSetting(tx, "")
		if e != nil {
			return e
		}
		out.RoutingUpdatedAt = s.UpdatedAt.UTC()
		if s.DefaultGroupID != nil {
			out.DefaultGroupID = str(*s.DefaultGroupID)
			g, e := getRow(tx, *s.DefaultGroupID, false)
			if e == nil {
				out.DefaultGroupName = g.Name
			} else if !errors.Is(e, gorm.ErrRecordNotFound) {
				return e
			}
		}
		query := tx.Table("whatsapp_group").Where("deleted_at IS NULL")
		if q != "" {
			query = query.Where("strpos(lower(name),lower(?))>0 OR EXISTS(SELECT 1 FROM whatsapp_group_country c WHERE c.group_id=whatsapp_group.id AND c.deleted_at IS NULL AND c.country_code=upper(?))", q, q)
		}
		if e := query.Count(&out.Total).Error; e != nil {
			return e
		}
		var rows []groupRow
		if e := query.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; e != nil {
			return e
		}
		for _, g := range rows {
			b, e := detail(tx, g, s, false)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, b)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return out, err
}
func timestampFields(old time.Time) map[string]any {
	return map[string]any{"updated_at": gorm.Expr("GREATEST(clock_timestamp(),?::timestamptz + interval '1 microsecond')", old)}
}
func bump(tx *gorm.DB, ctx context.Context, g groupRow, fields map[string]any) error {
	fields["updated_at"] = timestampFields(g.UpdatedAt)["updated_at"]
	fields["updated_by"] = actor(ctx)
	res := tx.Table("whatsapp_group").Where("id=? AND updated_at=? AND deleted_at IS NULL", g.ID, g.UpdatedAt).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errdef.New(409, "UPDATE_CONFLICT")
	}
	return nil
}
func isDefault(s setting, id int64) bool { return s.DefaultGroupID != nil && *s.DefaultGroupID == id }
func protect(tx *gorm.DB, s setting, g groupRow, previouslyUsable bool) error {
	if !isDefault(s, g.ID) {
		return nil
	}
	if !g.Enabled {
		return errdef.New(409, "DEFAULT_GROUP_PROTECTED")
	}
	pool, e := accounts(tx, g.ID, true)
	if e != nil {
		return e
	}
	if previouslyUsable && len(pool) == 0 {
		return errdef.New(409, "DEFAULT_GROUP_PROTECTED")
	}
	return nil
}
func locked(tx *gorm.DB, id int64, expected string) (setting, groupRow, error) {
	// Bound locks even for a request whose context has no deadline (maintenance/tests).
	if e := tx.Exec("SET LOCAL lock_timeout='3s'").Error; e != nil {
		return setting{}, groupRow{}, e
	}
	if e := tx.Exec("SET LOCAL statement_timeout='10s'").Error; e != nil {
		return setting{}, groupRow{}, e
	}
	s, e := getSetting(tx, "UPDATE")
	if e != nil {
		return s, groupRow{}, e
	}
	g, e := getRow(tx, id, true)
	if e != nil {
		return s, g, e
	}
	return s, g, biz.MatchTimestamp(expected, g.UpdatedAt)
}
func uniqueError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		if pg.ConstraintName == "whatsapp_country_unique" {
			return errdef.New(409, "COUNTRY_ALREADY_ASSIGNED")
		}
		if pg.ConstraintName == "whatsapp_account_phone" {
			return errdef.New(409, "DUPLICATE_WHATSAPP_ACCOUNT")
		}
	}
	return err
}
func (r *Repo) Save(ctx context.Context, id int64, in biz.Input) (biz.Group, error) {
	if e := in.Validate(); e != nil {
		return biz.Group{}, e
	}
	for _, c := range in.CountryCodes {
		if !biz.ValidCountry(c) {
			return biz.Group{}, errdef.Invalid()
		}
	}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SET LOCAL lock_timeout='3s'").Error; e != nil {
			return e
		}
		if e := tx.Exec("SET LOCAL statement_timeout='10s'").Error; e != nil {
			return e
		}
		s, e := getSetting(tx, "UPDATE")
		if e != nil {
			return e
		}
		var g groupRow
		oldPool := []accountRow{}
		oldAccounts := []accountRow{}
		if id != 0 {
			g, e = getRow(tx, id, true)
			if e != nil {
				return e
			}
			if e = biz.MatchTimestamp(in.ExpectedUpdatedAt, g.UpdatedAt); e != nil {
				return e
			}
			oldPool, e = accounts(tx, id, true)
			if e != nil {
				return e
			}
			oldAccounts, e = accounts(tx, id, false)
			if e != nil {
				return e
			}
		} else {
			if in.ExpectedUpdatedAt != "" || len(in.Accounts) == 0 {
				return errdef.Invalid()
			}
			if e = tx.Raw("INSERT INTO whatsapp_group(name,enabled,created_by,updated_by) VALUES(?,?,?,?) RETURNING id", in.Name, in.Enabled, actor(ctx), actor(ctx)).Scan(&id).Error; e != nil {
				return e
			}
			if e = tx.Exec("INSERT INTO whatsapp_cursor(group_id) VALUES(?)", id).Error; e != nil {
				return e
			}
		}
		// Lock every submitted image globally in sorted order before per-account refs.
		imageIDs := []int64{}
		imageSet := map[int64]bool{}
		for _, a := range in.Accounts {
			mid, e := ids.ParseID(a.QRCodeMediaID)
			if e != nil {
				return e
			}
			if !imageSet[mid] {
				imageSet[mid] = true
				imageIDs = append(imageIDs, mid)
			}
		}
		sort.Slice(imageIDs, func(i, j int) bool { return imageIDs[i] < imageIDs[j] })
		for _, mid := range imageIDs {
			var m struct{ ID int64 }
			if e = tx.Table("media").Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id=? AND available AND deleted_at IS NULL AND media_type='image'", mid).Take(&m).Error; e != nil {
				if errors.Is(e, gorm.ErrRecordNotFound) {
					return errdef.Invalid()
				}
				return e
			}
		}
		oldMap := map[int64]accountRow{}
		keep := map[int64]bool{}
		for _, a := range oldAccounts {
			oldMap[a.ID] = a
		}
		parsed := map[int64]biz.AccountInput{}
		for _, a := range in.Accounts {
			if a.ID != "" {
				aid, e := ids.ParseID(a.ID)
				if e != nil {
					return e
				}
				if _, ok := oldMap[aid]; !ok {
					return errdef.Invalid()
				}
				keep[aid] = true
				parsed[aid] = a
			}
		}
		for _, a := range oldAccounts {
			if !keep[a.ID] {
				if e = tx.Exec("UPDATE whatsapp_account SET deleted_at=clock_timestamp(),updated_at=GREATEST(clock_timestamp(),updated_at+interval '1 microsecond'),updated_by=? WHERE id=?", actor(ctx), a.ID).Error; e != nil {
					return e
				}
				if e = media.SyncReferences(tx, "whatsapp_account", str(a.ID), nil); e != nil {
					return e
				}
			}
		}
		// Vacate changed phone values temporarily so swapping two retained numbers
		// cannot hit the unique index. These markers are never visible outside tx.
		for aid, a := range parsed {
			if oldMap[aid].PhoneNumber != a.PhoneNumber {
				if e = tx.Exec("UPDATE whatsapp_account SET phone_number=? WHERE id=?", "pending:"+str(aid), aid).Error; e != nil {
					return e
				}
			}
		}
		for _, a := range in.Accounts {
			mid, _ := ids.ParseID(a.QRCodeMediaID)
			var aid int64
			if a.ID == "" {
				if e = tx.Raw("INSERT INTO whatsapp_account(group_id,name,phone_number,qr_code_media_id,enabled,created_by,updated_by,greeting) VALUES(?,?,?,?,?,?,?,?) RETURNING id", id, a.Name, a.PhoneNumber, mid, a.Enabled, actor(ctx), actor(ctx), a.Greeting).Scan(&aid).Error; e != nil {
					return e
				}
			} else {
				aid, _ = ids.ParseID(a.ID)
				if e = tx.Exec("UPDATE whatsapp_account SET name=?,phone_number=?,qr_code_media_id=?,enabled=?,updated_by=?,greeting=?,updated_at=GREATEST(clock_timestamp(),updated_at+interval '1 microsecond') WHERE id=?", a.Name, a.PhoneNumber, mid, a.Enabled, actor(ctx), a.Greeting, aid).Error; e != nil {
					return e
				}
			}
			if e = media.SyncReferences(tx, "whatsapp_account", str(aid), map[string][]int64{"qr_code": {mid}}); e != nil {
				return e
			}
		}
		if e = tx.Exec("UPDATE whatsapp_group_country SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by=? WHERE group_id=? AND deleted_at IS NULL AND country_code NOT IN ?", actor(ctx), id, in.CountryCodes).Error; e != nil {
			return e
		}
		for _, c := range in.CountryCodes {
			var count int64
			if e = tx.Table("whatsapp_group_country").Where("group_id=? AND country_code=? AND deleted_at IS NULL", id, c).Count(&count).Error; e != nil {
				return e
			}
			if count == 0 {
				if e = tx.Exec("INSERT INTO whatsapp_group_country(group_id,country_code,created_by,updated_by) VALUES(?,?,?,?)", id, c, actor(ctx), actor(ctx)).Error; e != nil {
					return e
				}
			}
		}
		if g.ID != 0 {
			if e = bump(tx, ctx, g, map[string]any{"name": in.Name, "enabled": in.Enabled}); e != nil {
				return e
			}
		}
		newG := groupRow{ID: id, Enabled: in.Enabled}
		if e = protect(tx, s, newG, len(oldPool) > 0); e != nil {
			return e
		}
		if !isDefault(s, id) && len(in.Accounts) == 0 {
			return errdef.Invalid()
		}
		action := "whatsapp.update"
		if g.ID == 0 {
			action = "whatsapp.create"
		}
		return audit(tx, ctx, action, id)
	})
	if err != nil {
		return biz.Group{}, uniqueError(err)
	}
	return r.Get(ctx, id)
}
func (r *Repo) Enabled(ctx context.Context, id, accountID int64, expected string, enabled bool) (biz.Group, error) {
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, g, e := locked(tx, id, expected)
		if e != nil {
			return e
		}
		pool, e := accounts(tx, id, true)
		if e != nil {
			return e
		}
		fields := map[string]any{}
		if accountID == 0 {
			g.Enabled = enabled
			fields["enabled"] = enabled
		} else {
			res := tx.Exec("UPDATE whatsapp_account SET enabled=?,updated_by=?,updated_at=GREATEST(clock_timestamp(),updated_at+interval '1 microsecond') WHERE id=? AND group_id=? AND deleted_at IS NULL", enabled, actor(ctx), accountID, id)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != 1 {
				return gorm.ErrRecordNotFound
			}
		}
		if e = protect(tx, s, g, len(pool) > 0); e != nil {
			return e
		}
		if e = bump(tx, ctx, g, fields); e != nil {
			return e
		}
		return audit(tx, ctx, "whatsapp.enabled", id)
	})
	if err != nil {
		return biz.Group{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repo) Delete(ctx context.Context, id int64, expected string) error {
	return r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, g, e := locked(tx, id, expected)
		if e != nil {
			return e
		}
		if isDefault(s, id) {
			return errdef.New(409, "DEFAULT_GROUP_PROTECTED")
		}
		rows, e := accounts(tx, id, false)
		if e != nil {
			return e
		}
		for _, a := range rows {
			if e = media.SyncReferences(tx, "whatsapp_account", str(a.ID), nil); e != nil {
				return e
			}
		}
		if e = tx.Exec("UPDATE whatsapp_account SET deleted_at=clock_timestamp(),updated_at=GREATEST(clock_timestamp(),updated_at+interval '1 microsecond'),updated_by=? WHERE group_id=? AND deleted_at IS NULL", actor(ctx), id).Error; e != nil {
			return e
		}
		if e = tx.Exec("UPDATE whatsapp_group_country SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by=? WHERE group_id=? AND deleted_at IS NULL", actor(ctx), id).Error; e != nil {
			return e
		}
		if e = bump(tx, ctx, g, map[string]any{"deleted_at": gorm.Expr("clock_timestamp()")}); e != nil {
			return e
		}
		return audit(tx, ctx, "whatsapp.delete", id)
	})
}
func (r *Repo) Default(ctx context.Context, in biz.DefaultInput) (biz.DefaultResult, error) {
	out := biz.DefaultResult{}
	id, e := ids.ParseID(in.GroupID)
	if e != nil {
		return out, e
	}
	e = r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, g, e := locked(tx, id, in.ExpectedGroupUpdatedAt)
		if e != nil {
			return e
		}
		if e = biz.MatchTimestamp(in.ExpectedUpdatedAt, s.UpdatedAt); e != nil {
			return e
		}
		pool, e := accounts(tx, id, true)
		if e != nil {
			return e
		}
		if !g.Enabled || len(pool) == 0 {
			return errdef.New(409, "DEFAULT_GROUP_UNAVAILABLE")
		}
		if !isDefault(s, id) {
			res := tx.Table("whatsapp_routing_setting").Where("id=? AND updated_at=?", s.ID, s.UpdatedAt).Updates(map[string]any{"default_group_id": id, "updated_at": timestampFields(s.UpdatedAt)["updated_at"], "updated_by": actor(ctx)})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != 1 {
				return errdef.New(409, "UPDATE_CONFLICT")
			}
			if e = audit(tx, ctx, "whatsapp.default", id); e != nil {
				return e
			}
			s, e = getSetting(tx, "")
			if e != nil {
				return e
			}
		}
		out = biz.DefaultResult{DefaultGroupID: str(id), RoutingUpdatedAt: s.UpdatedAt.UTC()}
		return nil
	})
	return out, e
}
func (r *Repo) Status(ctx context.Context) (bool, error) {
	var count int64
	err := r.Data.DB.WithContext(ctx).Raw(`SELECT count(*) FROM whatsapp_routing_setting s JOIN whatsapp_group g ON g.id=s.default_group_id WHERE g.enabled AND g.deleted_at IS NULL AND EXISTS (SELECT 1 FROM whatsapp_account a JOIN media m ON m.id=a.qr_code_media_id WHERE a.group_id=g.id AND a.enabled AND a.deleted_at IS NULL AND m.available AND m.deleted_at IS NULL AND m.media_type='image' AND a.phone_number ~ '^[1-9][0-9]{6,14}$')`).Scan(&count).Error
	return count > 0, err
}
func (r *Repo) Contact(ctx context.Context, country string) (biz.Contact, error) {
	out := biz.Contact{}
	err := r.Data.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SET LOCAL lock_timeout='3s'").Error; e != nil {
			return e
		}
		if e := tx.Exec("SET LOCAL statement_timeout='10s'").Error; e != nil {
			return e
		}
		s, e := getSetting(tx, "SHARE")
		if e != nil {
			return e
		}
		var g groupRow
		pool := []accountRow{}
		if country != "" {
			e = tx.Table("whatsapp_group g").Select("g.*").Joins("JOIN whatsapp_group_country c ON c.group_id=g.id AND c.deleted_at IS NULL").Where("c.country_code=? AND g.enabled AND g.deleted_at IS NULL", country).Take(&g).Error
			if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
				return e
			}
			if e == nil {
				pool, e = accounts(tx, g.ID, true)
				if e != nil {
					return e
				}
			}
		}
		if len(pool) == 0 {
			if s.DefaultGroupID == nil {
				return errdef.New(503, "WHATSAPP_UNAVAILABLE")
			}
			g, e = getRow(tx, *s.DefaultGroupID, false)
			if e != nil {
				if errors.Is(e, gorm.ErrRecordNotFound) {
					return errdef.New(503, "WHATSAPP_UNAVAILABLE")
				}
				return e
			}
			if !g.Enabled {
				return errdef.New(503, "WHATSAPP_UNAVAILABLE")
			}
			pool, e = accounts(tx, g.ID, true)
			if e != nil {
				return e
			}
			if len(pool) == 0 {
				return errdef.New(503, "WHATSAPP_UNAVAILABLE")
			}
		}
		var cursor struct{ LastAccountID *int64 }
		if e = tx.Table("whatsapp_cursor").Clauses(clause.Locking{Strength: "UPDATE"}).Where("group_id=?", g.ID).Take(&cursor).Error; e != nil {
			return e
		}
		// Re-read under the cursor lock so each concurrent request uses the latest
		// runtime row values. Configuration is stable under the shared setting lock.
		pool, e = accounts(tx, g.ID, true)
		if e != nil {
			return e
		}
		if len(pool) == 0 {
			return errdef.New(503, "WHATSAPP_UNAVAILABLE")
		}
		selected := pool[0]
		if cursor.LastAccountID != nil {
			for _, a := range pool {
				if a.ID > *cursor.LastAccountID {
					selected = a
					break
				}
			}
		}
		if e = tx.Exec("UPDATE whatsapp_cursor SET last_account_id=?,allocation_count=allocation_count+1,updated_at=clock_timestamp() WHERE group_id=?", selected.ID, g.ID).Error; e != nil {
			return e
		}
		if e = tx.Exec("UPDATE whatsapp_account SET allocation_count=allocation_count+1,last_allocated_at=clock_timestamp() WHERE id=?", selected.ID).Error; e != nil {
			return e
		}
		out = biz.Contact{AccountName: selected.Name, PhoneNumber: selected.PhoneNumber, QRCodeURL: selected.QRCodeURL, URL: "https://wa.me/" + selected.PhoneNumber}
		if selected.Greeting != "" {
			out.URL += "?" + url.Values{"text": {selected.Greeting}}.Encode()
		}
		return nil
	})
	return out, err
}
