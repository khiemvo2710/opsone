package agent

import (
	"context"
	"log"

	"opsone/internal/store"
)

// BaselineJob aggregates metric history into hourly baselines for anomaly detection.
type BaselineJob struct {
	DB *store.DB
}

// RunOnce recomputes baselines from the last 14 days of analysis history.
func (b *BaselineJob) RunOnce(ctx context.Context) {
	if err := b.DB.AggregateBaselines(ctx, 14); err != nil {
		log.Printf("baseline_job: aggregate error: %v", err)
	} else {
		log.Println("baseline_job: hourly baselines updated")
	}
}
