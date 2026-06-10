package agent

import (
	"testing"

	"opsone/internal/domain"
	"opsone/internal/store"
	"opsone/internal/tools"
)

func TestBuildReopenRoutingPlanUsesBaseline(t *testing.T) {
	baseline := map[string]float64{"ESALE": 50, "IMEDIA": 30, "SHOPPAY": 20}
	pc := ProductContext{
		Product: domain.Product{ProductCode: "DATA_VINA", RoutingMode: domain.RoutingSKU},
		Routing: tools.GetRoutingOutput{
			Scope: "sku",
			RoutingBySKU: map[string]map[string]float64{
				"V100K": {"ESALE": 100, "IMEDIA": 0, "SHOPPAY": 0},
			},
		},
		Scopes: []ScopeContext{
			{
				SKUCode: "V100K", ProviderCode: "ESALE",
				Metrics: &tools.GetMetricsOutput{SuccessRate: 97, PendingRate: 0, FailRate: 3, TotalTransactions: 1000},
			},
			{
				SKUCode: "V100K", ProviderCode: "IMEDIA",
				Metrics: &tools.GetMetricsOutput{SuccessRate: 98, PendingRate: 0, FailRate: 2, TotalTransactions: 800},
			},
			{
				SKUCode: "V100K", ProviderCode: "SHOPPAY",
				Metrics: &tools.GetMetricsOutput{SuccessRate: 97, PendingRate: 0, FailRate: 2, TotalTransactions: 500},
			},
		},
	}
	th := store.ProductThreshold{Enabled: true}
	plan := BuildReopenRoutingPlan(pc, baseline, th, "")
	if plan.Proposed["ESALE"] != 50 || plan.Proposed["IMEDIA"] != 30 || plan.Proposed["SHOPPAY"] != 20 {
		t.Fatalf("proposed = %v, want baseline 50/30/20", plan.Proposed)
	}
	if plan.Current["ESALE"] != 100 {
		t.Fatalf("current ESALE = %v, want 100", plan.Current["ESALE"])
	}
}
