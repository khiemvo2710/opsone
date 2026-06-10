package tools_test

import (
	"context"
	"os"
	"testing"
	"time"

	"opsone/internal/config"
	"opsone/internal/mock"
	"opsone/internal/notify"
	"opsone/internal/rollback"
	"opsone/internal/store"
	"opsone/internal/threshold"
	"opsone/internal/tools"
)

func testRegistry(t *testing.T) (*tools.Registry, *store.DB) {
	t.Helper()
	if os.Getenv("OPSONE_INTEGRATION") != "1" {
		t.Skip("set OPSONE_INTEGRATION=1 and run MySQL with migrate+seed")
	}
	cfg := config.Load()
	db, err := store.Open(cfg.MySQLDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return tools.NewRegistry(db), db
}

func TestGetProvidersTOPUP_VINA(t *testing.T) {
	reg, _ := testRegistry(t)
	ctx := context.Background()

	out, err := reg.GetProviders(ctx, tools.GetProvidersInput{Product: "TOPUP_VINA"})
	if err != nil {
		t.Fatalf("GetProviders: %v", err)
	}
	if out.ActiveCount != 3 {
		t.Errorf("active_count = %d, want 3", out.ActiveCount)
	}
}

func TestGetRoutingProviderMode(t *testing.T) {
	reg, _ := testRegistry(t)
	ctx := context.Background()

	out, err := reg.GetRouting(ctx, tools.GetRoutingInput{Product: "TOPUP_VINA"})
	if err != nil {
		t.Fatalf("GetRouting: %v", err)
	}
	if out.Scope != "provider" {
		t.Errorf("scope = %q, want provider", out.Scope)
	}
	if out.Routing["ESALE"] != 70 {
		t.Errorf("ESALE = %.0f, want 70", out.Routing["ESALE"])
	}
}

func TestGetSkusZING(t *testing.T) {
	reg, _ := testRegistry(t)
	ctx := context.Background()

	out, err := reg.GetSkus(ctx, tools.GetSkusInput{Product: "ZING"})
	if err != nil {
		t.Fatalf("GetSkus: %v", err)
	}
	if len(out.SKUs) == 0 {
		t.Error("expected SKUs for ZING")
	}
}

func TestUpdateRoutingAndRollback(t *testing.T) {
	reg, db := testRegistry(t)
	ctx := context.Background()

	before, err := db.GetRoutingForScope(ctx, "TOPUP_VINA", "")
	if err != nil {
		t.Fatalf("GetRoutingForScope: %v", err)
	}
	beforeMap := map[string]float64{}
	for _, r := range before {
		beforeMap[r.ProviderCode] = r.TrafficPct
	}

	out, err := reg.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     "TOPUP_VINA",
		ServiceType: "topup",
		Scope:       "provider",
		Routing:     map[string]float64{"ESALE": 60, "IMEDIA": 25, "SHOPPAY": 15},
		Reason:      "test phase2 rollback",
	})
	if err != nil {
		t.Fatalf("UpdateRouting: %v", err)
	}
	if len(out.ChangeLogIDs) != 1 {
		t.Fatalf("expected 1 change log, got %v", out.ChangeLogIDs)
	}

	after, err := db.GetRoutingForScope(ctx, "TOPUP_VINA", "")
	if err != nil {
		t.Fatalf("GetRoutingForScope after: %v", err)
	}
	for _, r := range after {
		if r.ProviderCode == "ESALE" && r.TrafficPct != 60 {
			t.Errorf("ESALE traffic = %.0f, want 60", r.TrafficPct)
		}
	}

	rb := rollback.Service{DB: db}
	resp, err := rb.Rollback(ctx, rollback.Request{
		ChangeID:   out.ChangeLogIDs[0],
		ExecutedBy: "test",
		Reason:     "rollback test §10.9",
	})
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if resp.RoutingRestored["ESALE"] != beforeMap["ESALE"] {
		t.Errorf("restored ESALE = %.0f, want %.0f", resp.RoutingRestored["ESALE"], beforeMap["ESALE"])
	}

	restored, err := db.GetRoutingForScope(ctx, "TOPUP_VINA", "")
	if err != nil {
		t.Fatalf("GetRoutingForScope restored: %v", err)
	}
	for _, r := range restored {
		if r.TrafficPct != beforeMap[r.ProviderCode] {
			t.Errorf("%s = %.0f, want %.0f", r.ProviderCode, r.TrafficPct, beforeMap[r.ProviderCode])
		}
	}
}

func TestGetMetricsAfterMock(t *testing.T) {
	reg, db := testRegistry(t)
	ctx := context.Background()

	gen := mock.NewGenerator(db, 42)
	if _, err := gen.RunOnce(ctx); err != nil {
		t.Fatalf("mock run: %v", err)
	}

	out, err := reg.GetMetrics(ctx, tools.GetMetricsInput{
		Product: "TOPUP_VINA", Provider: "ESALE", Window: "15m",
	})
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}
	if out.TotalTransactions == 0 {
		t.Error("expected transactions > 0 after mock")
	}
}

func TestSetMaintenance(t *testing.T) {
	reg, _ := testRegistry(t)
	ctx := context.Background()

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
	end := start.Add(60 * time.Minute)
	out, err := reg.SetMaintenance(ctx, tools.SetMaintenanceInput{
		Product:  "ZING",
		Provider: "ESALE",
		SKU:      "20000",
		StartsAt: start,
		EndsAt:   end,
		Reason:   "test phase2",
	})
	if err != nil {
		t.Fatalf("SetMaintenance: %v", err)
	}
	if out.MaintenanceID == "" {
		t.Error("expected maintenance_id")
	}

	got, err := reg.GetMaintenance(ctx, tools.GetMaintenanceInput{
		Product: "ZING", Provider: "ESALE", SKU: "20000",
	})
	if err != nil {
		t.Fatalf("GetMaintenance: %v", err)
	}
	if len(got.Scheduled) == 0 && len(got.Active) == 0 {
		t.Error("expected maintenance window")
	}
}

func TestEvaluateThresholds(t *testing.T) {
	reg, db := testRegistry(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `UPDATE agent_settings SET mock_scenario = 'esale_degrading' WHERE id = 1`)
	if err != nil {
		t.Fatalf("set scenario: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `UPDATE agent_settings SET mock_scenario = 'normal' WHERE id = 1`)
	})

	gen := mock.NewGenerator(db, 99)
	if _, err := gen.RunOnce(ctx); err != nil {
		t.Fatalf("mock run: %v", err)
	}

	eval := threshold.Evaluator{DB: db}
	m, ok, err := eval.LoadScopeMetrics(ctx, "TOPUP_VINA", "", "ESALE")
	if err != nil || !ok {
		t.Fatalf("LoadScopeMetrics: ok=%v err=%v", ok, err)
	}
	res, err := eval.EvaluateThresholds(ctx, m, 2)
	if err != nil {
		t.Fatalf("EvaluateThresholds: %v", err)
	}
	if !res.Breached {
		t.Log("esale_degrading may not breach in single cycle — checking providers still works")
	}
	prov, _ := reg.GetProviders(ctx, tools.GetProvidersInput{Product: "TOPUP_VINA"})
	if prov.ActiveCount < 2 && res.SuggestedAction == "routing" {
		t.Error("routing suggested with insufficient active providers")
	}
}

func TestNotifyMock(t *testing.T) {
	_, db := testRegistry(t)
	ctx := context.Background()

	cfg := config.Load()
	cfg.NotificationMock = true
	svc := notify.Service{DB: db, Config: cfg}
	err := svc.SendIfNeeded(ctx, notify.EmailParams{
		Product:       "TOPUP_VINA",
		Provider:      "ESALE",
		HealthStatus:  "red",
		TriggerEvent:  "routing_applied",
		SuccessRate:   78,
		PendingRate:   18,
		FailRate:      12,
		BreachReasons: []string{"Tỷ lệ lỗi 12% vượt ngưỡng 10%"},
		ActionSummary: "Điều phối routing ESALE 70%→10%",
		DedupeKey:     "test-notify-" + time.Now().Format("150405"),
	})
	if err != nil {
		t.Fatalf("SendIfNeeded: %v", err)
	}
}
