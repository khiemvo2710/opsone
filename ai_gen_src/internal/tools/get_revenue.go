package tools

import (
	"context"
	"time"
)

// GetRevenueInput §6.6.
type GetRevenueInput struct {
	Product  string
	Provider string
	SKU      string
	Window   string
}

// GetRevenueOutput §6.6.
type GetRevenueOutput struct {
	Product         string `json:"product"`
	Provider        string `json:"provider,omitempty"`
	SKU             string `json:"sku,omitempty"`
	RevenueLastHour uint64 `json:"revenue_last_hour"`
	Meta            Meta   `json:"meta"`
}

// GetRevenue reads revenue from latest metric row (§6.6).
func (r *Registry) GetRevenue(ctx context.Context, in GetRevenueInput) (GetRevenueOutput, error) {
	if in.Product == "" || in.Provider == "" {
		return GetRevenueOutput{}, newErr("invalid_input", "Thiếu product hoặc provider")
	}
	dur, winLabel, err := ParseWindow(in.Window)
	if err != nil {
		return GetRevenueOutput{}, err
	}
	ds, err := r.dataSource(ctx)
	if err != nil {
		return GetRevenueOutput{}, err
	}
	since := time.Now().Add(-dur)
	m, ok, err := r.DB.GetMetricsInWindow(ctx, ds, in.Product, in.SKU, in.Provider, since)
	if err != nil {
		return GetRevenueOutput{}, err
	}
	if !ok {
		return GetRevenueOutput{}, newErr("no_data", "Không có dữ liệu doanh thu")
	}
	return GetRevenueOutput{
		Product:         in.Product,
		Provider:        in.Provider,
		SKU:             in.SKU,
		RevenueLastHour: m.RevenueLastHour,
		Meta:            Meta{DataSource: ds, QueriedAt: time.Now(), Window: winLabel},
	}, nil
}
