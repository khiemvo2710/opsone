package store

import (
	"context"
	"fmt"
	"time"
)

// AnalysisHistoryRow is one agent_analysis_history insert.
type AnalysisHistoryRow struct {
	RecordedAt   time.Time
	ProductCode  string
	ServiceType  string
	SKUCode      string
	ProviderCode string
	SuccessRate  float64
	PendingRate  float64
	FailRate     float64
}

// StartAnalysisCycle inserts agent_analysis_cycles with status running.
func (db *DB) StartAnalysisCycle(ctx context.Context, startedAt time.Time, dataSource string) (uint64, error) {
	const query = `
		INSERT INTO agent_analysis_cycles (cycle_started, data_source, status)
		VALUES (?, ?, 'running')`
	res, err := db.ExecContext(ctx, query, startedAt, dataSource)
	if err != nil {
		return 0, fmt.Errorf("start analysis cycle: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint64(id), nil
}

// FinishAnalysisCycle updates cycle completion.
func (db *DB) FinishAnalysisCycle(ctx context.Context, cycleID uint64, finishedAt time.Time, healthStatus, decision string) error {
	return db.FinishAnalysisCycleFull(ctx, cycleID, finishedAt, healthStatus, "", decision)
}

// FinishAnalysisCycleFull updates cycle with optional health_summary (§3).
func (db *DB) FinishAnalysisCycleFull(ctx context.Context, cycleID uint64, finishedAt time.Time, healthStatus, healthSummary, decision string) error {
	const query = `
		UPDATE agent_analysis_cycles
		SET cycle_finished = ?, health_status = ?, health_summary = ?, decision = ?, status = 'success'
		WHERE id = ?`
	var summary *string
	if healthSummary != "" {
		summary = &healthSummary
	}
	_, err := db.ExecContext(ctx, query, finishedAt, healthStatus, summary, decision, cycleID)
	if err != nil {
		return fmt.Errorf("finish analysis cycle: %w", err)
	}
	return nil
}

// FailAnalysisCycle marks cycle failed.
func (db *DB) FailAnalysisCycle(ctx context.Context, cycleID uint64, finishedAt time.Time) error {
	const query = `
		UPDATE agent_analysis_cycles
		SET cycle_finished = ?, status = 'failed', health_status = 'red'
		WHERE id = ?`
	_, err := db.ExecContext(ctx, query, finishedAt, cycleID)
	return err
}

// InsertAnalysisHistory batch-inserts history rows for a cycle.
func (db *DB) InsertAnalysisHistory(ctx context.Context, cycleID uint64, rows []AnalysisHistoryRow) error {
	const query = `
		INSERT INTO agent_analysis_history (
			cycle_id, recorded_at, product_code, service_type,
			sku_code, provider_code, success_rate, pending_rate, fail_rate
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, r := range rows {
		_, err := db.ExecContext(ctx, query,
			cycleID, r.RecordedAt, r.ProductCode, r.ServiceType,
			r.SKUCode, r.ProviderCode, r.SuccessRate, r.PendingRate, r.FailRate,
		)
		if err != nil {
			return fmt.Errorf("insert analysis history: %w", err)
		}
	}
	return nil
}

// CountAnalysisHistory returns rows for a cycle.
func (db *DB) CountAnalysisHistoryByCycle(ctx context.Context, cycleID uint64) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM agent_analysis_history WHERE cycle_id = ?`, cycleID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count analysis history: %w", err)
	}
	return n, nil
}

// CountAnalysisCycles returns total cycles.
func (db *DB) CountAnalysisCycles(ctx context.Context) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_analysis_cycles`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
