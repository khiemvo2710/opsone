package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// AgentChangeRecord is one agent_change_log row.
type AgentChangeRecord struct {
	ID            uint64
	ProductCode   string
	Scope         string
	SKUCode       string
	RoutingBefore RoutingSnapshot
	RoutingAfter  RoutingSnapshot
	TriggerType   string
	ChangeStatus  string
	CycleID       sql.NullInt64
	ExecutedBy    sql.NullString
	Reason        sql.NullString
}

// GetAgentChangeByID loads one change log entry.
func (db *DB) GetAgentChangeByID(ctx context.Context, id uint64) (AgentChangeRecord, error) {
	const query = `
		SELECT id, product_code, scope, sku_code, routing_before, routing_after,
		       trigger_type, change_status, cycle_id, executed_by, reason
		FROM agent_change_log
		WHERE id = ?`
	var rec AgentChangeRecord
	var beforeRaw, afterRaw []byte
	err := db.QueryRowContext(ctx, query, id).Scan(
		&rec.ID, &rec.ProductCode, &rec.Scope, &rec.SKUCode,
		&beforeRaw, &afterRaw, &rec.TriggerType, &rec.ChangeStatus,
		&rec.CycleID, &rec.ExecutedBy, &rec.Reason,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentChangeRecord{}, fmt.Errorf("change %d not found", id)
	}
	if err != nil {
		return AgentChangeRecord{}, err
	}
	if err := json.Unmarshal(beforeRaw, &rec.RoutingBefore); err != nil {
		return AgentChangeRecord{}, err
	}
	if err := json.Unmarshal(afterRaw, &rec.RoutingAfter); err != nil {
		return AgentChangeRecord{}, err
	}
	return rec, nil
}

// IsLatestAppliedChange checks LIFO rule (§8.7).
func (db *DB) IsLatestAppliedChange(ctx context.Context, id uint64, product, scope, sku string) (bool, error) {
	latest, ok, err := db.GetLatestAppliedRoutingChange(ctx, product, scope, sku)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return latest.ID == id, nil
}

// InsertRollbackChange logs rollback action.
func (db *DB) InsertRollbackChange(ctx context.Context, original AgentChangeRecord, restored RoutingSnapshot, executedBy, reason string) (uint64, error) {
	before, err := json.Marshal(original.RoutingAfter)
	if err != nil {
		return 0, err
	}
	after, err := json.Marshal(restored)
	if err != nil {
		return 0, err
	}
	const query = `
		INSERT INTO agent_change_log (
			change_type, product_code, scope, sku_code,
			routing_before, routing_after, trigger_type, change_status,
			rollback_of_id, reason, executed_by
		) VALUES ('routing', ?, ?, ?, ?, ?, 'rollback', 'applied', ?, ?, ?)`
	res, err := db.ExecContext(ctx, query,
		original.ProductCode, original.Scope, original.SKUCode,
		before, after, original.ID, reason, executedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("insert rollback change: %w", err)
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

// RoutingMatchesCurrent checks if snapshot matches routing_config (§8.7 conflict).
func (db *DB) RoutingMatchesCurrent(ctx context.Context, product, scope, sku string, expected RoutingSnapshot) (bool, error) {
	current, err := db.BuildRoutingSnapshot(ctx, product, scope, sku)
	if err != nil {
		return false, err
	}
	if len(current.Providers) != len(expected.Providers) {
		return false, nil
	}
	for p, pct := range expected.Providers {
		if current.Providers[p] != pct {
			return false, nil
		}
	}
	return true, nil
}
