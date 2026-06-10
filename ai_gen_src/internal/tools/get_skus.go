package tools

import (
	"context"
	"time"

	"opsone/internal/domain"
)

// SKUItem §6.4.
type SKUItem struct {
	SKU   string `json:"sku"`
	Label string `json:"label"`
}

// GetSkusInput §6.4.
type GetSkusInput struct {
	Product string
}

// GetSkusOutput §6.4.
type GetSkusOutput struct {
	Product     string  `json:"product"`
	ServiceType string  `json:"service_type"`
	SKUs        []SKUItem `json:"skus"`
	Meta        Meta    `json:"meta"`
}

// GetSkus returns SKU list for sku-mode products (§6.4).
func (r *Registry) GetSkus(ctx context.Context, in GetSkusInput) (GetSkusOutput, error) {
	if in.Product == "" {
		return GetSkusOutput{}, newErr("invalid_input", "Thiếu product")
	}
	ds, err := r.dataSource(ctx)
	if err != nil {
		return GetSkusOutput{}, err
	}
	p, err := r.DB.GetProductByCode(ctx, in.Product)
	if err != nil {
		return GetSkusOutput{}, newErr("product_not_found", "Không tìm thấy sản phẩm")
	}
	if p.RoutingMode == domain.RoutingProvider {
		return GetSkusOutput{}, newErr("sku_not_applicable", "Topup tiền không dùng SKU")
	}
	skus, err := r.DB.ListSKUsForProduct(ctx, in.Product, true)
	if err != nil {
		return GetSkusOutput{}, err
	}
	items := make([]SKUItem, 0, len(skus))
	for _, s := range skus {
		items = append(items, SKUItem{SKU: s.SKUCode, Label: s.Label})
	}
	return GetSkusOutput{
		Product:     in.Product,
		ServiceType: string(p.ServiceType),
		SKUs:        items,
		Meta:        Meta{DataSource: ds, QueriedAt: time.Now()},
	}, nil
}
