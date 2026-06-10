package rules_test

import (
	"testing"

	"opsone/internal/rules"
	"opsone/internal/store"
)

func TestR3FailSpike(t *testing.T) {
	engine := rules.Engine{}
	in := rules.ScopeInput{
		Scope: rules.ScopeData{
			ServiceType: "topup",
			FailRate:    13, SuccessRate: 75, Breached: true,
		},
		History:    []store.ScopeHistoryPoint{{FailRate: 7}},
		FailMaxPct: 10,
	}
	results := engine.EvaluateScope(in)
	found := false
	for _, r := range results {
		if r.RuleID == "R3_FAIL_SPIKE" && r.Triggered {
			found = true
		}
	}
	if !found {
		t.Error("expected R3_FAIL_SPIKE triggered")
	}
}

func TestR1NoTriggerWithShortHistory(t *testing.T) {
	engine := rules.Engine{}
	in := rules.ScopeInput{
		Scope: rules.ScopeData{ServiceType: "topup", SuccessRate: 70, Breached: true},
		History: []store.ScopeHistoryPoint{{SuccessRate: 72}},
	}
	results := engine.EvaluateScope(in)
	for _, r := range results {
		if r.RuleID == "R1_SUCCESS_DECLINE" && r.Triggered {
			t.Error("R1 should not trigger with < 2 history points")
		}
	}
}
