package data

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log/slog"
	"peptide-site/internal/conf"
	"regexp"
	"time"
)

// sqlLogger 将数据库日志交给服务日志处理器，绑定参数和数据库错误详情不进入日志。
type sqlLogger struct {
	mode logger.LogLevel
	full bool
	slow time.Duration
}

func newSQLLogger(c conf.Config) *sqlLogger {
	return &sqlLogger{mode: logger.Info, full: c.Data.Database.LogSQL && c.Log.Level == "debug", slow: time.Duration(c.Data.Database.SlowSQLMS) * time.Millisecond}
}
func (l *sqlLogger) LogMode(mode logger.LogLevel) logger.Interface {
	next := *l
	next.mode = mode
	return &next
}
func (l *sqlLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.mode >= logger.Info {
		slog.InfoContext(ctx, "数据库信息")
	}
}
func (l *sqlLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.mode >= logger.Warn {
		slog.WarnContext(ctx, "数据库警告")
	}
}
func (l *sqlLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.mode >= logger.Error {
		slog.ErrorContext(ctx, "数据库错误")
	}
}

// ParamsFilter 禁止 GORM 将业务参数插入 SQL 日志，包括密码哈希、验证码和正文。
func (l *sqlLogger) ParamsFilter(ctx context.Context, query string, params ...any) (string, []any) {
	return query, nil
}

var sqlLiteral = regexp.MustCompile(`'(?:[^']|'')*'`)

func (l *sqlLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.mode == logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	failed := err != nil && !errors.Is(err, gorm.ErrRecordNotFound)
	level := slog.LevelDebug
	message := "数据库查询"
	if failed && l.mode >= logger.Error {
		level = slog.LevelError
		message = "数据库查询失败"
	} else if elapsed > l.slow && l.mode >= logger.Warn {
		level = slog.LevelWarn
		message = "数据库慢查询"
	} else if !l.full || l.mode < logger.Info {
		return
	}
	if !slog.Default().Enabled(ctx, level) {
		return
	}
	query, rows := fc()
	args := []any{"sql", sqlLiteral.ReplaceAllString(query, "?"), "elapsed", elapsed, "rows", rows}
	var pg *pgconn.PgError
	if failed && errors.As(err, &pg) {
		args = append(args, "sqlstate", pg.Code)
	}
	slog.Log(ctx, level, message, args...)
}
