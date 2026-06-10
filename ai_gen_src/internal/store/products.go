package store

import (
	"context"
	"database/sql"
	"fmt"

	"opsone/internal/domain"
)

// CountProducts returns the number of enabled products (or all if enabledOnly is false).
func (db *DB) CountProducts(ctx context.Context, enabledOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM products WHERE enabled = 1`
	if !enabledOnly {
		query = `SELECT COUNT(*) FROM products`
	}
	var n int
	if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
		return 0, fmt.Errorf("count products: %w", err)
	}
	return n, nil
}

// ListProducts returns catalog products ordered by product_code.
func (db *DB) ListProducts(ctx context.Context, enabledOnly bool) ([]domain.Product, error) {
	query := `
		SELECT id, product_code, label, service_type, routing_mode, enabled
		FROM products
		WHERE enabled = 1
		ORDER BY product_code`
	if !enabledOnly {
		query = `
			SELECT id, product_code, label, service_type, routing_mode, enabled
			FROM products
			ORDER BY product_code`
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var out []domain.Product
	for rows.Next() {
		var p domain.Product
		var svc, mode string
		var enabled int
		if err := rows.Scan(&p.ID, &p.ProductCode, &p.Label, &svc, &mode, &enabled); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		p.ServiceType = domain.ServiceType(svc)
		p.RoutingMode = domain.RoutingMode(mode)
		p.Enabled = enabled == 1
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return out, nil
}

// GetProductByCode loads one product by product_code.
func (db *DB) GetProductByCode(ctx context.Context, productCode string) (domain.Product, error) {
	const query = `
		SELECT id, product_code, label, service_type, routing_mode, enabled
		FROM products
		WHERE product_code = ?`
	var p domain.Product
	var svc, mode string
	var enabled int
	err := db.QueryRowContext(ctx, query, productCode).Scan(
		&p.ID, &p.ProductCode, &p.Label, &svc, &mode, &enabled,
	)
	if err == sql.ErrNoRows {
		return domain.Product{}, fmt.Errorf("product %q not found", productCode)
	}
	if err != nil {
		return domain.Product{}, fmt.Errorf("get product: %w", err)
	}
	p.ServiceType = domain.ServiceType(svc)
	p.RoutingMode = domain.RoutingMode(mode)
	p.Enabled = enabled == 1
	return p, nil
}

// ListProvidersForProduct returns active providers for a product (§6.3).
func (db *DB) ListProvidersForProduct(ctx context.Context, productCode string, activeOnly bool) ([]domain.Provider, error) {
	query := `
		SELECT pr.provider_code, pr.label, pp.enabled AND pr.enabled AS enabled
		FROM products p
		JOIN product_providers pp ON pp.product_id = p.id
		JOIN providers pr ON pr.id = pp.provider_id
		WHERE p.product_code = ?`
	if activeOnly {
		query += ` AND pp.enabled = 1 AND pr.enabled = 1`
	}
	query += ` ORDER BY pr.provider_code`

	rows, err := db.QueryContext(ctx, query, productCode)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()

	var out []domain.Provider
	for rows.Next() {
		var pr domain.Provider
		var enabled int
		if err := rows.Scan(&pr.ProviderCode, &pr.Label, &enabled); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		pr.Enabled = enabled == 1
		out = append(out, pr)
	}
	return out, rows.Err()
}

// ListSKUsForProduct returns SKUs for a sku-mode product.
func (db *DB) ListSKUsForProduct(ctx context.Context, productCode string, enabledOnly bool) ([]domain.SKU, error) {
	query := `
		SELECT ps.sku_code, ps.label, ps.enabled
		FROM products p
		JOIN product_skus ps ON ps.product_id = p.id
		WHERE p.product_code = ?`
	if enabledOnly {
		query += ` AND ps.enabled = 1`
	}
	query += ` ORDER BY
		CASE WHEN ps.sku_code REGEXP '^[0-9]+$' THEN 0 ELSE 1 END,
		CASE WHEN ps.sku_code REGEXP '^[0-9]+$' THEN CAST(ps.sku_code AS UNSIGNED) ELSE NULL END,
		ps.sku_code`

	rows, err := db.QueryContext(ctx, query, productCode)
	if err != nil {
		return nil, fmt.Errorf("list skus: %w", err)
	}
	defer rows.Close()

	var out []domain.SKU
	for rows.Next() {
		var s domain.SKU
		var enabled int
		if err := rows.Scan(&s.SKUCode, &s.Label, &enabled); err != nil {
			return nil, fmt.Errorf("scan sku: %w", err)
		}
		s.Enabled = enabled == 1
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountActiveProviders returns providers with enabled=1 for product.
func (db *DB) CountActiveProviders(ctx context.Context, productCode string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM products p
		JOIN product_providers pp ON pp.product_id = p.id AND pp.enabled = 1
		JOIN providers pr ON pr.id = pp.provider_id AND pr.enabled = 1
		WHERE p.product_code = ?`
	var n int
	if err := db.QueryRowContext(ctx, query, productCode).Scan(&n); err != nil {
		return 0, fmt.Errorf("count active providers: %w", err)
	}
	return n, nil
}
