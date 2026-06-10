package agent_test

import (
	"context"
	"os"
	"testing"

	"opsone/internal/agent"
	"opsone/internal/config"
	"opsone/internal/domain"
	"opsone/internal/mock"
	"opsone/internal/store"
)

func testDB(t *testing.T) *store.DB {
	t.Helper()
	if os.Getenv("OPSONE_INTEGRATION") != "1" {
		t.Skip("set OPSONE_INTEGRATION=1 for integration tests")
	}
	cfg := config.Load()
	db, err := store.Open(cfg.MySQLDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestAgentCoreCycle(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	gen := mock.NewGenerator(db, 111)
	if _, err := gen.RunOnce(ctx); err != nil {
		t.Fatalf("mock: %v", err)
	}

	runner := agent.NewRunner(db)
	result, err := runner.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if result.CycleID == 0 {
		t.Fatal("expected cycle id")
	}
	if len(result.Products) != 11 {
		t.Errorf("products = %d, want 11", len(result.Products))
	}
	if err := agent.ValidateDryRun(ctx, db, result.CycleID); err != nil {
		t.Fatal(err)
	}

	var topup, zing *agent.ProductContext
	for i := range result.Products {
		p := &result.Products[i]
		switch p.Product.ProductCode {
		case "TOPUP_VINA":
			topup = p
		case "ZING":
			zing = p
		}
	}
	if topup == nil || zing == nil {
		t.Fatal("missing TOPUP_VINA or ZING in context")
	}
	if topup.Product.RoutingMode != domain.RoutingProvider {
		t.Errorf("TOPUP_VINA routing_mode = %q", topup.Product.RoutingMode)
	}
	if topup.Routing.Scope != "provider" {
		t.Errorf("TOPUP_VINA routing scope = %q", topup.Routing.Scope)
	}
	if zing.Routing.Scope != "sku" {
		t.Errorf("ZING routing scope = %q", zing.Routing.Scope)
	}
	for _, s := range topup.Scopes {
		if s.SKUCode != "" {
			t.Errorf("provider-mode scope should have empty sku, got %q", s.SKUCode)
		}
	}
	for _, s := range zing.Scopes {
		if s.SKUCode == "" {
			t.Error("sku-mode scope should have sku")
			break
		}
	}
}

func TestAgentCoreHealthStatusWritten(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	gen := mock.NewGenerator(db, 222)
	if _, err := gen.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	result, err := agent.NewRunner(db).RunOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var n int
	err = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM health_status_product WHERE cycle_id = ?`, result.CycleID,
	).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 11 {
		t.Errorf("health_status_product rows = %d, want 11", n)
	}
}
