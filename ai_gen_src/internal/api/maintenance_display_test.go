package api

import (
	"database/sql"
	"testing"
	"time"

	"opsone/internal/store"
)

func TestMaintenanceOverviewSKUScopeWhenAllActiveInMaint(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	mws := []store.MaintenanceRow{
		{MaintenanceID: "a", ProviderCode: "ESALE", SKUCode: "VNP50", StartsAt: now.Add(-time.Hour), EndsAt: now.Add(6 * time.Minute), Reason: sql.NullString{String: "bt", Valid: true}},
		{MaintenanceID: "b", ProviderCode: "IMEDIA", SKUCode: "VNP50", StartsAt: now.Add(-time.Hour), EndsAt: now.Add(6 * time.Minute)},
	}
	routing := map[string]float64{"IMEDIA": 100}
	out := maintenanceOverview(mws, routing, "VNP50", now)
	if out == nil {
		t.Fatal("expected maintenance overview")
	}
	if scope, _ := out["scope_level"].(bool); !scope {
		t.Fatalf("scope_level = false, want true when sole active provider is in maintenance")
	}
	if code, _ := out["provider_code"].(string); code != "VNP50" {
		t.Fatalf("provider_code = %q, want VNP50", code)
	}
}

func TestMaintenanceOverviewHidesZeroRoutingProvider(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	mws := []store.MaintenanceRow{
		{MaintenanceID: "a", ProviderCode: "ESALE", SKUCode: "V100K", StartsAt: now.Add(-time.Hour), EndsAt: now.Add(45 * time.Minute)},
	}
	routing := map[string]float64{"ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20}
	out := maintenanceOverview(mws, routing, "V100K", now)
	if out == nil {
		t.Fatal("expected maintenance overview")
	}
	if scope, _ := out["scope_level"].(bool); scope {
		t.Fatal("scope_level should be false for partial maintenance")
	}
	if code, _ := out["provider_code"].(string); code != "ESALE" {
		t.Fatalf("provider_code = %q, want ESALE", code)
	}
	codes, _ := out["provider_codes"].([]string)
	if len(codes) != 1 || codes[0] != "ESALE" {
		t.Fatalf("provider_codes = %v, want [ESALE]", codes)
	}
}

func TestMaintenanceOverviewNilWhenOnlyZeroRoutingMaintained(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	mws := []store.MaintenanceRow{
		{MaintenanceID: "a", ProviderCode: "ESALE", SKUCode: "VNP50", StartsAt: now.Add(-time.Hour), EndsAt: now.Add(6 * time.Minute)},
	}
	routing := map[string]float64{"IMEDIA": 100}
	if out := maintenanceOverview(mws, routing, "VNP50", now); out != nil {
		t.Fatalf("expected nil when maintained provider has 0%% routing, got %#v", out)
	}
}
