package mock_test

import (
	"context"
	"os"
	"testing"

	"opsone/internal/agent"
	"opsone/internal/config"
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

func TestMockGeneratorRunOnce(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	before, err := db.CountMockMetrics(ctx)
	if err != nil {
		t.Fatal(err)
	}

	gen := mock.NewGenerator(db, 123)
	n, err := gen.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if n == 0 {
		t.Fatal("expected metric rows")
	}

	after, err := db.CountMockMetrics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after <= before {
		t.Errorf("mock_metrics should increase: before=%d after=%d", before, after)
	}
}

func TestAgentDryRunCycle(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	gen := mock.NewGenerator(db, 456)
	if _, err := gen.RunOnce(ctx); err != nil {
		t.Fatalf("mock seed: %v", err)
	}

	runner := agent.NewDryRunner(db, nil)
	cycleID, rows, err := runner.RunOnce(ctx)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if cycleID == 0 || rows == 0 {
		t.Fatalf("cycleID=%d rows=%d", cycleID, rows)
	}
	if err := agent.ValidateDryRun(ctx, db, cycleID); err != nil {
		t.Fatal(err)
	}
}

func TestEsaleDegradingInDB(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `UPDATE agent_settings SET mock_scenario = 'esale_degrading' WHERE id = 1`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `UPDATE agent_settings SET mock_scenario = 'normal' WHERE id = 1`)
	})

	gen := mock.NewGenerator(db, 789)
	for i := 0; i < 3; i++ {
		if _, err := gen.RunOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}

	m, ok, err := db.GetLatestMockMetric(ctx, "TOPUP_VINA", "", "ESALE")
	if err != nil || !ok {
		t.Fatalf("latest metric: ok=%v err=%v", ok, err)
	}
	if m.SuccessRate > 90 {
		t.Errorf("TOPUP_VINA ESALE should degrade, success=%.2f", m.SuccessRate)
	}
}
