package e2e_test

import (
	"context"
	"os"
	"testing"

	"opsone/internal/agent"
	"opsone/internal/mock"
)

// §10.7 / §12 — ≥2 chu kỳ agent_analysis_cycles.
// Mặc định: kiểm tra DB đã có ≥1 cycle (sau seed/dev). Chạy đủ 2 cycle: OPSONE_E2E_FULL=1 (~5–10 phút).
func TestE2E_ScenarioG_AgentCycleHistory(t *testing.T) {
	db, _ := requireE2E(t)
	ctx := context.Background()

	if os.Getenv("OPSONE_E2E_FULL") == "1" {
		runner := agent.NewRunner(db)
		gen := mock.NewGenerator(db, 777)
		for i := 0; i < 2; i++ {
			if _, err := gen.RunOnce(ctx); err != nil {
				t.Fatalf("mock cycle %d: %v", i+1, err)
			}
			result, err := runner.RunOnce(ctx)
			if err != nil {
				t.Fatalf("agent cycle %d: %v", i+1, err)
			}
			n, err := db.CountAnalysisHistoryByCycle(ctx, result.CycleID)
			if err != nil {
				t.Fatal(err)
			}
			if n == 0 {
				t.Errorf("cycle %d: no analysis history rows", result.CycleID)
			}
		}
	}

	n, err := db.CountAnalysisCycles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	minWant := 1
	if os.Getenv("OPSONE_E2E_FULL") == "1" {
		minWant = 2
	}
	if n < minWant {
		t.Errorf("agent_analysis_cycles = %d, want >= %d (set OPSONE_E2E_FULL=1 to run 2 fresh cycles)", n, minWant)
	}
}
