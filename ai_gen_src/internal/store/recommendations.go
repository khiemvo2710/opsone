package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PendingRecommendation is an open maintenance/monitor suggestion for dashboard (§9.0).
type PendingRecommendation struct {
	ID           uint64
	ProductCode  string
	SKUCode      string
	ProviderCode string
	Detail       string
	ActionType   string
	CreatedAt    time.Time
}

// FormatMaintenanceDetail prefixes SKU for per-scope matching on dashboard.
func FormatMaintenanceDetail(sku, detail string) string {
	if sku == "" {
		return detail
	}
	return fmt.Sprintf("SKU %s — %s", sku, detail)
}

func parseSKUFromDetail(detail string) string {
	if !strings.HasPrefix(detail, "SKU ") {
		return ""
	}
	head, _, _ := strings.Cut(detail, " — ")
	return strings.TrimSpace(strings.TrimPrefix(head, "SKU "))
}

func parseProviderFromDetail(detail string) string {
	upper := strings.ToUpper(detail)
	for _, p := range []string{"ESALE", "IMEDIA", "SHOPPAY"} {
		if strings.Contains(upper, p) {
			return p
		}
	}
	return ""
}

// HasRecentRecommendation reports whether product+sku has a recommendation within 24h.
func (db *DB) HasRecentRecommendation(ctx context.Context, product, sku, actionType string) (bool, error) {
	var n int
	if sku != "" {
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM recommendations
			WHERE product_code = ? AND action_type = ?
			  AND action_detail LIKE ?
			  AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)`,
			product, actionType, "SKU "+sku+"%",
		).Scan(&n)
		if err != nil {
			return false, fmt.Errorf("has recent recommendation: %w", err)
		}
		if n > 0 {
			return true, nil
		}
	}
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM recommendations
		WHERE product_code = ? AND action_type = ?
		  AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)`,
		product, actionType,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("has recent recommendation: %w", err)
	}
	return n > 0, nil
}

// LatestPendingRecommendation returns the newest recommendation for a product.
func (db *DB) LatestPendingRecommendation(ctx context.Context, product, actionType string) (PendingRecommendation, bool, error) {
	const query = `
		SELECT id, product_code, action_type, action_detail, created_at
		FROM recommendations
		WHERE product_code = ? AND action_type = ?
		ORDER BY created_at DESC
		LIMIT 1`
	return scanPendingRecommendationRow(ctx, db, query, product, actionType)
}

// MaintenanceScopeKey is the map key for product+sku maintenance lookups.
func MaintenanceScopeKey(product, sku string) string {
	return product + "\x00" + sku
}

// ListPendingMaintenanceByScope returns newest maintenance recommendation per product+sku (24h).
func (db *DB) ListPendingMaintenanceByScope(ctx context.Context) (map[string]PendingRecommendation, error) {
	const query = `
		SELECT id, product_code, action_type, action_detail, created_at
		FROM recommendations
		WHERE action_type = 'maintenance'
		  AND action_detail NOT LIKE '%DISMISSED:%'
		  AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]PendingRecommendation)
	for rows.Next() {
		var rec PendingRecommendation
		var detail string
		if err := rows.Scan(&rec.ID, &rec.ProductCode, &rec.ActionType, &detail, &rec.CreatedAt); err != nil {
			return nil, err
		}
		rec.Detail = detail
		rec.SKUCode = parseSKUFromDetail(detail)
		rec.ProviderCode = parseProviderFromDetail(detail)
		key := MaintenanceScopeKey(rec.ProductCode, rec.SKUCode)
		if _, ok := out[key]; ok {
			continue
		}
		out[key] = rec
	}
	return out, rows.Err()
}

// LatestDismissedMaintenanceCycleForScope returns cycle_id of the newest DISMISSED maintenance row for a scope.
func (db *DB) LatestDismissedMaintenanceCycleForScope(ctx context.Context, product, sku string) (uint64, bool, error) {
	pattern := "%"
	extraFilter := ""
	if sku != "" {
		pattern = "SKU " + sku + "%"
	} else {
		extraFilter = ` AND action_detail NOT LIKE 'SKU %'`
	}
	const base = `
		SELECT cycle_id FROM recommendations
		WHERE product_code = ? AND action_type = 'maintenance'
		  AND action_detail LIKE '%DISMISSED:%'
		  AND action_detail LIKE ?`
	var cycleID sql.NullInt64
	err := db.QueryRowContext(ctx, base+extraFilter+`
		ORDER BY id DESC
		LIMIT 1`, product, pattern).Scan(&cycleID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !cycleID.Valid {
		return 0, true, nil
	}
	return uint64(cycleID.Int64), true, nil
}

// LatestPendingMaintenanceForScope returns the newest maintenance recommendation for product+sku.
func (db *DB) LatestPendingMaintenanceForScope(ctx context.Context, product, sku string) (PendingRecommendation, bool, error) {
	const query = `
		SELECT id, product_code, action_type, action_detail, created_at
		FROM recommendations
		WHERE product_code = ? AND action_type = 'maintenance'
		  AND action_detail NOT LIKE '%DISMISSED:%'
		  AND action_detail LIKE ?
		  AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		ORDER BY created_at DESC
		LIMIT 1`
	pattern := "%"
	extraFilter := ""
	if sku != "" {
		pattern = "SKU " + sku + "%"
	} else {
		// Provider-level scope (topup): không lấy recommendation gắn SKU thẻ/data.
		extraFilter = ` AND action_detail NOT LIKE 'SKU %'`
	}
	return scanPendingRecommendationRow(ctx, db, query+extraFilter, product, pattern)
}

func scanPendingRecommendationRow(ctx context.Context, db *DB, query string, args ...any) (PendingRecommendation, bool, error) {
	var rec PendingRecommendation
	var detail string
	err := db.QueryRowContext(ctx, query, args...).Scan(
		&rec.ID, &rec.ProductCode, &rec.ActionType, &detail, &rec.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return PendingRecommendation{}, false, nil
	}
	if err != nil {
		return PendingRecommendation{}, false, err
	}
	rec.Detail = detail
	rec.SKUCode = parseSKUFromDetail(detail)
	rec.ProviderCode = parseProviderFromDetail(detail)
	return rec, true, nil
}

// GetRecommendation loads one recommendation by id.
func (db *DB) GetRecommendation(ctx context.Context, id uint64) (PendingRecommendation, error) {
	const query = `
		SELECT id, product_code, action_type, action_detail, created_at
		FROM recommendations WHERE id = ?`
	rec, ok, err := scanPendingRecommendationRow(ctx, db, query, id)
	if err != nil {
		return PendingRecommendation{}, err
	}
	if !ok {
		return PendingRecommendation{}, fmt.Errorf("recommendation %d not found", id)
	}
	return rec, nil
}

// DeleteRecommendation removes a recommendation after approve/reject.
func (db *DB) DeleteRecommendation(ctx context.Context, id uint64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM recommendations WHERE id = ?`, id)
	return err
}
