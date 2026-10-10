package data_test

import (
	"bytes"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"peptide-site/internal/data"
	"peptide-site/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestDatabaseTimeoutsAndPool(t *testing.T) {
	_, c := testutil.Postgres(t)
	c.Data.Database.DSN += " statement_timeout=6000 lock_timeout=5000 connect_timeout=20"
	c.Data.Database.StatementTimeout = 80 * time.Millisecond
	c.Data.Database.LockTimeout = 20 * time.Millisecond
	c.Data.Database.ConnMaxIdleTime = 50 * time.Millisecond
	d, closeDB, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	defer closeDB()
	var value string
	if err = d.DB.Raw("SHOW statement_timeout").Scan(&value).Error; err != nil || value != "80ms" {
		t.Fatal("statement setting", value, err)
	}
	if err = d.DB.Raw("SHOW lock_timeout").Scan(&value).Error; err != nil || value != "20ms" {
		t.Fatal("lock setting", value, err)
	}
	err = d.DB.Exec("SELECT pg_sleep(0.2)").Error
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "57014" {
		t.Fatal("statement timeout not enforced", err)
	}
	held := d.DB.Begin()
	if held.Error != nil {
		t.Fatal(held.Error)
	}
	defer held.Rollback()
	if err = held.Exec("LOCK TABLE currency IN ACCESS EXCLUSIVE MODE").Error; err != nil {
		t.Fatal(err)
	}
	err = d.DB.Exec("UPDATE currency SET symbol=symbol WHERE code=?", "USD").Error
	if !errors.As(err, &pg) || pg.Code != "55P03" {
		t.Fatal("lock timeout not enforced", err)
	}
	if err = held.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	pool, _ := d.DB.DB()
	if pool.Stats().MaxOpenConnections != c.Data.Database.MaxOpenConns {
		t.Fatal("pool bound not applied")
	}
	deadline := time.Now().Add(3 * time.Second)
	for pool.Stats().MaxIdleTimeClosed == 0 {
		if time.Now().After(deadline) {
			t.Fatal("idle connections not reclaimed", pool.Stats())
		}
		time.Sleep(50 * time.Millisecond)
	}
}
func TestSQLLoggingPolicyAndRedaction(t *testing.T) {
	_, c := testutil.Postgres(t)
	previous := slog.Default()
	defer slog.SetDefault(previous)
	var output bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	for _, tt := range []struct {
		level string
		full  bool
		want  bool
	}{{"info", true, false}, {"debug", false, false}, {"debug", true, true}} {
		c.Log.Level = tt.level
		c.Data.Database.LogSQL = tt.full
		c.Data.Database.SlowSQLMS = 30000
		d, closeDB, err := data.NewData(c)
		if err != nil {
			t.Fatal(err)
		}
		output.Reset()
		err = d.DB.Exec("SELECT ?::text", "private-password-value").Error
		if err != nil {
			t.Fatal(err)
		}
		logs := output.String()
		closeDB()
		if strings.Contains(logs, "数据库查询") != tt.want || strings.Contains(logs, "private-password-value") {
			t.Fatal("SQL policy or parameter leak", logs)
		}
	}
	c.Log.Level = "info"
	c.Data.Database.LogSQL = false
	c.Data.Database.SlowSQLMS = 10
	d, closeDB, err := data.NewData(c)
	if err != nil {
		t.Fatal(err)
	}
	defer closeDB()
	output.Reset()
	if err = d.DB.Exec("SELECT pg_sleep(0.04)").Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "数据库慢查询") {
		t.Fatal("missing slow query warning", output.String())
	}
	output.Reset()
	err = d.DB.Exec("SELECT CAST(? AS integer)", "private-error-value").Error
	if err == nil {
		t.Fatal("expected SQL error")
	}
	logs := output.String()
	if !strings.Contains(logs, "数据库查询失败") || !strings.Contains(logs, "22P02") || strings.Contains(logs, "private-error-value") {
		t.Fatal("error logging or redaction", logs)
	}
	output.Reset()
	err = d.DB.Exec("SELECT CAST('private-literal-value' AS integer)").Error
	if err == nil {
		t.Fatal("expected literal SQL error")
	}
	if strings.Contains(output.String(), "private-literal-value") {
		t.Fatal("literal value leaked", output.String())
	}
}
