package store

import "testing"

func TestResolveEffectiveScopeAutoProductPriority(t *testing.T) {
	byKey := map[string]ScopeAutoConfig{
		scopeAutoKey("GARENA", ""): {
			ProductCode: "GARENA",
			SKUCode:     "",
			AutoAction:  "auto",
		},
		scopeAutoKey("GARENA", "10000"): {
			ProductCode: "GARENA",
			SKUCode:     "10000",
			AutoAction:  "recommend_only",
		},
	}

	got := ResolveEffectiveScopeAuto(byKey, "GARENA", "10000")
	if got.AutoAction != "auto" {
		t.Fatalf("expected product-level auto, got %q", got.AutoAction)
	}
}

func TestResolveEffectiveScopeAutoSkuFallback(t *testing.T) {
	byKey := map[string]ScopeAutoConfig{
		scopeAutoKey("GARENA", "10000"): {
			ProductCode: "GARENA",
			SKUCode:     "10000",
			AutoAction:  "time_window",
			WindowStart: "2026-06-10T08:00",
			WindowEnd:   "2026-06-10T18:00",
		},
	}

	got := ResolveEffectiveScopeAuto(byKey, "GARENA", "10000")
	if got.AutoAction != "time_window" {
		t.Fatalf("expected sku-level time_window, got %q", got.AutoAction)
	}
}

func TestResolveEffectiveScopeAutoProviderScope(t *testing.T) {
	byKey := map[string]ScopeAutoConfig{
		scopeAutoKey("TOPUP_VINA", ""): {
			ProductCode: "TOPUP_VINA",
			SKUCode:     "",
			AutoAction:  "auto",
		},
	}

	got := ResolveEffectiveScopeAuto(byKey, "TOPUP_VINA", "")
	if got.AutoAction != "auto" {
		t.Fatalf("expected provider-scope auto, got %q", got.AutoAction)
	}
}
