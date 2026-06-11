package tools

import (
	"context"
	"time"
)

// MaintenanceItem active or scheduled window.
type MaintenanceItem struct {
	MaintenanceID    string    `json:"maintenance_id"`
	ProductCode      string    `json:"product_code,omitempty"`
	ProviderCode     string    `json:"provider_code,omitempty"`
	SKUCode          string    `json:"sku_code,omitempty"`
	StartsAt         time.Time `json:"starts_at"`
	EndsAt           time.Time `json:"ends_at"`
	Status           string    `json:"status"`
	RemainingMinutes int       `json:"remaining_minutes,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}

// GetMaintenanceInput §6.8.
type GetMaintenanceInput struct {
	Product  string
	Provider string
	SKU      string
}

// GetMaintenanceOutput §6.8.
type GetMaintenanceOutput struct {
	Active    []MaintenanceItem `json:"active"`
	Scheduled []MaintenanceItem `json:"scheduled"`
	Meta      Meta              `json:"meta"`
}

// GetMaintenance lists maintenance windows (§6.8).
// provider/sku optional — để trống để liệt kê mọi mệnh giá/provider của dịch vụ (vd thẻ GARENA).
func (r *Registry) GetMaintenance(ctx context.Context, in GetMaintenanceInput) (GetMaintenanceOutput, error) {
	if in.Product == "" {
		return GetMaintenanceOutput{}, newErr("invalid_input", "Thiếu product")
	}
	ds, err := r.dataSource(ctx)
	if err != nil {
		return GetMaintenanceOutput{}, err
	}
	rows, err := r.DB.ListActiveMaintenance(ctx, in.Product, in.Provider, in.SKU)
	if err != nil {
		return GetMaintenanceOutput{}, err
	}
	now := time.Now()
	out := BuildMaintenanceOutput(rows, now, in.Provider, in.SKU)
	out.Meta = Meta{DataSource: ds, QueriedAt: now}
	return out, nil
}
