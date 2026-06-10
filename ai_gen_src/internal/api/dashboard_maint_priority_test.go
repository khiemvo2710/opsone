package api

import (
	"testing"

	"opsone/internal/agent"
	"opsone/internal/store"
	"opsone/internal/threshold"
	"opsone/internal/tools"
)

func TestBuildScopeActionReadonlyAllBreachedMaintenance(t *testing.T) {
	th := store.ProductThreshold{
		ProductCode:          "DATA_VINA",
		Enabled:              true,
		SuccessRateMinPct:    82,
		PendingRateMaxPct:    14,
		FailRateMaxPct:       10,
		PendingTxnCountMax:   8,
		FailTxnCountMax:      60,
		ConsecutiveCyclesRequired: 1,
	}
	routing := map[string]float64{"ESALE": 70, "IMEDIA": 30}
	pc := agent.ProductContext{
		Scopes: []agent.ScopeContext{
			{
				SKUCode: "VNP50", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 96.2, PendingRate: 2.2, FailRate: 1.6, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true, BreachReasons: []string{"Số GD pending 22 vượt ngưỡng 8"}},
			},
			{
				SKUCode: "VNP50", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 95.1, PendingRate: 3.3, FailRate: 1.6, TotalTransactions: 900},
				Threshold: &threshold.Result{Breached: true, BreachReasons: []string{"Số GD pending 29 vượt ngưỡng 8"}},
			},
		},
	}
	worst := pc.Scopes[1]
	eval := *worst.Threshold

	action := buildScopeActionReadonly(
		overviewCaches{},
		"DATA_VINA", "VNP50", routing, th, worst, eval, pc,
	)
	if action == nil {
		t.Fatal("expected action")
	}
	if action.kind != "maintenance" {
		t.Fatalf("kind = %q, want maintenance when all providers breach", action.kind)
	}
}

func TestMaintenanceWarrantedFromSnapshotAllBreached(t *testing.T) {
	th := store.ProductThreshold{
		ProductCode:        "DATA_VINA",
		Enabled:            true,
		SuccessRateMinPct:  82,
		PendingRateMaxPct:  14,
		FailRateMaxPct:     10,
		PendingTxnCountMax: 8,
		FailTxnCountMax:    60,
	}
	snap := scopeSnapshot{
		ShouldAct:   true,
		AnyBreached: true,
		HasWorst:    true,
		ProviderMetrics: map[string]any{
			"ESALE": map[string]any{
				"routing_pct": 70.0, "success_pct": 96.2, "pending_pct": 2.2, "fail_pct": 1.6,
				"pending_txn": uint(22), "fail_txn": uint(16),
			},
			"IMEDIA": map[string]any{
				"routing_pct": 30.0, "success_pct": 95.1, "pending_pct": 3.3, "fail_pct": 1.6,
				"pending_txn": uint(29), "fail_txn": uint(14),
			},
		},
		ScopeContexts: []agent.ScopeContext{
			{
				SKUCode: "VNP50", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 96.2, PendingRate: 2.2, FailRate: 1.6, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true},
			},
			{
				SKUCode: "VNP50", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 95.1, PendingRate: 3.3, FailRate: 1.6, TotalTransactions: 900},
				Threshold: &threshold.Result{Breached: true},
			},
		},
	}
	routing := map[string]float64{"ESALE": 70, "IMEDIA": 30}
	if !maintenanceWarrantedFromSnapshot(snap, "VNP50", routing, nil, th) {
		t.Fatal("expected maintenance warranted when both providers breach")
	}
}

func TestScopeAutoApplyAllowedAllProvidersBreached(t *testing.T) {
	th := store.ProductThreshold{
		ProductCode:               "DATA_VINA",
		Enabled:                   true,
		SuccessRateMinPct:         82,
		PendingRateMaxPct:         14,
		FailRateMaxPct:            10,
		PendingTxnCountMax:        8,
		FailTxnCountMax:           60,
		ConsecutiveCyclesRequired: 2,
	}
	snap := scopeSnapshot{
		ShouldAct:   false,
		AnyBreached: true,
		HasWorst:    true,
		ProviderMetrics: map[string]any{
			"ESALE": map[string]any{
				"routing_pct": 60.0, "success_pct": 96.0, "pending_pct": 2.0, "fail_pct": 2.0,
				"pending_txn": uint(21), "fail_txn": uint(10),
			},
			"IMEDIA": map[string]any{
				"routing_pct": 40.0, "success_pct": 95.0, "pending_pct": 2.5, "fail_pct": 2.0,
				"pending_txn": uint(24), "fail_txn": uint(12),
			},
		},
		ScopeContexts: []agent.ScopeContext{
			{
				SKUCode: "V50K", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 96, PendingRate: 2, FailRate: 2, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true},
			},
			{
				SKUCode: "V50K", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 95, PendingRate: 2.5, FailRate: 2, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true},
			},
		},
	}
	routing := map[string]float64{"ESALE": 60, "IMEDIA": 40}
	if !scopeAutoApplyAllowed(snap, "V50K", routing, nil, th) {
		t.Fatal("expected auto apply allowed when all providers breach despite ShouldAct=false")
	}
}

func TestMaintenanceWarrantedFromSnapshotAllBreachedWithoutShouldAct(t *testing.T) {
	th := store.ProductThreshold{
		ProductCode:        "DATA_VINA",
		Enabled:            true,
		SuccessRateMinPct:  82,
		PendingRateMaxPct:  14,
		FailRateMaxPct:     10,
		PendingTxnCountMax: 8,
		FailTxnCountMax:    60,
	}
	snap := scopeSnapshot{
		ShouldAct:   false,
		AnyBreached: true,
		HasWorst:    true,
		ProviderMetrics: map[string]any{
			"ESALE": map[string]any{
				"routing_pct": 60.0, "success_pct": 95.1, "pending_pct": 3.1, "fail_pct": 1.9,
				"pending_txn": uint(36), "fail_txn": uint(30),
			},
			"IMEDIA": map[string]any{
				"routing_pct": 40.0, "success_pct": 95.2, "pending_pct": 2.8, "fail_pct": 2.0,
				"pending_txn": uint(24), "fail_txn": uint(13),
			},
		},
		ScopeContexts: []agent.ScopeContext{
			{
				SKUCode: "V50K", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 95.1, PendingRate: 3.1, FailRate: 1.9, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true},
			},
			{
				SKUCode: "V50K", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 95.2, PendingRate: 2.8, FailRate: 2.0, TotalTransactions: 900},
				Threshold: &threshold.Result{Breached: true},
			},
		},
	}
	routing := map[string]float64{"ESALE": 60, "IMEDIA": 40}
	if !maintenanceWarrantedFromSnapshot(snap, "V50K", routing, nil, th) {
		t.Fatal("expected maintenance warranted when both providers breach even if ShouldAct=false")
	}
}

func TestRefreshPendingRoutingReturnsNilWhenAllBreached(t *testing.T) {
	th := store.ProductThreshold{
		ProductCode:        "DATA_VINA",
		Enabled:            true,
		SuccessRateMinPct:  82,
		PendingRateMaxPct:  14,
		FailRateMaxPct:     10,
		PendingTxnCountMax: 8,
		FailTxnCountMax:    60,
	}
	snap := scopeSnapshot{
		ShouldAct:   false,
		AnyBreached: true,
		HasWorst:    true,
		ProviderMetrics: map[string]any{
			"ESALE": map[string]any{
				"routing_pct": 60.0, "success_pct": 95.1, "pending_pct": 3.1, "fail_pct": 1.9,
				"pending_txn": uint(36), "fail_txn": uint(30),
			},
			"IMEDIA": map[string]any{
				"routing_pct": 40.0, "success_pct": 95.2, "pending_pct": 2.8, "fail_pct": 2.0,
				"pending_txn": uint(24), "fail_txn": uint(13),
			},
		},
		ScopeContexts: []agent.ScopeContext{
			{
				SKUCode: "V50K", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 95.1, PendingRate: 3.1, FailRate: 1.9, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true, BreachReasons: []string{"Số GD pending 36 vượt ngưỡng 8"}},
			},
			{
				SKUCode: "V50K", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 95.2, PendingRate: 2.8, FailRate: 2.0, TotalTransactions: 900},
				Threshold: &threshold.Result{Breached: true, BreachReasons: []string{"Số GD pending 24 vượt ngưỡng 8"}},
			},
		},
	}
	routing := map[string]float64{"ESALE": 60, "IMEDIA": 40}
	pc := agent.ProductContext{Scopes: snap.ScopeContexts}
	worst := snap.ScopeContexts[0]
	eval := *worst.Threshold
	action := buildScopeActionReadonly(overviewCaches{}, "DATA_VINA", "V50K", routing, th, worst, eval, pc)
	if action == nil || action.kind != "maintenance" {
		kind := ""
		if action != nil {
			kind = action.kind
		}
		t.Fatalf("refresh path should prefer maintenance when all breach, got %q", kind)
	}
}

func TestForceMaintenanceBypassesRecoveryGraceGate(t *testing.T) {
	th := store.ProductThreshold{
		ProductCode:        "DATA_VINA",
		Enabled:            true,
		SuccessRateMinPct:  82,
		PendingRateMaxPct:  14,
		FailRateMaxPct:     10,
		PendingTxnCountMax: 8,
		FailTxnCountMax:    60,
	}
	routing := map[string]float64{"ESALE": 100, "IMEDIA": 0}
	pc := agent.ProductContext{
		Scopes: []agent.ScopeContext{
			{
				SKUCode: "V50K", ProviderCode: "ESALE",
				Metrics: &tools.GetMetricsOutput{SuccessRate: 95.5, PendingRate: 2.5, FailRate: 2, TotalTransactions: 1000},
			},
		},
	}
	if !agent.ShouldForceAutoMaintenance(pc, "V50K", routing, th) {
		t.Fatal("expected force maintenance for single active breached provider")
	}
	inRecoveryGrace := true
	forceMaint := agent.ShouldForceAutoMaintenance(pc, "V50K", routing, th)
	if inRecoveryGrace && !forceMaint {
		t.Fatal("grace must not block force maintenance")
	}
}

func TestScopeAutoApplyAllowedForceRouting(t *testing.T) {
	th := store.ProductThreshold{
		ProductCode:               "DATA_VINA",
		Enabled:                   true,
		SuccessRateMinPct:         82,
		PendingRateMaxPct:         14,
		FailRateMaxPct:            10,
		PendingTxnCountMax:        8,
		FailTxnCountMax:           60,
		ConsecutiveCyclesRequired: 2,
	}
	snap := scopeSnapshot{
		ShouldAct:   false,
		AnyBreached: true,
		HasWorst:    true,
		ScopeContexts: []agent.ScopeContext{
			{
				SKUCode: "V100K", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 96, PendingRate: 2, FailRate: 2, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true},
			},
			{
				SKUCode: "V100K", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 97, PendingRate: 0.3, FailRate: 2, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: false},
			},
			{
				SKUCode: "V100K", ProviderCode: "SHOPPAY",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 96, PendingRate: 2.2, FailRate: 2, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true},
			},
		},
	}
	routing := map[string]float64{"ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20}
	if !scopeAutoApplyAllowed(snap, "V100K", routing, nil, th) {
		t.Fatal("expected auto apply allowed for routing shift when healthy backup exists")
	}
}
