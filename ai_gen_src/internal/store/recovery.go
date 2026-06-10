package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// MarkRecoveryStart records the analysis cycle when routing was applied (§3.1, §8.6.3).
func (db *DB) MarkRecoveryStart(ctx context.Context, product, sku string, applyCycleID uint64) error {
	const query = `
		INSERT INTO routing_scope_state (product_code, sku_code, recovery_apply_cycle_id)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE recovery_apply_cycle_id = VALUES(recovery_apply_cycle_id)`
	_, err := db.ExecContext(ctx, query, product, sku, applyCycleID)
	return err
}

// ClearRecoveryStart removes recovery tracking after green recovery.
func (db *DB) ClearRecoveryStart(ctx context.Context, product, sku string) error {
	const query = `
		UPDATE routing_scope_state SET recovery_apply_cycle_id = NULL
		WHERE product_code = ? AND sku_code = ?`
	_, err := db.ExecContext(ctx, query, product, sku)
	return err
}

// GetRecoveryApplyCycle returns cycle id when routing recovery started for scope.
func (db *DB) GetRecoveryApplyCycle(ctx context.Context, product, sku string) (uint64, bool, error) {
	const query = `
		SELECT recovery_apply_cycle_id FROM routing_scope_state
		WHERE product_code = ? AND sku_code = ? AND recovery_apply_cycle_id IS NOT NULL`
	var id uint64
	err := db.QueryRowContext(ctx, query, product, sku).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// LatestCompletedCycleID is the most recent successful analysis cycle.
func (db *DB) LatestCompletedCycleID(ctx context.Context) (uint64, error) {
	const query = `
		SELECT id FROM agent_analysis_cycles
		WHERE status = 'success'
		ORDER BY id DESC LIMIT 1`
	var id uint64
	err := db.QueryRowContext(ctx, query).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("latest completed cycle: %w", err)
	}
	return id, nil
}

// CountAnalysisCyclesSince counts successful cycles with id in (afterCycleID, currentCycleID].
func (db *DB) CountAnalysisCyclesSince(ctx context.Context, afterCycleID, currentCycleID uint64) (int, error) {
	if currentCycleID <= afterCycleID {
		return 0, nil
	}
	const query = `
		SELECT COUNT(*) FROM agent_analysis_cycles
		WHERE id > ? AND id <= ? AND status = 'success'`
	var n int
	err := db.QueryRowContext(ctx, query, afterCycleID, currentCycleID).Scan(&n)
	return n, err
}
