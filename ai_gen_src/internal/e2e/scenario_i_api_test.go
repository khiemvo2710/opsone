package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"opsone/internal/tools"
)

// §10.9 — UpdateRouting → agent_change_log → rollback API → routing về before.
func TestE2E_ScenarioI_RollbackViaAPI(t *testing.T) {
	db, cfg := requireE2E(t)
	ctx := context.Background()
	srv := testAPIServer(t, db, cfg)
	reg := tools.NewRegistry(db)

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
		Routing:     map[string]float64{"ESALE": 55, "IMEDIA": 30, "SHOPPAY": 15},
		Reason:      "e2e §10.9 rollback via API",
	})
	if err != nil {
		t.Fatalf("UpdateRouting: %v", err)
	}
	if len(out.ChangeLogIDs) != 1 {
		t.Fatalf("change_log_ids = %v, want 1", out.ChangeLogIDs)
	}
	changeID := out.ChangeLogIDs[0]

	rec := apiPostJSON(t, srv, fmt.Sprintf("/api/v1/agent-changes/%d/rollback", changeID),
		`{"reason":"e2e rollback §10.9"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("rollback API: %d %s", rec.Code, rec.Body.String())
	}

	restored, err := db.GetRoutingForScope(ctx, "TOPUP_VINA", "")
	if err != nil {
		t.Fatalf("GetRoutingForScope after rollback: %v", err)
	}
	for _, r := range restored {
		want := beforeMap[r.ProviderCode]
		if r.TrafficPct != want {
			t.Errorf("%s = %.0f, want %.0f (seed/before)", r.ProviderCode, r.TrafficPct, want)
		}
	}
}

// Approve routing plan → UpdateRouting + agent_change_log.
func TestE2E_ApproveRoutingPlan(t *testing.T) {
	db, cfg := requireE2E(t)
	ctx := context.Background()
	srv := testAPIServer(t, db, cfg)

	before, err := db.GetRoutingForScope(ctx, "TOPUP_VINA", "")
	if err != nil {
		t.Fatalf("GetRoutingForScope: %v", err)
	}
	beforeESALE := float64(70)
	for _, r := range before {
		if r.ProviderCode == "ESALE" {
			beforeESALE = r.TrafficPct
		}
	}

	planJSON := map[string]any{
		"scope":        "provider",
		"sku":          "",
		"proposed_pct": map[string]float64{"ESALE": 65, "IMEDIA": 25, "SHOPPAY": 10},
	}
	planID, err := db.InsertRoutingPlan(ctx, nil, "TOPUP_VINA", "provider", "", planJSON, "pending_approve")
	if err != nil {
		t.Fatalf("InsertRoutingPlan: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `UPDATE routing_plans SET status = 'cancelled' WHERE id = ? AND status = 'pending_approve'`, planID)
	})

	rec := apiPostJSON(t, srv, fmt.Sprintf("/api/v1/routing-plans/%d/approve", planID),
		`{"proposed_pct":{"ESALE":55,"IMEDIA":30,"SHOPPAY":15}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		ChangeLogIDs []uint64 `json:"change_log_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.ChangeLogIDs) == 0 {
		t.Fatal("expected change_log_ids after approve")
	}

	after, err := db.GetRoutingForScope(ctx, "TOPUP_VINA", "")
	if err != nil {
		t.Fatalf("GetRoutingForScope after approve: %v", err)
	}
	var esaleAfter float64
	for _, r := range after {
		if r.ProviderCode == "ESALE" {
			esaleAfter = r.TrafficPct
		}
	}
	if esaleAfter != 55 {
		t.Errorf("ESALE after approve = %.0f, want 55 (custom admin input)", esaleAfter)
	}

	reg := tools.NewRegistry(db)
	restore := map[string]float64{"ESALE": beforeESALE, "IMEDIA": 20, "SHOPPAY": 10}
	for _, r := range before {
		if r.ProviderCode == "IMEDIA" {
			restore["IMEDIA"] = r.TrafficPct
		}
		if r.ProviderCode == "SHOPPAY" {
			restore["SHOPPAY"] = r.TrafficPct
		}
	}
	_, _ = reg.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product: "TOPUP_VINA", Scope: "provider",
		Routing: restore, Reason: "e2e cleanup restore seed",
	})
}

// Admin approve applies a large routing shift in one step.
func TestE2E_ApproveRoutingPlanLargeShift(t *testing.T) {
	db, cfg := requireE2E(t)
	ctx := context.Background()
	srv := testAPIServer(t, db, cfg)
	reg := tools.NewRegistry(db)

	_, err := reg.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     "DATA_VINA",
		Scope:       "sku",
		SKU:         "V100K",
		Routing:     map[string]float64{"ESALE": 10, "IMEDIA": 45, "SHOPPAY": 45},
		TriggerType: "admin_approve",
		Reason:      "e2e setup large-shift baseline",
	})
	if err != nil {
		t.Fatalf("setup routing: %v", err)
	}

	planJSON := map[string]any{
		"scope":        "sku",
		"sku":          "V100K",
		"proposed_pct": map[string]float64{"ESALE": 70, "IMEDIA": 15, "SHOPPAY": 15},
	}
	planID, err := db.InsertRoutingPlan(ctx, nil, "DATA_VINA", "sku", "V100K", planJSON, "pending_approve")
	if err != nil {
		t.Fatalf("InsertRoutingPlan: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `UPDATE routing_plans SET status = 'cancelled' WHERE id = ? AND status IN ('pending_approve','executed')`, planID)
		_, _ = reg.UpdateRouting(ctx, tools.UpdateRoutingInput{
			Product: "DATA_VINA", Scope: "sku", SKU: "V100K",
			TriggerType: "admin_approve",
			Routing:     map[string]float64{"ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20},
			Reason:      "e2e cleanup V100K seed",
		})
	})

	rec := apiPostJSON(t, srv, fmt.Sprintf("/api/v1/routing-plans/%d/approve", planID), `{}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve large shift: %d %s", rec.Code, rec.Body.String())
	}
}
