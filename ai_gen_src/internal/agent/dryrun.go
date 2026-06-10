package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"opsone/internal/store"
)

// DryRunner delegates to Agent Core Runner (Phase 3); kept for backward-compatible tests.
type DryRunner struct {
	inner *Runner
}

// NewDryRunner creates a dry-run compatible runner.
func NewDryRunner(db *store.DB) *DryRunner {
	return &DryRunner{inner: NewRunner(db)}
}

// RunOnce executes one agent core cycle.
func (r *DryRunner) RunOnce(ctx context.Context) (uint64, int, error) {
	ctxResult, err := r.inner.RunOnce(ctx)
	if err != nil {
		return 0, 0, err
	}
	if ctxResult.CycleID == 0 {
		return 0, 0, nil
	}
	rows := 0
	for _, p := range ctxResult.Products {
		for _, s := range p.Scopes {
			if s.Metrics != nil && !s.Skipped {
				rows++
			}
		}
	}
	return ctxResult.CycleID, rows, nil
}

// RunBlocking runs scheduler until ctx cancelled; skips overlapping runs.
func (r *DryRunner) RunBlocking(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	running := make(chan struct{}, 1)

	run := func() {
		select {
		case running <- struct{}{}:
		default:
			log.Println("agent: previous cycle still running, skip tick")
			return
		}
		defer func() { <-running }()

		if _, _, err := r.RunOnce(ctx); err != nil {
			log.Printf("agent: dry-run error: %v", err)
		}
	}

	run()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			run()
		}
	}
}

// IntervalFromSettings returns scheduler duration from agent_settings.
func IntervalFromSettings(settings store.AgentSettings) time.Duration {
	m := settings.SchedulerIntervalMin
	if m <= 0 {
		m = 5
	}
	return time.Duration(m) * time.Minute
}

// MockIntervalFromSettings returns mock tick duration.
func MockIntervalFromSettings(settings store.AgentSettings) time.Duration {
	m := settings.MockIntervalMin
	if m <= 0 {
		m = 1
	}
	return time.Duration(m) * time.Minute
}

// ValidateDryRun checks cycle wrote history (for tests).
func ValidateDryRun(ctx context.Context, db *store.DB, cycleID uint64) error {
	n, err := db.CountAnalysisHistoryByCycle(ctx, cycleID)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("dry-run cycle %d has 0 history rows", cycleID)
	}
	return nil
}
