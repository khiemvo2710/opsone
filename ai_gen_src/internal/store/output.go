package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// InsertIncident creates an incident row (§8.3).
func (db *DB) InsertIncident(ctx context.Context, p IncidentInsert) error {
	const query = `
		INSERT INTO incidents (
			incident_id, cycle_id, severity, product_code, provider_code, sku_code,
			success_before, success_after, fail_before, fail_after, summary, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'open')`
	_, err := db.ExecContext(ctx, query,
		p.IncidentID, p.CycleID, p.Severity, p.ProductCode, p.ProviderCode, p.SKUCode,
		p.SuccessBefore, p.SuccessAfter, p.FailBefore, p.FailAfter, p.Summary,
	)
	return err
}

// IncidentInsert params.
type IncidentInsert struct {
	IncidentID    string
	CycleID       *uint64
	Severity      string
	ProductCode   string
	ProviderCode  string
	SKUCode       string
	SuccessBefore *float64
	SuccessAfter  *float64
	FailBefore    *float64
	FailAfter     *float64
	Summary       string
}

// NextIncidentID generates YYYYMMDD-NNN for today.
func (db *DB) NextIncidentID(ctx context.Context, day time.Time) (string, error) {
	prefix := day.Format("20060102")
	const query = `
		SELECT COUNT(*) FROM incidents
		WHERE incident_id LIKE ?`
	var n int
	err := db.QueryRowContext(ctx, query, prefix+"-%").Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%03d", prefix, n+1), nil
}

// InsertRoutingPlan stores routing plan JSON (§8.6).
func (db *DB) InsertRoutingPlan(ctx context.Context, cycleID *uint64, product, scope, sku string, plan any, status string) (uint64, error) {
	raw, err := json.Marshal(plan)
	if err != nil {
		return 0, err
	}
	if status == "" {
		status = "pending_approve"
	}
	const query = `
		INSERT INTO routing_plans (cycle_id, product_code, scope, sku_code, plan_json, status)
		VALUES (?, ?, ?, ?, ?, ?)`
	res, err := db.ExecContext(ctx, query, cycleID, product, scope, sku, raw, status)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

// UpdatePendingRoutingPlanForScope refreshes plan_json on the latest pending plan for a scope.
func (db *DB) UpdatePendingRoutingPlanForScope(ctx context.Context, cycleID *uint64, product, sku string, plan any) error {
	raw, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	const query = `
		UPDATE routing_plans SET plan_json = ?, cycle_id = COALESCE(?, cycle_id)
		WHERE id = (
			SELECT max_id FROM (
				SELECT MAX(id) AS max_id FROM routing_plans
				WHERE product_code = ? AND sku_code = ? AND status IN ('pending_approve', 'draft')
			) latest
		)`
	_, err = db.ExecContext(ctx, query, raw, cycleID, product, sku)
	return err
}

// InsertRecommendation stores monitor/maintenance recommendation.
func (db *DB) InsertRecommendation(ctx context.Context, cycleID *uint64, incidentID *string, product, sku, actionType, detail string) error {
	_, err := db.InsertRecommendationReturningID(ctx, cycleID, incidentID, product, sku, actionType, detail)
	return err
}

// InsertRecommendationReturningID stores a recommendation and returns its id.
func (db *DB) InsertRecommendationReturningID(ctx context.Context, cycleID *uint64, incidentID *string, product, sku, actionType, detail string) (uint64, error) {
	if sku != "" && actionType == "maintenance" {
		detail = FormatMaintenanceDetail(sku, detail)
	}
	const query = `
		INSERT INTO recommendations (cycle_id, incident_id, product_code, action_type, action_detail)
		VALUES (?, ?, ?, ?, ?)`
	res, err := db.ExecContext(ctx, query, cycleID, incidentID, product, actionType, detail)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

// FindOpenIncidentForScope returns the latest open incident_id for product/sku (optional cycle match).
func (db *DB) FindOpenIncidentForScope(ctx context.Context, product, sku string, cycleID *uint64) (string, error) {
	if cycleID != nil {
		var incidentID string
		err := db.QueryRowContext(ctx, `
			SELECT incident_id FROM incidents
			WHERE status = 'open' AND product_code = ? AND sku_code = ? AND cycle_id = ?
			ORDER BY created_at DESC LIMIT 1`, product, sku, *cycleID).Scan(&incidentID)
		if err == nil {
			return incidentID, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("find open incident: %w", err)
		}
	}
	var incidentID string
	err := db.QueryRowContext(ctx, `
		SELECT incident_id FROM incidents
		WHERE status = 'open' AND product_code = ? AND sku_code = ?
		ORDER BY created_at DESC LIMIT 1`, product, sku).Scan(&incidentID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find open incident: %w", err)
	}
	return incidentID, nil
}

// IncidentHandleUpdate persists status change with handler audit (§8.3).
type IncidentHandleUpdate struct {
	Status           string
	HandledBy        string
	ResolutionAction string
}

// UpdateIncidentHandled sets status, handler, timestamp and resolution action.
func (db *DB) UpdateIncidentHandled(ctx context.Context, incidentID string, u IncidentHandleUpdate) error {
	if incidentID == "" {
		return nil
	}
	if u.HandledBy == "" {
		u.HandledBy = "opsone-agent"
	}
	res, err := db.ExecContext(ctx, `
		UPDATE incidents SET
			status = ?,
			handled_by = ?,
			handled_at = COALESCE(handled_at, NOW()),
			resolution_action = COALESCE(?, resolution_action)
		WHERE incident_id = ?
		  AND (status = 'open' OR handled_at IS NULL)`,
		u.Status, u.HandledBy, nullIfEmpty(u.ResolutionAction), incidentID)
	if err != nil {
		return fmt.Errorf("update incident handled: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("incident %q not found or already handled", incidentID)
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ResolveOpenIncidentForScope marks the latest open incident resolved for a routing scope.
func (db *DB) ResolveOpenIncidentForScope(ctx context.Context, product, sku string, cycleID *uint64, handledBy, resolutionAction string) error {
	incID, err := db.FindOpenIncidentForScope(ctx, product, sku, cycleID)
	if err != nil || incID == "" {
		return err
	}
	return db.UpdateIncidentHandled(ctx, incID, IncidentHandleUpdate{
		Status:           "resolved",
		HandledBy:        handledBy,
		ResolutionAction: resolutionAction,
	})
}
