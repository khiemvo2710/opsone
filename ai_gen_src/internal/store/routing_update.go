package store

import (
	"context"
	"encoding/json"
	"fmt"

	"opsone/internal/domain"
)

// RoutingSnapshot JSON for agent_change_log (§8.7).
type RoutingSnapshot struct {
	ProductCode string             `json:"product_code"`
	Scope       string             `json:"scope"`
	SKUCode     string             `json:"sku_code"`
	Providers   map[string]float64 `json:"providers"`
}

// UpdateTrafficPct updates traffic_pct for one provider in scope.
func (db *DB) UpdateTrafficPct(ctx context.Context, product, sku, provider string, pct float64, updatedBy string) error {
	const query = `
		UPDATE routing_config
		SET traffic_pct = ?, updated_by = ?, updated_at = NOW()
		WHERE product_code = ? AND sku_code = ? AND provider_code = ?`
	_, err := db.ExecContext(ctx, query, pct, updatedBy, product, sku, provider)
	return err
}

// UpdateBaselineAndTraffic sets both baseline and traffic (set_as_baseline=true).
func (db *DB) UpdateBaselineAndTraffic(ctx context.Context, product, sku, provider string, pct float64, updatedBy string) error {
	const query = `
		UPDATE routing_config
		SET baseline_pct = ?, traffic_pct = ?, baseline_updated_by = ?, baseline_updated_at = NOW(),
		    updated_by = ?, updated_at = NOW()
		WHERE product_code = ? AND sku_code = ? AND provider_code = ?`
	_, err := db.ExecContext(ctx, query, pct, pct, updatedBy, updatedBy, product, sku, provider)
	return err
}

// SetPendingRestore updates routing_scope_state (§8.6.5).
func (db *DB) SetPendingRestore(ctx context.Context, product, sku string, pending bool, by string) error {
	val := 0
	if pending {
		val = 1
	}
	const query = `
		INSERT INTO routing_scope_state (product_code, sku_code, pending_restore, manual_override_by, manual_override_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			pending_restore = VALUES(pending_restore),
			manual_override_by = VALUES(manual_override_by),
			manual_override_at = NOW()`
	var byPtr *string
	if by != "" {
		byPtr = &by
	}
	_, err := db.ExecContext(ctx, query, product, sku, val, byPtr)
	return err
}

// BuildRoutingSnapshot reads current traffic_pct map for scope.
func (db *DB) BuildRoutingSnapshot(ctx context.Context, product, scope, sku string) (RoutingSnapshot, error) {
	rows, err := db.GetRoutingForScope(ctx, product, sku)
	if err != nil {
		return RoutingSnapshot{}, err
	}
	m := make(map[string]float64, len(rows))
	for _, r := range rows {
		m[r.ProviderCode] = r.TrafficPct
	}
	return RoutingSnapshot{
		ProductCode: product,
		Scope:       scope,
		SKUCode:     sku,
		Providers:   m,
	}, nil
}

// AgentChangeInsert params for agent_change_log.
type AgentChangeInsert struct {
	ProductCode   string
	Scope         string
	SKUCode       string
	RoutingBefore RoutingSnapshot
	RoutingAfter  RoutingSnapshot
	TriggerType   string
	ExecutedBy    string
	Reason        string
	CycleID       *uint64
	RoutingPlanID *uint64
	IncidentID    *string
}

// InsertAgentChangeLog writes a routing change record.
func (db *DB) InsertAgentChangeLog(ctx context.Context, p AgentChangeInsert) (uint64, error) {
	before, err := json.Marshal(p.RoutingBefore)
	if err != nil {
		return 0, err
	}
	after, err := json.Marshal(p.RoutingAfter)
	if err != nil {
		return 0, err
	}
	const query = `
		INSERT INTO agent_change_log (
			change_type, product_code, scope, sku_code,
			routing_before, routing_after, trigger_type, change_status,
			cycle_id, routing_plan_id, incident_id, reason, executed_by
		) VALUES ('routing', ?, ?, ?, ?, ?, ?, 'applied', ?, ?, ?, ?, ?)`
	res, err := db.ExecContext(ctx, query,
		p.ProductCode, p.Scope, p.SKUCode,
		before, after, p.TriggerType,
		p.CycleID, p.RoutingPlanID, p.IncidentID, p.Reason, p.ExecutedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("insert agent_change_log: %w", err)
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

// ApplyRoutingSnapshot restores routing_config traffic from snapshot map.
func (db *DB) ApplyRoutingSnapshot(ctx context.Context, snap RoutingSnapshot, updatedBy string, updateBaseline bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for provider, pct := range snap.Providers {
		if updateBaseline {
			_, err = tx.ExecContext(ctx, `
				UPDATE routing_config SET baseline_pct=?, traffic_pct=?, baseline_updated_by=?, baseline_updated_at=NOW(), updated_by=?, updated_at=NOW()
				WHERE product_code=? AND sku_code=? AND provider_code=?`,
				pct, pct, updatedBy, updatedBy, snap.ProductCode, snap.SKUCode, provider)
		} else {
			_, err = tx.ExecContext(ctx, `
				UPDATE routing_config SET traffic_pct=?, updated_by=?, updated_at=NOW()
			 WHERE product_code=? AND sku_code=? AND provider_code=?`,
				pct, updatedBy, snap.ProductCode, snap.SKUCode, provider)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetAllRoutingForProduct returns all routing rows for product (all SKUs).
func (db *DB) GetAllRoutingForProduct(ctx context.Context, productCode string) ([]domain.RoutingPct, error) {
	const query = `
		SELECT product_code, sku_code, provider_code, baseline_pct, traffic_pct
		FROM routing_config
		WHERE product_code = ?
		ORDER BY sku_code, provider_code`
	rows, err := db.QueryContext(ctx, query, productCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RoutingPct
	for rows.Next() {
		var r domain.RoutingPct
		if err := rows.Scan(&r.ProductCode, &r.SKUCode, &r.ProviderCode, &r.BaselinePct, &r.TrafficPct); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
