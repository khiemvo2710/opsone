package e2e_test

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"opsone/internal/api"
	"opsone/internal/config"
	"opsone/internal/store"
)

func requireE2E(t *testing.T) (*store.DB, config.Config) {
	t.Helper()
	if os.Getenv("OPSONE_INTEGRATION") != "1" {
		t.Skip("set OPSONE_INTEGRATION=1 (MySQL + seed) for E2E tests")
	}
	cfg := config.Load()
	cfg.DevAuthBypass = true
	db, err := store.Open(cfg.MySQLDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, cfg
}

func testAPIServer(t *testing.T, db *store.DB, cfg config.Config) *api.Server {
	t.Helper()
	return api.NewServer(db, cfg)
}

func apiGet(t *testing.T, srv *api.Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func apiPostJSON(t *testing.T, srv *api.Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpsOne-Role", "admin")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func apiPutJSON(t *testing.T, srv *api.Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("PUT", path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpsOne-Role", "admin")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func countProducts(ctx context.Context, db *store.DB) int {
	n, _ := db.CountProducts(ctx, false)
	return n
}
