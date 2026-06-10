package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"opsone/internal/tools"
)

// §10.1 / §12 — TOPUP_VINA provider routing seed (ESALE 70 / IMEDIA 20 / SHOPPAY 10).
func TestE2E_ScenarioA_TopupProviderRouting(t *testing.T) {
	db, cfg := requireE2E(t)
	ctx := context.Background()
	reg := tools.NewRegistry(db)

	out, err := reg.GetRouting(ctx, tools.GetRoutingInput{Product: "TOPUP_VINA"})
	if err != nil {
		t.Fatalf("GetRouting: %v", err)
	}
	if out.Scope != "provider" {
		t.Errorf("scope = %q, want provider", out.Scope)
	}
	want := map[string]float64{"ESALE": 70, "IMEDIA": 20, "SHOPPAY": 10}
	for k, v := range want {
		if out.Routing[k] != v {
			t.Errorf("%s = %.0f, want %.0f", k, out.Routing[k], v)
		}
	}
	_ = cfg
}

// §10.8 / §12 — Health Status API trả green|yellow|red.
func TestE2E_ScenarioH_HealthStatusAPI(t *testing.T) {
	db, cfg := requireE2E(t)
	srv := testAPIServer(t, db, cfg)

	rec := apiGet(t, srv, "/api/v1/health-status")
	if rec.Code != http.StatusOK {
		t.Fatalf("health-status: %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	hs, ok := body["health_status"].(string)
	if !ok || hs == "" {
		t.Errorf("health_status = %v, want green|yellow|red", body["health_status"])
	}
	switch hs {
	case "green", "yellow", "red":
	default:
		t.Errorf("health_status = %q, want green|yellow|red", hs)
	}
}

// §12 — Catalog 11 product trong MySQL.
func TestE2E_DoD_ElevenProducts(t *testing.T) {
	db, _ := requireE2E(t)
	ctx := context.Background()
	n := countProducts(ctx, db)
	if n != 11 {
		t.Errorf("products = %d, want 11", n)
	}
}

// §9.0 / §12 — Dashboard overview endpoint.
func TestE2E_DoD_DashboardOverview(t *testing.T) {
	db, cfg := requireE2E(t)
	srv := testAPIServer(t, db, cfg)

	rec := apiGet(t, srv, "/api/v1/dashboard/overview")
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard/overview: %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Rows []struct {
			ProductCode string `json:"product_code"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Rows) == 0 {
		t.Error("expected routing rows in dashboard overview")
	}
}
