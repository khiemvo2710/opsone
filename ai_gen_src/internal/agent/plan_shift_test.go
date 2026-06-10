package agent

import (
	"math"
	"testing"

	"opsone/internal/store"
	"opsone/internal/threshold"
	"opsone/internal/tools"
)

func defaultTestThreshold() store.ProductThreshold {
	return store.ProductThreshold{
		Enabled:              true,
		SuccessRateMinPct:    80,
		PendingRateMaxPct:    15,
		FailRateMaxPct:       10,
		FailTxnCountMax:      50,
		PendingTxnCountMax:   5,
	}
}

func TestProposeShiftSuccessBreachHardCut(t *testing.T) {
	current := map[string]float64{"ESALE": 80, "IMEDIA": 20}
	pc := ProductContext{
		Scopes: []ScopeContext{
			{SKUCode: "20000", ProviderCode: "ESALE", Metrics: &tools.GetMetricsOutput{SuccessRate: 75, PendingRate: 0.3, FailRate: 1.8, TotalTransactions: 1000}},
			{
				SKUCode: "20000", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 97, PendingRate: 0.4, FailRate: 1.9, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: false},
			},
		},
	}
	bad := pc.Scopes[0]
	bad.Threshold = &threshold.Result{
		Breached:      true,
		BreachReasons: []string{"Tỷ lệ thành công 75% dưới ngưỡng 80%"},
	}
	out := proposeShift(current, pc, bad, defaultTestThreshold())
	if out["ESALE"] != 0 {
		t.Fatalf("ESALE = %.1f, want 0 (hard cut)", out["ESALE"])
	}
	if out["IMEDIA"] < 99 {
		t.Fatalf("IMEDIA = %.1f, want ~100", out["IMEDIA"])
	}
}

func TestProposeHardCutZerosBadProvider(t *testing.T) {
	current := map[string]float64{"ESALE": 70, "IMEDIA": 20, "SHOPPAY": 10}
	pc := ProductContext{
		Scopes: []ScopeContext{
			{SKUCode: "V100K", ProviderCode: "ESALE", Metrics: &tools.GetMetricsOutput{SuccessRate: 72}},
			{SKUCode: "V100K", ProviderCode: "IMEDIA", Metrics: &tools.GetMetricsOutput{SuccessRate: 98}},
			{SKUCode: "V100K", ProviderCode: "SHOPPAY", Metrics: &tools.GetMetricsOutput{SuccessRate: 96}},
		},
	}
	bad := ScopeContext{
		ProviderCode: "ESALE",
		SKUCode:      "V100K",
		Threshold: &threshold.Result{
			Breached:      true,
			BreachReasons: []string{"Số GD fail 142 vượt ngưỡng 120"},
		},
	}
	out := proposeShift(current, pc, bad, defaultTestThreshold())
	if out["ESALE"] != 0 {
		t.Fatalf("ESALE = %.1f, want 0", out["ESALE"])
	}
	var sum float64
	for _, v := range out {
		sum += v
	}
	if sum < 99.9 || sum > 100.1 {
		t.Fatalf("sum = %.1f, want 100", sum)
	}
}

func TestRoundPctToInt100Sum(t *testing.T) {
	cases := []map[string]float64{
		{"ESALE": 9.5, "IMEDIA": 28.6, "SHOPPAY": 61.9},
		{"ESALE": 33.33, "IMEDIA": 33.33, "SHOPPAY": 33.34},
		{"ESALE": 0, "IMEDIA": 45.5, "SHOPPAY": 54.5},
	}
	for _, in := range cases {
		out := roundPctToInt100(in)
		var sum float64
		for _, v := range out {
			if v != math.Floor(v) {
				t.Fatalf("non-integer %v in %v", v, out)
			}
			sum += v
		}
		if sum != 100 {
			t.Fatalf("in %v -> %v sum = %.0f, want 100", in, out, sum)
		}
	}
}

func TestBuildRoutingPlanIntSum100(t *testing.T) {
	current := map[string]float64{"ESALE": 35, "IMEDIA": 15, "SHOPPAY": 50}
	pc := ProductContext{
		Routing: tools.GetRoutingOutput{
			Scope: "sku",
			RoutingBySKU: map[string]map[string]float64{
				"V100K": current,
			},
		},
		Scopes: []ScopeContext{
			{SKUCode: "V100K", ProviderCode: "ESALE", Metrics: &tools.GetMetricsOutput{SuccessRate: 72}},
			{SKUCode: "V100K", ProviderCode: "IMEDIA", Metrics: &tools.GetMetricsOutput{SuccessRate: 98}},
			{SKUCode: "V100K", ProviderCode: "SHOPPAY", Metrics: &tools.GetMetricsOutput{SuccessRate: 96}},
		},
	}
	bad := ScopeContext{
		ProviderCode: "ESALE",
		SKUCode:      "V100K",
		Threshold: &threshold.Result{
			Breached:      true,
			BreachReasons: []string{"Số GD fail 142 vượt ngưỡng 120"},
		},
	}
	plan := BuildRoutingPlan(pc, bad, "test", defaultTestThreshold())
	var sum float64
	for _, v := range plan.Proposed {
		sum += v
	}
	if sum != 100 {
		t.Fatalf("proposed sum = %.0f, want 100: %v", sum, plan.Proposed)
	}
}

func TestSKURoutingDecisionAllBreachedMaintenance(t *testing.T) {
	th := defaultTestThreshold()
	routing := map[string]float64{"ESALE": 60, "IMEDIA": 40}
	pc := ProductContext{
		Scopes: []ScopeContext{
			{
				SKUCode: "10000", ProviderCode: "ESALE",
				Metrics: &tools.GetMetricsOutput{SuccessRate: 95, PendingRate: 6, FailRate: 3, TotalTransactions: 1000},
			},
			{
				SKUCode: "10000", ProviderCode: "IMEDIA",
				Metrics: &tools.GetMetricsOutput{SuccessRate: 95, PendingRate: 6, FailRate: 2, TotalTransactions: 1000},
			},
		},
	}
	action, _ := SKURoutingDecision(pc, "10000", routing, th)
	if action != "maintenance" {
		t.Fatalf("action = %q, want maintenance", action)
	}
}

func TestSKURoutingDecisionSkipsMaintainedProvider(t *testing.T) {
	th := store.ProductThreshold{
		Enabled:            true,
		SuccessRateMinPct:  82,
		PendingRateMaxPct:  14,
		FailRateMaxPct:     10,
		FailTxnCountMax:    60,
		PendingTxnCountMax: 8,
	}
	routing := map[string]float64{"ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20}
	pc := ProductContext{
		MaintainedProviders: map[string]bool{"ESALE": true},
		Scopes: []ScopeContext{
			{SKUCode: "V100K", ProviderCode: "ESALE", Metrics: &tools.GetMetricsOutput{SuccessRate: 97.6, PendingRate: 0, FailRate: 2.4, TotalTransactions: 1000}},
			{SKUCode: "V100K", ProviderCode: "IMEDIA", Metrics: &tools.GetMetricsOutput{SuccessRate: 96.7, PendingRate: 0.7, FailRate: 2.6, TotalTransactions: 1000}},
			{SKUCode: "V100K", ProviderCode: "SHOPPAY", Metrics: &tools.GetMetricsOutput{SuccessRate: 97.2, PendingRate: 0.8, FailRate: 2, TotalTransactions: 1000}},
		},
	}
	// Only SHOPPAY breaches pending_txn (8); ESALE in maintenance is ignored.
	pc.Scopes[2].Metrics.TotalTransactions = 1000
	action, _ := SKURoutingDecision(pc, "V100K", routing, th)
	if action != "routing" {
		t.Fatalf("action = %q, want routing", action)
	}
}

func TestProposeShiftSkipsMaintainedTarget(t *testing.T) {
	current := map[string]float64{"ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20}
	th := store.ProductThreshold{
		Enabled:            true,
		SuccessRateMinPct:  82,
		PendingRateMaxPct:  14,
		FailRateMaxPct:     10,
		FailTxnCountMax:    60,
		PendingTxnCountMax: 8,
	}
	pc := ProductContext{
		MaintainedProviders: map[string]bool{"ESALE": true},
		Scopes: []ScopeContext{
			{SKUCode: "V100K", ProviderCode: "ESALE", Metrics: &tools.GetMetricsOutput{SuccessRate: 97}},
			{SKUCode: "V100K", ProviderCode: "IMEDIA", Metrics: &tools.GetMetricsOutput{SuccessRate: 96}},
			{SKUCode: "V100K", ProviderCode: "SHOPPAY", Metrics: &tools.GetMetricsOutput{SuccessRate: 97, TotalTransactions: 1000, PendingRate: 0.8}},
		},
	}
	bad := ScopeContext{
		SKUCode: "V100K", ProviderCode: "SHOPPAY",
		Threshold: &threshold.Result{Breached: true},
	}
	out := proposeShift(current, pc, bad, th)
	if out["SHOPPAY"] != 0 {
		t.Fatalf("SHOPPAY = %.1f, want 0", out["SHOPPAY"])
	}
	if out["ESALE"] != 50 {
		t.Fatalf("ESALE = %.1f, want 50 (in maintenance, no extra traffic)", out["ESALE"])
	}
	if out["IMEDIA"] < 49 {
		t.Fatalf("IMEDIA = %.1f, want ~50 from SHOPPAY shift", out["IMEDIA"])
	}
}

func TestSKURoutingDecisionSingleHealthyBackup(t *testing.T) {
	th := defaultTestThreshold()
	routing := map[string]float64{"ESALE": 80, "IMEDIA": 20}
	pc := ProductContext{
		Scopes: []ScopeContext{
			{
				SKUCode: "20000", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 96, PendingRate: 1.2, FailRate: 2.5, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true},
			},
			{
				SKUCode: "20000", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 97, PendingRate: 0, FailRate: 2.6, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: false},
			},
		},
	}
	action, _ := SKURoutingDecision(pc, "20000", routing, th)
	if action != "routing" {
		t.Fatalf("action = %q, want routing", action)
	}
}

func TestProposeShiftZerosEveryBreachedProvider(t *testing.T) {
	th := defaultTestThreshold()
	current := map[string]float64{"ESALE": 80, "IMEDIA": 20}
	pc := ProductContext{
		Scopes: []ScopeContext{
			{SKUCode: "20000", ProviderCode: "ESALE", Metrics: &tools.GetMetricsOutput{SuccessRate: 98, PendingRate: 0.3, FailRate: 1.8, TotalTransactions: 1000}},
			{
				SKUCode: "20000", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 97, PendingRate: 0.8, FailRate: 1.9, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true, BreachReasons: []string{"Số GD pending 7 vượt ngưỡng 5"}},
			},
		},
	}
	bad := pc.Scopes[1]
	out := proposeShift(current, pc, bad, th)
	if out["IMEDIA"] != 0 {
		t.Fatalf("IMEDIA = %.1f, want 0", out["IMEDIA"])
	}
	if out["ESALE"] < 99 {
		t.Fatalf("ESALE = %.1f, want ~100", out["ESALE"])
	}
}

func TestBuildRoutingPlanSingleHealthyGets100(t *testing.T) {
	th := defaultTestThreshold()
	current := map[string]float64{"ESALE": 70, "IMEDIA": 30}
	pc := ProductContext{
		Routing: tools.GetRoutingOutput{
			Scope: "sku",
			RoutingBySKU: map[string]map[string]float64{
				"VNP20": current,
			},
		},
		Scopes: []ScopeContext{
			{
				SKUCode: "VNP20", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 97, PendingRate: 0.4, FailRate: 2.3, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: false},
			},
			{
				SKUCode: "VNP20", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 96, PendingRate: 0.8, FailRate: 2.7, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true, BreachReasons: []string{"Số GD fail 50 vượt ngưỡng 50"}},
			},
		},
	}
	bad := pc.Scopes[1]
	plan := BuildRoutingPlan(pc, bad, "test", th)
	if plan.Proposed["IMEDIA"] != 0 {
		t.Fatalf("IMEDIA = %.0f, want 0", plan.Proposed["IMEDIA"])
	}
	if plan.Proposed["ESALE"] != 100 {
		t.Fatalf("ESALE = %.0f, want 100 (single healthy)", plan.Proposed["ESALE"])
	}
}

func TestBuildRoutingPlanHardCutZerosAllBreached(t *testing.T) {
	th := defaultTestThreshold()
	current := map[string]float64{"ESALE": 40, "IMEDIA": 35, "SHOPPAY": 25}
	pc := ProductContext{
		Routing: tools.GetRoutingOutput{
			Scope: "sku",
			RoutingBySKU: map[string]map[string]float64{
				"100000": current,
			},
		},
		Scopes: []ScopeContext{
			{
				SKUCode: "100000", ProviderCode: "ESALE",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 97.3, PendingRate: 0, FailRate: 2.7, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: false},
			},
			{
				SKUCode: "100000", ProviderCode: "IMEDIA",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 98, PendingRate: 0.5, FailRate: 1.5, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true, BreachReasons: []string{"Số GD pending 5 vượt ngưỡng 5"}},
			},
			{
				SKUCode: "100000", ProviderCode: "SHOPPAY",
				Metrics:   &tools.GetMetricsOutput{SuccessRate: 96.8, PendingRate: 0.6, FailRate: 2.6, TotalTransactions: 1000},
				Threshold: &threshold.Result{Breached: true, BreachReasons: []string{"Số GD pending 5 vượt ngưỡng 5"}},
			},
		},
	}
	bad := pc.Scopes[1]
	plan := BuildRoutingPlan(pc, bad, "test", th)
	if plan.Proposed["IMEDIA"] != 0 {
		t.Fatalf("IMEDIA = %.0f, want 0", plan.Proposed["IMEDIA"])
	}
	if plan.Proposed["SHOPPAY"] != 0 {
		t.Fatalf("SHOPPAY = %.0f, want 0", plan.Proposed["SHOPPAY"])
	}
	if plan.Proposed["ESALE"] != 100 {
		t.Fatalf("ESALE = %.0f, want 100", plan.Proposed["ESALE"])
	}
}
