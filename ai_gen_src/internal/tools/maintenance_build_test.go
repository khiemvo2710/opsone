package tools

import (
	"testing"
	"time"

	"opsone/internal/store"
)

func TestMaintenanceInWindow_matchesDashboard(t *testing.T) {
	now := time.Date(2026, 6, 10, 18, 0, 0, 0, time.Local)
	starts := now.Add(-time.Hour)
	ends := now.Add(time.Hour)
	if !MaintenanceInWindow(starts, ends, now) {
		t.Fatal("expected in window")
	}
	if MaintenanceInWindow(starts, now.Add(-time.Minute), now) {
		t.Fatal("expired window should be out")
	}
}

func TestBuildMaintenanceOutput_filtersSKU(t *testing.T) {
	now := time.Now()
	rows := []store.MaintenanceWindow{
		{
			MaintenanceID: "m1", ProductCode: "GARENA", ProviderCode: "ESALE", SKUCode: "10000",
			StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour), Status: "active",
		},
		{
			MaintenanceID: "m2", ProductCode: "GARENA", ProviderCode: "IMEDIA", SKUCode: "20000",
			StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour), Status: "active",
		},
	}
	out := BuildMaintenanceOutput(rows, now, "", "10000")
	if len(out.Active) != 1 || out.Active[0].SKUCode != "10000" {
		t.Fatalf("got %+v", out.Active)
	}
}
