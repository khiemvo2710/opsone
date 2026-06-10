package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"opsone/internal/domain"
)

// ListAllRoutingConfig returns every routing_config row.
func (db *DB) ListAllRoutingConfig(ctx context.Context) ([]domain.RoutingPct, error) {
	const query = `
		SELECT rc.product_code, rc.sku_code, rc.provider_code, rc.baseline_pct, rc.traffic_pct
		FROM routing_config rc
		ORDER BY rc.product_code,
			CASE WHEN rc.sku_code REGEXP '^[0-9]+$' THEN 0 ELSE 1 END,
			CASE WHEN rc.sku_code REGEXP '^[0-9]+$' THEN CAST(rc.sku_code AS UNSIGNED) ELSE NULL END,
			rc.sku_code, rc.provider_code`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all routing: %w", err)
	}
	defer rows.Close()
	var out []domain.RoutingPct
	for rows.Next() {
		var r domain.RoutingPct
		if err := rows.Scan(&r.ProductCode, &r.SKUCode, &r.ProviderCode, &r.BaselinePct, &r.TrafficPct); err != nil {
			return nil, fmt.Errorf("scan routing: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListPendingRoutingPlansPerScope returns latest pending plan per product+sku.
func (db *DB) ListPendingRoutingPlansPerScope(ctx context.Context) ([]RoutingPlanRow, error) {
	const query = `
		SELECT rp.id, rp.cycle_id, rp.product_code, rp.scope, rp.sku_code, rp.plan_json, rp.status, rp.created_at
		FROM routing_plans rp
		INNER JOIN (
			SELECT product_code, sku_code, MAX(id) AS max_id
			FROM routing_plans
			WHERE status IN ('pending_approve', 'draft')
			GROUP BY product_code, sku_code
		) latest ON rp.id = latest.max_id
		ORDER BY rp.product_code,
			CASE WHEN rp.sku_code REGEXP '^[0-9]+$' THEN 0 ELSE 1 END,
			CASE WHEN rp.sku_code REGEXP '^[0-9]+$' THEN CAST(rp.sku_code AS UNSIGNED) ELSE NULL END,
			rp.sku_code`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRoutingPlans(rows)
}

// CancelPendingRoutingPlansForScope marks pending plans cancelled (e.g. when maintenance is proposed).
func (db *DB) CancelPendingRoutingPlansForScope(ctx context.Context, productCode, skuCode string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE routing_plans SET status = 'cancelled'
		WHERE product_code = ? AND sku_code = ? AND status IN ('pending_approve', 'draft')`,
		productCode, skuCode,
	)
	return err
}

// ListRecentRoutingRejectionScopeKeys returns scopes with a rejected routing plan within the window.
func (db *DB) ListRecentRoutingRejectionScopeKeys(ctx context.Context, within time.Duration) (map[string]struct{}, error) {
	since := time.Now().Add(-within)
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT product_code, sku_code FROM routing_plans
		WHERE status = 'rejected' AND created_at >= ?`, since)
	if err != nil {
		return nil, fmt.Errorf("list routing rejections: %w", err)
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var product, sku string
		if err := rows.Scan(&product, &sku); err != nil {
			return nil, err
		}
		out[MaintenanceScopeKey(product, sku)] = struct{}{}
	}
	return out, rows.Err()
}

// ListDismissedMaintenanceScopeKeys returns scopes dismissed by admin within 2 hours.
func (db *DB) ListDismissedMaintenanceScopeKeys(ctx context.Context) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT product_code, action_detail FROM recommendations
		WHERE action_type = 'maintenance'
		  AND action_detail LIKE ?
		  AND created_at >= DATE_SUB(NOW(), INTERVAL 2 HOUR)`,
		"%DISMISSED:%")
	if err != nil {
		return nil, fmt.Errorf("list dismissed maintenance: %w", err)
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var product, detail string
		if err := rows.Scan(&product, &detail); err != nil {
			return nil, err
		}
		sku := parseSKUFromDetail(detail)
		out[MaintenanceScopeKey(product, sku)] = struct{}{}
	}
	return out, rows.Err()
}

// HasRecentRoutingRejection reports a rejected routing plan for scope within the window.
func (db *DB) HasRecentRoutingRejection(ctx context.Context, productCode, skuCode string, within time.Duration) (bool, error) {
	since := time.Now().Add(-within)
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM routing_plans
		WHERE product_code = ? AND sku_code = ? AND status = 'rejected'
		  AND created_at >= ?`,
		productCode, skuCode, since,
	).Scan(&n)
	return n > 0, err
}

// HasDismissedMaintenanceSuggestion reports admin dismissed synthetic maintenance (2h).
func (db *DB) HasDismissedMaintenanceSuggestion(ctx context.Context, productCode, skuCode string) (bool, error) {
	var n int
	dismissed := "%DISMISSED:%"
	if skuCode != "" {
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM recommendations
			WHERE product_code = ? AND action_type = 'maintenance'
			  AND action_detail LIKE ?
			  AND action_detail LIKE ?
			  AND created_at >= DATE_SUB(NOW(), INTERVAL 2 HOUR)`,
			productCode, dismissed, "SKU "+skuCode+"%",
		).Scan(&n)
		return n > 0, err
	}
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM recommendations
		WHERE product_code = ? AND action_type = 'maintenance'
		  AND action_detail LIKE ?
		  AND created_at >= DATE_SUB(NOW(), INTERVAL 2 HOUR)`,
		productCode, dismissed,
	).Scan(&n)
	return n > 0, err
}

// HasPendingMaintenanceRecommendation reports open maintenance suggestion for scope (24h).
func (db *DB) HasPendingMaintenanceRecommendation(ctx context.Context, productCode, skuCode string) (bool, error) {
	_, ok, err := db.LatestPendingMaintenanceForScope(ctx, productCode, skuCode)
	return ok, err
}

// GetPendingRoutingPlanForScope returns the latest pending routing plan for one scope.
func (db *DB) GetPendingRoutingPlanForScope(ctx context.Context, productCode, skuCode string) (RoutingPlanRow, bool, error) {
	const query = `
		SELECT id, cycle_id, product_code, scope, sku_code, plan_json, status, created_at
		FROM routing_plans
		WHERE product_code = ? AND sku_code = ? AND status IN ('pending_approve', 'draft')
		ORDER BY id DESC
		LIMIT 1`
	var row RoutingPlanRow
	var cycleID sql.NullInt64
	err := db.QueryRowContext(ctx, query, productCode, skuCode).Scan(
		&row.ID, &cycleID, &row.ProductCode, &row.Scope, &row.SKUCode, &row.PlanJSON, &row.Status, &row.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingPlanRow{}, false, nil
	}
	if err != nil {
		return RoutingPlanRow{}, false, err
	}
	row.CycleID = cycleID
	return row, true, nil
}

// HasPendingRoutingPlan reports whether a scope already has a plan awaiting approval.
func (db *DB) HasPendingRoutingPlan(ctx context.Context, productCode, skuCode string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM routing_plans
		WHERE product_code = ? AND sku_code = ? AND status IN ('pending_approve', 'draft')`,
		productCode, skuCode,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ProductMeta holds catalog fields for dashboard rows.
type ProductMeta struct {
	Label       string
	ServiceType string
}

// ProductMetaMap returns product_code → label + service_type.
func (db *DB) ProductMetaMap(ctx context.Context) (map[string]ProductMeta, error) {
	const query = `SELECT product_code, label, service_type FROM products ORDER BY product_code`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]ProductMeta)
	for rows.Next() {
		var code, label, svc string
		if err := rows.Scan(&code, &label, &svc); err != nil {
			return nil, err
		}
		out[code] = ProductMeta{Label: label, ServiceType: svc}
	}
	return out, rows.Err()
}

// ProductLabelMap returns product_code → label.
func (db *DB) ProductLabelMap(ctx context.Context) (map[string]string, error) {
	const query = `SELECT product_code, label FROM products ORDER BY product_code`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var code, label string
		if err := rows.Scan(&code, &label); err != nil {
			return nil, err
		}
		out[code] = label
	}
	return out, rows.Err()
}
