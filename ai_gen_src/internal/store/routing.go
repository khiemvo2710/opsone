package store

import (
	"context"
	"fmt"

	"opsone/internal/domain"
)

// GetRoutingForScope returns routing rows for product + sku scope.
func (db *DB) GetRoutingForScope(ctx context.Context, productCode, skuCode string) ([]domain.RoutingPct, error) {
	const query = `
		SELECT product_code, sku_code, provider_code, baseline_pct, traffic_pct
		FROM routing_config
		WHERE product_code = ? AND sku_code = ?
		ORDER BY provider_code`

	rows, err := db.QueryContext(ctx, query, productCode, skuCode)
	if err != nil {
		return nil, fmt.Errorf("get routing: %w", err)
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

// SumTrafficPct validates traffic_pct sums to 100 for a scope.
func (db *DB) SumTrafficPct(ctx context.Context, productCode, skuCode string) (float64, error) {
	const query = `
		SELECT COALESCE(SUM(traffic_pct), 0)
		FROM routing_config
		WHERE product_code = ? AND sku_code = ?`
	var sum float64
	if err := db.QueryRowContext(ctx, query, productCode, skuCode).Scan(&sum); err != nil {
		return 0, fmt.Errorf("sum traffic_pct: %w", err)
	}
	return sum, nil
}

// GetAgentSettingsLocale returns agent_locale from agent_settings row 1.
func (db *DB) GetAgentSettingsLocale(ctx context.Context) (string, error) {
	const query = `SELECT agent_locale FROM agent_settings WHERE id = 1`
	var locale string
	if err := db.QueryRowContext(ctx, query).Scan(&locale); err != nil {
		return "", fmt.Errorf("get agent locale: %w", err)
	}
	return locale, nil
}
