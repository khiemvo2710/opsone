package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MockMetricRow is one insert into mock_metrics.
type MockMetricRow struct {
	RecordedAt        time.Time
	ProductCode       string
	SKUCode           string
	ProviderCode      string
	SuccessRate       float64
	PendingRate       float64
	FailRate          float64
	TotalTransactions uint
	RevenueLastHour   uint64
	GeneratorRunID    uint64
}

// MockErrorRow is one insert into mock_error_stats.
type MockErrorRow struct {
	RecordedAt     time.Time
	ProductCode    string
	ProviderCode   string
	SKUCode        string
	ErrorCode      string
	ErrorCount     uint
	GeneratorRunID uint64
}

// StartMockGeneratorRun inserts a running mock_generator_run row.
func (db *DB) StartMockGeneratorRun(ctx context.Context, startedAt time.Time, scenario string) (uint64, error) {
	const query = `
		INSERT INTO mock_generator_run (started_at, scenario, status)
		VALUES (?, ?, 'running')`
	res, err := db.ExecContext(ctx, query, startedAt, scenario)
	if err != nil {
		return 0, fmt.Errorf("start mock run: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("mock run id: %w", err)
	}
	return uint64(id), nil
}

// FinishMockGeneratorRun marks run success/failed.
func (db *DB) FinishMockGeneratorRun(ctx context.Context, runID uint64, finishedAt time.Time, rowsMetrics int, errMsg string) error {
	status := "success"
	var msg *string
	if errMsg != "" {
		status = "failed"
		msg = &errMsg
	}
	const query = `
		UPDATE mock_generator_run
	 SET finished_at = ?, rows_metrics = ?, status = ?, error_message = ?
		WHERE id = ?`
	_, err := db.ExecContext(ctx, query, finishedAt, rowsMetrics, status, msg, runID)
	if err != nil {
		return fmt.Errorf("finish mock run: %w", err)
	}
	return nil
}

// InsertMockMetrics batch-inserts metric rows.
func (db *DB) InsertMockMetrics(ctx context.Context, rows []MockMetricRow) error {
	if len(rows) == 0 {
		return nil
	}
	const query = `
		INSERT INTO mock_metrics (
			recorded_at, product_code, sku_code, provider_code,
			success_rate, pending_rate, fail_rate,
			total_transactions, revenue_last_hour, generator_run_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, r := range rows {
		_, err := db.ExecContext(ctx, query,
			r.RecordedAt, r.ProductCode, r.SKUCode, r.ProviderCode,
			r.SuccessRate, r.PendingRate, r.FailRate,
			r.TotalTransactions, r.RevenueLastHour, r.GeneratorRunID,
		)
		if err != nil {
			return fmt.Errorf("insert mock metric: %w", err)
		}
	}
	return nil
}

// InsertMockErrors batch-inserts error stat rows.
func (db *DB) InsertMockErrors(ctx context.Context, rows []MockErrorRow) error {
	if len(rows) == 0 {
		return nil
	}
	const query = `
		INSERT INTO mock_error_stats (
			recorded_at, product_code, provider_code, sku_code,
			error_code, error_count, generator_run_id
		) VALUES (?, ?, ?, ?, ?, ?, ?)`
	for _, r := range rows {
		_, err := db.ExecContext(ctx, query,
			r.RecordedAt, r.ProductCode, r.ProviderCode, r.SKUCode,
			r.ErrorCode, r.ErrorCount, r.GeneratorRunID,
		)
		if err != nil {
			return fmt.Errorf("insert mock error: %w", err)
		}
	}
	return nil
}

// CountMockMetrics returns total rows in mock_metrics.
func (db *DB) CountMockMetrics(ctx context.Context) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mock_metrics`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count mock metrics: %w", err)
	}
	return n, nil
}

// PurgeMockMetricsOlderThan deletes mock data past retention (§13.11).
func (db *DB) PurgeMockMetricsOlderThan(ctx context.Context, cutoff time.Time) error {
	const qMetrics = `DELETE FROM mock_metrics WHERE recorded_at < ?`
	const qErrors = `DELETE FROM mock_error_stats WHERE recorded_at < ?`
	if _, err := db.ExecContext(ctx, qMetrics, cutoff); err != nil {
		return fmt.Errorf("purge mock_metrics: %w", err)
	}
	if _, err := db.ExecContext(ctx, qErrors, cutoff); err != nil {
		return fmt.Errorf("purge mock_error_stats: %w", err)
	}
	return nil
}

// LatestMockMetric holds the most recent mock metric for a scope.
type LatestMockMetric struct {
	SuccessRate float64
	PendingRate float64
	FailRate    float64
	RecordedAt  time.Time
}

// GetLatestMockMetric returns newest mock_metrics row for scope.
func (db *DB) GetLatestMockMetric(ctx context.Context, productCode, skuCode, providerCode string) (LatestMockMetric, bool, error) {
	const query = `
		SELECT success_rate, pending_rate, fail_rate, recorded_at
		FROM mock_metrics
		WHERE product_code = ? AND sku_code = ? AND provider_code = ?
		ORDER BY recorded_at DESC
		LIMIT 1`
	var m LatestMockMetric
	err := db.QueryRowContext(ctx, query, productCode, skuCode, providerCode).Scan(
		&m.SuccessRate, &m.PendingRate, &m.FailRate, &m.RecordedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LatestMockMetric{}, false, nil
		}
		return LatestMockMetric{}, false, fmt.Errorf("latest mock metric: %w", err)
	}
	return m, true, nil
}
