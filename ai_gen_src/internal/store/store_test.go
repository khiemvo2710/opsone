package store_test

import (
	"context"
	"os"
	"testing"

	"opsone/internal/config"
	"opsone/internal/store"
)

func testDB(t *testing.T) *store.DB {
	t.Helper()
	if os.Getenv("OPSONE_INTEGRATION") != "1" {
		t.Skip("set OPSONE_INTEGRATION=1 and run MySQL with migrate+seed to run integration tests")
	}
	cfg := config.Load()
	db, err := store.Open(cfg.MySQLDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestCountProducts(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	n, err := db.CountProducts(ctx, true)
	if err != nil {
		t.Fatalf("CountProducts: %v", err)
	}
	if n != 11 {
		t.Errorf("expected 11 products, got %d", n)
	}
}

func TestGetProductZING(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	p, err := db.GetProductByCode(ctx, "ZING")
	if err != nil {
		t.Fatalf("GetProductByCode: %v", err)
	}
	if p.RoutingMode != "sku" {
		t.Errorf("ZING routing_mode = %q, want sku", p.RoutingMode)
	}
	if p.ServiceType != "card" {
		t.Errorf("ZING service_type = %q, want card", p.ServiceType)
	}
}

func TestListProvidersTOPUP_VINA(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	providers, err := db.ListProvidersForProduct(ctx, "TOPUP_VINA", true)
	if err != nil {
		t.Fatalf("ListProvidersForProduct: %v", err)
	}
	if len(providers) != 3 {
		t.Errorf("expected 3 active providers, got %d", len(providers))
	}
}

func TestRoutingSumTOPUP_VINA(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	sum, err := db.SumTrafficPct(ctx, "TOPUP_VINA", "")
	if err != nil {
		t.Fatalf("SumTrafficPct: %v", err)
	}
	if sum < 99.99 || sum > 100.01 {
		t.Errorf("TOPUP_VINA traffic sum = %.2f, want 100", sum)
	}

	rows, err := db.GetRoutingForScope(ctx, "TOPUP_VINA", "")
	if err != nil {
		t.Fatalf("GetRoutingForScope: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 routing rows, got %d", len(rows))
	}
}

func TestListSKUsZING(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	skus, err := db.ListSKUsForProduct(ctx, "ZING", true)
	if err != nil {
		t.Fatalf("ListSKUsForProduct: %v", err)
	}
	if len(skus) != 4 {
		t.Errorf("expected 4 SKUs for ZING, got %d", len(skus))
	}
}

func TestAgentLocale(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	locale, err := db.GetAgentSettingsLocale(ctx)
	if err != nil {
		t.Fatalf("GetAgentSettingsLocale: %v", err)
	}
	if locale != "vi-VN" {
		t.Errorf("agent_locale = %q, want vi-VN", locale)
	}
}
