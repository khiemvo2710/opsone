package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

// OpenWithRetry connects to MySQL, retrying until maxWait elapses (GreenNode cold start / vDB wake).
func OpenWithRetry(ctx context.Context, dsn string, maxWait time.Duration) (*DB, error) {
	deadline := time.Now().Add(maxWait)
	var lastErr error
	for attempt := 1; ; attempt++ {
		db, err := Open(dsn)
		if err == nil {
			if attempt > 1 {
				log.Printf("mysql: connected after %d attempts", attempt)
			}
			return db, nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			break
		}
		log.Printf("mysql: %v (retry %d)", err, attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, lastErr
}

// Ping checks database connectivity.
func (db *DB) Ping() error {
	return db.DB.Ping()
}
