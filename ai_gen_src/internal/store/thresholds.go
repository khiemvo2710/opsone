package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ProductThreshold holds effective thresholds for a product (§13.2.1).
type ProductThreshold struct {
	ProductCode               string
	Enabled                   bool
	SuccessRateMinPct         float64
	PendingRateMaxPct         float64
	FailRateMaxPct            float64
	FailTxnCountMax           uint
	PendingTxnCountMax        uint
	ErrorEventCountMax        uint
	MetricsWindowMin          int
	ConsecutiveCyclesRequired int
	AlertEmailEnabled         bool
}

// ListAllProductThresholds loads every product threshold in one query (dashboard overview).
func (db *DB) ListAllProductThresholds(ctx context.Context) (map[string]ProductThreshold, error) {
	const query = `
		SELECT
			pat.product_code,
			pat.enabled,
			COALESCE(pat.success_rate_min_pct, s.default_success_rate_min_pct),
			COALESCE(pat.pending_rate_max_pct, s.default_pending_rate_max_pct),
			COALESCE(pat.fail_rate_max_pct, s.default_fail_rate_max_pct),
			COALESCE(pat.fail_txn_count_max, s.default_fail_txn_count_max, 50),
			COALESCE(pat.pending_txn_count_max, s.default_pending_txn_count_max, 5),
			COALESCE(pat.error_event_count_max, s.default_error_event_count_max),
			pat.metrics_window_min,
			pat.consecutive_cycles_required,
			pat.alert_email_enabled
		FROM product_alert_thresholds pat
		CROSS JOIN agent_settings s
		WHERE s.id = 1`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list thresholds: %w", err)
	}
	defer rows.Close()

	out := make(map[string]ProductThreshold)
	for rows.Next() {
		var t ProductThreshold
		var enabled, alert int
		if err := rows.Scan(
			&t.ProductCode, &enabled,
			&t.SuccessRateMinPct, &t.PendingRateMaxPct, &t.FailRateMaxPct,
			&t.FailTxnCountMax, &t.PendingTxnCountMax, &t.ErrorEventCountMax,
			&t.MetricsWindowMin, &t.ConsecutiveCyclesRequired,
			&alert,
		); err != nil {
			return nil, err
		}
		t.Enabled = enabled == 1
		t.AlertEmailEnabled = alert == 1
		out[t.ProductCode] = t
	}
	return out, rows.Err()
}

// GetProductThreshold loads product thresholds with fallback to agent_settings defaults.
func (db *DB) GetProductThreshold(ctx context.Context, productCode string) (ProductThreshold, error) {
	const query = `
		SELECT
			pat.product_code,
			pat.enabled,
			COALESCE(pat.success_rate_min_pct, s.default_success_rate_min_pct),
			COALESCE(pat.pending_rate_max_pct, s.default_pending_rate_max_pct),
			COALESCE(pat.fail_rate_max_pct, s.default_fail_rate_max_pct),
			COALESCE(pat.fail_txn_count_max, s.default_fail_txn_count_max, 50),
			COALESCE(pat.pending_txn_count_max, s.default_pending_txn_count_max, 5),
			COALESCE(pat.error_event_count_max, s.default_error_event_count_max),
			pat.metrics_window_min,
			pat.consecutive_cycles_required,
			pat.alert_email_enabled
		FROM product_alert_thresholds pat
		CROSS JOIN agent_settings s
		WHERE pat.product_code = ? AND s.id = 1`

	var t ProductThreshold
	var enabled, alert int
	err := db.QueryRowContext(ctx, query, productCode).Scan(
		&t.ProductCode, &enabled,
		&t.SuccessRateMinPct, &t.PendingRateMaxPct, &t.FailRateMaxPct,
		&t.FailTxnCountMax, &t.PendingTxnCountMax, &t.ErrorEventCountMax,
		&t.MetricsWindowMin, &t.ConsecutiveCyclesRequired,
		&alert,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ProductThreshold{}, fmt.Errorf("thresholds for %q not found", productCode)
	}
	if err != nil {
		return ProductThreshold{}, fmt.Errorf("get threshold: %w", err)
	}
	t.Enabled = enabled == 1
	t.AlertEmailEnabled = alert == 1
	return t, nil
}

// UpdateProductThreshold persists admin-edited thresholds (§9.5.4).
func (db *DB) UpdateProductThreshold(ctx context.Context, t ProductThreshold, updatedBy string) error {
	const query = `
		UPDATE product_alert_thresholds SET
			success_rate_min_pct = ?,
			pending_rate_max_pct = ?,
			fail_rate_max_pct = ?,
			fail_txn_count_max = ?,
			pending_txn_count_max = ?,
			error_event_count_max = ?,
			alert_email_enabled = ?,
			updated_by = ?
		WHERE product_code = ?`
	alert := 0
	if t.AlertEmailEnabled {
		alert = 1
	}
	res, err := db.ExecContext(ctx, query,
		t.SuccessRateMinPct, t.PendingRateMaxPct, t.FailRateMaxPct,
		t.FailTxnCountMax, t.PendingTxnCountMax, t.ErrorEventCountMax,
		alert, updatedBy, t.ProductCode,
	)
	if err != nil {
		return fmt.Errorf("update threshold: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("thresholds for %q not found", t.ProductCode)
	}
	return nil
}

// GetRoutingGoodThresholdPct from agent_settings.
func (db *DB) GetRoutingGoodThresholdPct(ctx context.Context) (float64, error) {
	var v float64
	err := db.QueryRowContext(ctx, `SELECT routing_good_threshold_pct FROM agent_settings WHERE id = 1`).Scan(&v)
	return v, err
}

