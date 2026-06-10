package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// LatestCycle is the most recent analysis cycle.
type LatestCycle struct {
	ID            uint64
	CycleStarted  time.Time
	CycleFinished sql.NullTime
	HealthStatus  string
	HealthSummary sql.NullString
	Decision      sql.NullString
}

// GetLatestCycle returns newest successful analysis cycle.
func (db *DB) GetLatestCycle(ctx context.Context) (LatestCycle, bool, error) {
	const query = `
		SELECT id, cycle_started, cycle_finished, health_status, health_summary, decision
		FROM agent_analysis_cycles
		WHERE status = 'success'
		ORDER BY id DESC
		LIMIT 1`
	var c LatestCycle
	err := db.QueryRowContext(ctx, query).Scan(
		&c.ID, &c.CycleStarted, &c.CycleFinished, &c.HealthStatus, &c.HealthSummary, &c.Decision,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return LatestCycle{}, false, nil
	}
	if err != nil {
		return LatestCycle{}, false, err
	}
	return c, true, nil
}

// ProductHealthRow per product for a cycle.
type ProductHealthRow struct {
	ProductCode   string
	HealthStatus  string
	HealthSummary sql.NullString
}

// ListProductHealthByCycle returns health_status_product rows.
func (db *DB) ListProductHealthByCycle(ctx context.Context, cycleID uint64) ([]ProductHealthRow, error) {
	const query = `
		SELECT product_code, health_status, health_summary
		FROM health_status_product
		WHERE cycle_id = ?
		ORDER BY product_code`
	rows, err := db.QueryContext(ctx, query, cycleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProductHealthRow
	for rows.Next() {
		var r ProductHealthRow
		if err := rows.Scan(&r.ProductCode, &r.HealthStatus, &r.HealthSummary); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// IncidentRow list item.
type IncidentRow struct {
	ID           uint64
	IncidentID   string
	CycleID      sql.NullInt64
	Severity     string
	ProductCode  string
	ProviderCode string
	SKUCode      string
	Summary          sql.NullString
	Status           string
	HandledBy        sql.NullString
	HandledAt        sql.NullTime
	ResolutionAction sql.NullString
	CreatedAt        time.Time
}

const incidentSelectCols = `
	id, incident_id, cycle_id, severity, product_code, provider_code, sku_code,
	summary, status, handled_by, handled_at, resolution_action, created_at`

func scanIncidentRow(scanner interface {
	Scan(dest ...any) error
}) (IncidentRow, error) {
	var r IncidentRow
	err := scanner.Scan(
		&r.ID, &r.IncidentID, &r.CycleID, &r.Severity, &r.ProductCode,
		&r.ProviderCode, &r.SKUCode, &r.Summary, &r.Status,
		&r.HandledBy, &r.HandledAt, &r.ResolutionAction, &r.CreatedAt,
	)
	return r, err
}

// CountIncidents returns total rows (optional since filter).
func (db *DB) CountIncidents(ctx context.Context, since *time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM incidents`
	var args []any
	if since != nil {
		query += ` WHERE created_at >= ?`
		args = append(args, *since)
	}
	var n int
	err := db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

// ListIncidents returns a page of incidents (since optional, limit/offset pagination).
func (db *DB) ListIncidents(ctx context.Context, since *time.Time, limit, offset int) ([]IncidentRow, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	query := `SELECT` + incidentSelectCols + ` FROM incidents`
	var args []any
	if since != nil {
		query += ` WHERE created_at >= ?`
		args = append(args, *since)
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IncidentRow
	for rows.Next() {
		r, err := scanIncidentRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetIncidentByID loads one incident by incident_id string.
func (db *DB) GetIncidentByID(ctx context.Context, incidentID string) (IncidentRow, error) {
	const query = `SELECT` + incidentSelectCols + ` FROM incidents WHERE incident_id = ?`
	r, err := scanIncidentRow(db.QueryRowContext(ctx, query, incidentID))
	if errors.Is(err, sql.ErrNoRows) {
		return IncidentRow{}, fmt.Errorf("incident %q not found", incidentID)
	}
	return r, err
}

// RoutingPlanRow from DB.
type RoutingPlanRow struct {
	ID          uint64
	CycleID     sql.NullInt64
	ProductCode string
	Scope       string
	SKUCode     string
	PlanJSON    json.RawMessage
	Status      string
	CreatedAt   time.Time
}

// ListLatestRoutingPlans returns recent plans.
func (db *DB) ListLatestRoutingPlans(ctx context.Context, limit int) ([]RoutingPlanRow, error) {
	if limit <= 0 {
		limit = 10
	}
	const query = `
		SELECT id, cycle_id, product_code, scope, sku_code, plan_json, status, created_at
		FROM routing_plans
		ORDER BY created_at DESC
		LIMIT ?`
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRoutingPlans(rows)
}

// GetRoutingPlan loads plan by id.
func (db *DB) GetRoutingPlan(ctx context.Context, id uint64) (RoutingPlanRow, error) {
	const query = `
		SELECT id, cycle_id, product_code, scope, sku_code, plan_json, status, created_at
		FROM routing_plans WHERE id = ?`
	var r RoutingPlanRow
	err := db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.CycleID, &r.ProductCode, &r.Scope, &r.SKUCode, &r.PlanJSON, &r.Status, &r.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingPlanRow{}, fmt.Errorf("routing plan %d not found", id)
	}
	return r, err
}

// UpdateRoutingPlanStatus updates plan status.
func (db *DB) UpdateRoutingPlanStatus(ctx context.Context, id uint64, status, approvedBy string) error {
	if approvedBy != "" {
		_, err := db.ExecContext(ctx, `
			UPDATE routing_plans SET status = ?, approved_by = ?, approved_at = NOW() WHERE id = ?`,
			status, approvedBy, id)
		return err
	}
	_, err := db.ExecContext(ctx, `UPDATE routing_plans SET status = ? WHERE id = ?`, status, id)
	return err
}

func scanRoutingPlans(rows *sql.Rows) ([]RoutingPlanRow, error) {
	var out []RoutingPlanRow
	for rows.Next() {
		var r RoutingPlanRow
		if err := rows.Scan(&r.ID, &r.CycleID, &r.ProductCode, &r.Scope, &r.SKUCode, &r.PlanJSON, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// NotificationRow list item.
type NotificationRow struct {
	ID          uint64
	DedupeKey   string
	ProductCode string
	ProviderCode string
	Subject     string
	Status      string
	SentAt      sql.NullTime
	CreatedAt   time.Time
}

// ListNotifications returns recent notification_log rows.
func (db *DB) ListNotifications(ctx context.Context, limit int) ([]NotificationRow, error) {
	if limit <= 0 {
		limit = 50
	}
	const query = `
		SELECT id, dedupe_key, product_code, provider_code, subject, status, sent_at, created_at
		FROM notification_log
		ORDER BY created_at DESC
		LIMIT ?`
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NotificationRow
	for rows.Next() {
		var r NotificationRow
		if err := rows.Scan(&r.ID, &r.DedupeKey, &r.ProductCode, &r.ProviderCode, &r.Subject, &r.Status, &r.SentAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MaintenanceRow list item.
type MaintenanceRow struct {
	MaintenanceID string
	ProductCode   string
	ProviderCode  string
	SKUCode       string
	StartsAt      time.Time
	EndsAt        time.Time
	Status        string
	Reason        sql.NullString
}

// ListMaintenanceWindows lists maintenance windows.
func (db *DB) ListMaintenanceWindows(ctx context.Context, product, status string, limit int) ([]MaintenanceRow, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT maintenance_id, product_code, provider_code, sku_code, starts_at, ends_at, status, reason
		FROM maintenance_windows WHERE 1=1`
	var args []any
	if product != "" {
		query += ` AND product_code = ?`
		args = append(args, product)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY starts_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MaintenanceRow
	for rows.Next() {
		var r MaintenanceRow
		if err := rows.Scan(&r.MaintenanceID, &r.ProductCode, &r.ProviderCode, &r.SKUCode,
			&r.StartsAt, &r.EndsAt, &r.Status, &r.Reason); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetMaintenanceByID loads one window.
func (db *DB) GetMaintenanceByID(ctx context.Context, maintenanceID string) (MaintenanceRow, error) {
	const query = `
		SELECT maintenance_id, product_code, provider_code, sku_code, starts_at, ends_at, status, reason
		FROM maintenance_windows WHERE maintenance_id = ?`
	var r MaintenanceRow
	err := db.QueryRowContext(ctx, query, maintenanceID).Scan(
		&r.MaintenanceID, &r.ProductCode, &r.ProviderCode, &r.SKUCode,
		&r.StartsAt, &r.EndsAt, &r.Status, &r.Reason,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return MaintenanceRow{}, fmt.Errorf("maintenance %q not found", maintenanceID)
	}
	return r, err
}

// EscalationRow chat config.
type EscalationRow struct {
	ProviderCode  string
	ChatAppName   string
	ChatGroupName string
	MentionTags   string
}

// ListChatEscalations returns all provider chat configs.
func (db *DB) ListChatEscalations(ctx context.Context) ([]EscalationRow, error) {
	const query = `
		SELECT provider_code, chat_app_name, chat_group_name, mention_tags
		FROM provider_chat_escalation WHERE enabled = 1
		ORDER BY provider_code`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EscalationRow
	for rows.Next() {
		var r EscalationRow
		if err := rows.Scan(&r.ProviderCode, &r.ChatAppName, &r.ChatGroupName, &r.MentionTags); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountMockMetrics returns mock_metrics count for status endpoint.
func (db *DB) CountMockGeneratorRuns(ctx context.Context) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mock_generator_run WHERE status = 'success'`).Scan(&n)
	return n, err
}

// GetLastMockRun returns last mock run info.
func (db *DB) GetLastMockRun(ctx context.Context) (scenario string, started time.Time, rows int, ok bool, err error) {
	const query = `
		SELECT scenario, started_at, rows_metrics FROM mock_generator_run
		WHERE status = 'success' ORDER BY id DESC LIMIT 1`
	err = db.QueryRowContext(ctx, query).Scan(&scenario, &started, &rows)
	if errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, 0, false, nil
	}
	if err != nil {
		return "", time.Time{}, 0, false, err
	}
	return scenario, started, rows, true, nil
}
