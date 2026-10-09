// Package data implements PostgreSQL persistence.
package data

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"peptide-site/internal/conf"
)

// Data owns the database connection pool.
type Data struct{ DB *gorm.DB }

// NewData opens PostgreSQL and returns a cleanup function.
func NewData(c conf.Config) (*Data, func(), error) {
	db, err := gorm.Open(postgres.Open(c.Data.Database.DSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, nil, fmt.Errorf("connect postgres: %w", err)
	}
	pool, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get postgres pool: %w", err)
	}
	pool.SetMaxOpenConns(c.Data.Database.MaxOpenConns)
	pool.SetMaxIdleConns(c.Data.Database.MaxIdleConns)
	pool.SetConnMaxLifetime(c.Data.Database.ConnMaxLifetime)
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
