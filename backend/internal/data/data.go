// Package data implements PostgreSQL persistence.
package data

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"strconv"

	"peptide-site/internal/conf"
)

// Data owns the database connection pool.
type Data struct{ DB *gorm.DB }

// NewData opens PostgreSQL and returns a cleanup function.
func NewData(c conf.Config) (*Data, func(), error) {
	connection, err := pgx.ParseConfig(c.Data.Database.DSN)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid database configuration")
	}
	connection.ConnectTimeout = c.Data.Database.ConnectTimeout
	// 通过连接启动参数对每条连接设置执行与等锁超时，配置值优先于 DSN。
	connection.RuntimeParams["statement_timeout"] = strconv.FormatInt(c.Data.Database.StatementTimeout.Milliseconds(), 10)
	connection.RuntimeParams["lock_timeout"] = strconv.FormatInt(c.Data.Database.LockTimeout.Milliseconds(), 10)
	sqlPool := stdlib.OpenDB(*connection)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlPool}), &gorm.Config{Logger: newSQLLogger(c)})
	if err != nil {
		_ = sqlPool.Close()
		return nil, nil, fmt.Errorf("database connection unavailable")
	}
	pool, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get postgres pool: %w", err)
	}
	pool.SetMaxOpenConns(c.Data.Database.MaxOpenConns)
	pool.SetMaxIdleConns(c.Data.Database.MaxIdleConns)
	pool.SetConnMaxLifetime(c.Data.Database.ConnMaxLifetime)
	pool.SetConnMaxIdleTime(c.Data.Database.ConnMaxIdleTime)
	return &Data{DB: db}, func() { _ = pool.Close() /* Shutdown cannot return an error to Wire's cleanup caller. */ }, nil
}

// Check pings the connection pool using the request deadline.
func (d *Data) Check(ctx context.Context) error {
	pool, err := d.DB.DB()
	if err != nil {
		return err
	}
	return pool.PingContext(ctx)
}
