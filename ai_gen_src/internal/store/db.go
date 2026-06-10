package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB wraps *sql.DB with OpsOne store helpers.
type DB struct {
	*sql.DB
}

// Open connects to MySQL using a DSN. Caller must call Close.
func Open(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return &DB{DB: db}, nil
}

// Ping checks database connectivity.
func (db *DB) Ping() error {
	return db.DB.Ping()
}
