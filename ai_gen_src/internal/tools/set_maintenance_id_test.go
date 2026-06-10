package tools

import (
	"testing"
	"time"
)

func TestBuildMaintenanceIDUniquePerProvider(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 57, 39, 500_000_000, time.UTC)
	providers := []string{"ESALE", "IMEDIA"}
	seen := map[string]struct{}{}
	for i, provider := range providers {
		id := buildMaintenanceID(provider, i, now)
		if len(id) > 32 {
			t.Fatalf("id too long: %q (%d)", id, len(id))
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate maintenance_id %q for provider %s", id, provider)
		}
		seen[id] = struct{}{}
	}
}

func TestBuildMaintenanceIDIncludesProvider(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 57, 39, 0, time.UTC)
	id := buildMaintenanceID("ESALE", 0, now)
	if id == "20260610-M095739" {
		t.Fatal("legacy collision-prone id format")
	}
	if id[:2] != "MT" {
		t.Fatalf("expected MT prefix, got %q", id)
	}
}
