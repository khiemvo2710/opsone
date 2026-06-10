package api

import (
	"encoding/json"
	"testing"

	"opsone/internal/agent"
	"opsone/internal/store"
)

func TestRoutingProposalSuppressedAfterReject(t *testing.T) {
	rejectedPlan := agent.RoutingPlanJSON{
		Proposed: map[string]float64{"ESALE": 100, "IMEDIA": 0},
	}
	raw, err := json.Marshal(rejectedPlan)
	if err != nil {
		t.Fatal(err)
	}
	latest := store.RoutingPlanRow{Status: "rejected", PlanJSON: raw}

	same := agent.RoutingPlanJSON{
		Proposed: map[string]float64{"ESALE": 100, "IMEDIA": 0},
	}
	if !routingProposalSuppressedByReject(latest, same) {
		t.Fatal("expected identical proposal to be suppressed after reject")
	}

	different := agent.RoutingPlanJSON{
		Proposed: map[string]float64{"ESALE": 0, "IMEDIA": 100},
	}
	if routingProposalSuppressedByReject(latest, different) {
		t.Fatal("expected different proposal to show after reject")
	}

	executed := store.RoutingPlanRow{Status: "executed", PlanJSON: raw}
	if routingProposalSuppressedByReject(executed, same) {
		t.Fatal("executed plan must not suppress new proposals")
	}
}
