package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MetricWindowResult is aggregated metric for a scope in a time window.
type MetricWindowResult struct {
	SuccessRate       float64
	PendingRate       float64
	FailRate          float64
	TotalTransactions uint
	RevenueLastHour   uint64
	RecordedAt        time.Time
}

func metricsScopeKey(product, sku, provider string) string {
	return product + "\x00" + sku + "\x00" + provider
}

// LoadLatestMetricsSince returns the newest metric row per (product, sku, provider) since cutoff — one query for dashboard overview.
func (db *DB) LoadLatestMetricsSince(ctx context.Context, dataSource string, since time.Time) (map[string]MetricWindowResult, error) {
	table := "mock_metrics"
	if dataSource == "production" {
		table = "metrics_snapshot"
	}
	query := fmt.Sprintf(`
		SELECT m.product_code, m.sku_code, m.provider_code,
		       m.success_rate, m.pending_rate, m.fail_rate, m.total_transactions, m.revenue_last_hour, m.recorded_at
		FROM %s m
		INNER JOIN (
			SELECT product_code, sku_code, provider_code, MAX(recorded_at) AS recorded_at
			FROM %s
			WHERE recorded_at >= ?
			GROUP BY product_code, sku_code, provider_code
		) t ON m.product_code = t.product_code AND m.sku_code = t.sku_code
		   AND m.provider_code = t.provider_code AND m.recorded_at = t.recorded_at
		WHERE m.recorded_at >= ?`, table, table)

	rows, err := db.QueryContext(ctx, query, since, since)
	if err != nil {
		return nil, fmt.Errorf("load latest metrics: %w", err)
	}
	defer rows.Close()

	out := make(map[string]MetricWindowResult)
	for rows.Next() {
		var product, sku, provider string
		var m MetricWindowResult
		if err := rows.Scan(
			&product, &sku, &provider,
			&m.SuccessRate, &m.PendingRate, &m.FailRate,
			&m.TotalTransactions, &m.RevenueLastHour, &m.RecordedAt,
		); err != nil {
			return nil, err
		}
		out[metricsScopeKey(product, sku, provider)] = m
	}
	return out, rows.Err()
}

// GetMetricsInWindow reads latest row in window from mock or production table.
func (db *DB) GetMetricsInWindow(ctx context.Context, dataSource, product, sku, provider string, since time.Time) (MetricWindowResult, bool, error) {
	table := "mock_metrics"
	if dataSource == "production" {
		table = "metrics_snapshot"
	}
	// table name is fixed enum — not user input
	query := fmt.Sprintf(`
		SELECT success_rate, pending_rate, fail_rate, total_transactions, revenue_last_hour, recorded_at
		FROM %s
		WHERE product_code = ? AND sku_code = ? AND provider_code = ?
		  AND recorded_at >= ?
		ORDER BY recorded_at DESC
		LIMIT 1`, table)

	var m MetricWindowResult
	err := db.QueryRowContext(ctx, query, product, sku, provider, since).Scan(
		&m.SuccessRate, &m.PendingRate, &m.FailRate,
		&m.TotalTransactions, &m.RevenueLastHour, &m.RecordedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return MetricWindowResult{}, false, nil
	}
	if err != nil {
		return MetricWindowResult{}, false, fmt.Errorf("metrics window: %w", err)
	}
	return m, true, nil
}

// ErrorAgg is one error code aggregate.
type ErrorAgg struct {
	ErrorCode  string
	ErrorCount uint
}

// GetTopErrorsInWindow returns top N error codes in window.
func (db *DB) GetTopErrorsInWindow(ctx context.Context, dataSource, product, sku, provider string, since time.Time, limit int) ([]ErrorAgg, error) {
	if limit <= 0 {
		limit = 5
	}
	table := "mock_error_stats"
	if dataSource == "production" {
		// production errors would use a similar table; reuse mock for hackathon
		table = "mock_error_stats"
	}
	query := fmt.Sprintf(`
		SELECT error_code, SUM(error_count) AS cnt
		FROM %s
		WHERE product_code = ? AND sku_code = ? AND provider_code = ?
		  AND recorded_at >= ?
		GROUP BY error_code
		ORDER BY cnt DESC
		LIMIT ?`, table)

	rows, err := db.QueryContext(ctx, query, product, sku, provider, since, limit)
	if err != nil {
		return nil, fmt.Errorf("top errors: %w", err)
	}
	defer rows.Close()

	var out []ErrorAgg
	for rows.Next() {
		var e ErrorAgg
		if err := rows.Scan(&e.ErrorCode, &e.ErrorCount); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SumErrorEventsAtRecordedAt sums error events at the same snapshot as GetMetricsInWindow (§7.4).
// Metrics use the latest row in window; error counts must align to that row — not a multi-minute SUM.
func (db *DB) SumErrorEventsAtRecordedAt(ctx context.Context, dataSource, product, sku, provider string, at time.Time) (uint, error) {
	table := "mock_error_stats"
	if dataSource == "production" {
		table = "mock_error_stats"
	}
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(error_count), 0)
		FROM %s
		WHERE product_code = ? AND sku_code = ? AND provider_code = ?
		  AND recorded_at = ?`, table)

	var sum uint
	err := db.QueryRowContext(ctx, query, product, sku, provider, at).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum errors at snapshot: %w", err)
	}
	return sum, nil
}

// SumErrorEventsInWindow sums all error events for threshold (§7.4).
func (db *DB) SumErrorEventsInWindow(ctx context.Context, dataSource, product, sku, provider string, since time.Time) (uint, error) {
	table := "mock_error_stats"
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(error_count), 0)
		FROM %s
		WHERE product_code = ? AND sku_code = ? AND provider_code = ?
		  AND recorded_at >= ?`, table)

	var sum uint
	err := db.QueryRowContext(ctx, query, product, sku, provider, since).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum errors: %w", err)
	}
	return sum, nil
}
