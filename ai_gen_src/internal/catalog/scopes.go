package catalog

import (
	"context"
	"fmt"

	"opsone/internal/domain"
	"opsone/internal/store"
)

// MetricScope is one row of mock/agent metrics (product × sku × provider).
type MetricScope struct {
	ProductCode  string
	ServiceType  domain.ServiceType
	SKUCode      string
	ProviderCode string
}

// ListMetricScopes expands enabled catalog per §4.5.1 / §5.4.
func ListMetricScopes(ctx context.Context, db *store.DB) ([]MetricScope, error) {
	products, err := db.ListProducts(ctx, true)
	if err != nil {
		return nil, err
	}

	var scopes []MetricScope
	for _, p := range products {
		providers, err := db.ListProvidersForProduct(ctx, p.ProductCode, true)
		if err != nil {
			return nil, fmt.Errorf("providers %s: %w", p.ProductCode, err)
		}
		if p.RoutingMode == domain.RoutingProvider {
			for _, pr := range providers {
				scopes = append(scopes, MetricScope{
					ProductCode:  p.ProductCode,
					ServiceType:  p.ServiceType,
					SKUCode:      "",
					ProviderCode: pr.ProviderCode,
				})
			}
			continue
		}
		skus, err := db.ListSKUsForProduct(ctx, p.ProductCode, true)
		if err != nil {
			return nil, fmt.Errorf("skus %s: %w", p.ProductCode, err)
		}
		for _, sku := range skus {
			for _, pr := range providers {
				scopes = append(scopes, MetricScope{
					ProductCode:  p.ProductCode,
					ServiceType:  p.ServiceType,
					SKUCode:      sku.SKUCode,
					ProviderCode: pr.ProviderCode,
				})
			}
		}
	}
	return scopes, nil
}
