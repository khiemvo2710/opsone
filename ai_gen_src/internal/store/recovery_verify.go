package store

import (
	"context"
	"database/sql"
	"time"
)

// RoutingVerify tracks a routing action for auto-recovery verification.
type RoutingVerify struct {
	ID             uint64
	AgentChangeID  uint64
	ProductCode    string
	SKUCode        string
	PreSuccessRate float64
	PrePendingRate float64
	TakenAt        time.Time
	VerifyAfter    time.Time
}

// InsertRoutingVerify records a routing action for later verification.
func (db *DB) InsertRoutingVerify(ctx context.Context,
	changeID uint64, product, sku string,
	preSuccess, prePending float64,
	verifyAfter time.Time) error {

	const q = `INSERT INTO routing_action_verify
               (agent_change_id, product_code, sku_code, pre_success_rate, pre_pending_rate, verify_after)
               VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, q, changeID, product, sku, preSuccess, prePending, verifyAfter)
	return err
}

// ListPendingVerify returns records where verify_after <= now and status = 'pending'.
func (db *DB) ListPendingVerify(ctx context.Context) ([]RoutingVerify, error) {
	const q = `SELECT id, agent_change_id, product_code, sku_code,
               COALESCE(pre_success_rate,0), COALESCE(pre_pending_rate,0), taken_at, verify_after
               FROM routing_action_verify
               WHERE recovery_status = 'pending' AND verify_after <= NOW()
               ORDER BY verify_after ASC LIMIT 50`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoutingVerify
	for rows.Next() {
		var v RoutingVerify
		if err := rows.Scan(&v.ID, &v.AgentChangeID, &v.ProductCode, &v.SKUCode,
			&v.PreSuccessRate, &v.PrePendingRate, &v.TakenAt, &v.VerifyAfter); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// UpdateVerifyResult writes post-action metrics and sets recovery status.
func (db *DB) UpdateVerifyResult(ctx context.Context, id uint64, postSuccess, postPending float64, status string, escalated bool) error {
	esc := 0
	if escalated {
		esc = 1
	}
	const q = `UPDATE routing_action_verify
               SET verified_at=NOW(), post_success_rate=?, post_pending_rate=?,
                   recovery_status=?, escalated=?
               WHERE id=?`
	_, err := db.ExecContext(ctx, q, postSuccess, postPending, status, esc, id)
	return err
}

// MarkVerifyEscalated marks a verify record as escalated.
func (db *DB) MarkVerifyEscalated(ctx context.Context, id uint64) error {
	const q = `UPDATE routing_action_verify SET escalated=1 WHERE id=?`
	_, err := db.ExecContext(ctx, q, id)
	return err
}

// FindCurrentMetricsForScope queries the latest analysis_history for a product/sku.
func (db *DB) FindCurrentMetricsForScope(ctx context.Context, product, sku string) (successRate, pendingRate float64, found bool) {
	const q = `SELECT AVG(success_rate), AVG(pending_rate)
               FROM agent_analysis_history
               WHERE product_code=? AND sku_code=?
               AND recorded_at >= DATE_SUB(NOW(), INTERVAL 10 MINUTE)
               GROUP BY product_code, sku_code
               LIMIT 1`
	err := db.QueryRowContext(ctx, q, product, sku).Scan(&successRate, &pendingRate)
	if err == sql.ErrNoRows || err != nil {
		return 0, 0, false
	}
	return successRate, pendingRate, true
}
