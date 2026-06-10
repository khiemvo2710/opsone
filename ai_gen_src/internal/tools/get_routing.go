package tools

import (
	"context"
	"time"

	"opsone/internal/domain"
)

// GetRoutingInput §6.5.
type GetRoutingInput struct {
	Product string
}

// GetRoutingOutput §6.5.
type GetRoutingOutput struct {
	Product      string                       `json:"product"`
	ServiceType  string                       `json:"service_type"`
	Scope        string                       `json:"scope"`
	Routing      map[string]float64           `json:"routing,omitempty"`
	RoutingBySKU map[string]map[string]float64 `json:"routing_by_sku,omitempty"`
	Meta         Meta                         `json:"meta"`
}

// GetRouting returns current traffic_pct (§6.5).
func (r *Registry) GetRouting(ctx context.Context, in GetRoutingInput) (GetRoutingOutput, error) {
	if in.Product == "" {
		return GetRoutingOutput{}, newErr("invalid_input", "Thiếu product")
	}
	ds, err := r.dataSource(ctx)
	if err != nil {
		return GetRoutingOutput{}, err
	}
	p, err := r.DB.GetProductByCode(ctx, in.Product)
	if err != nil {
		return GetRoutingOutput{}, newErr("product_not_found", "Không tìm thấy sản phẩm")
	}

	rows, err := r.DB.GetAllRoutingForProduct(ctx, in.Product)
	if err != nil {
		return GetRoutingOutput{}, err
	}

	out := GetRoutingOutput{
		Product:     in.Product,
		ServiceType: string(p.ServiceType),
		Meta:        Meta{DataSource: ds, QueriedAt: time.Now()},
	}

	if p.RoutingMode == domain.RoutingProvider {
		out.Scope = "provider"
		out.Routing = make(map[string]float64)
		for _, row := range rows {
			if row.SKUCode == "" {
				out.Routing[row.ProviderCode] = row.TrafficPct
			}
		}
		return out, nil
	}

	out.Scope = "sku"
	out.RoutingBySKU = make(map[string]map[string]float64)
	for _, row := range rows {
		if out.RoutingBySKU[row.SKUCode] == nil {
			out.RoutingBySKU[row.SKUCode] = make(map[string]float64)
		}
		out.RoutingBySKU[row.SKUCode][row.ProviderCode] = row.TrafficPct
	}
	return out, nil
}
