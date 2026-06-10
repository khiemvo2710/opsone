package tools

import (
	"context"
	"time"
)

// GetProvidersInput §6.3.
type GetProvidersInput struct {
	Product string
}

// GetProvidersOutput §6.3.
type GetProvidersOutput struct {
	Product           string   `json:"product"`
	ActiveProviders   []string `json:"active_providers"`
	InactiveProviders []string `json:"inactive_providers"`
	ActiveCount       int      `json:"active_count"`
	TotalCount        int      `json:"total_count"`
	Meta              Meta     `json:"meta"`
}

// GetProviders lists active/inactive providers (§6.3).
func (r *Registry) GetProviders(ctx context.Context, in GetProvidersInput) (GetProvidersOutput, error) {
	if in.Product == "" {
		return GetProvidersOutput{}, newErr("invalid_input", "Thiếu product")
	}
	ds, err := r.dataSource(ctx)
	if err != nil {
		return GetProvidersOutput{}, err
	}

	all, err := r.DB.ListProvidersForProduct(ctx, in.Product, false)
	if err != nil {
		return GetProvidersOutput{}, err
	}
	var active, inactive []string
	for _, p := range all {
		if p.Enabled {
			active = append(active, p.ProviderCode)
		} else {
			inactive = append(inactive, p.ProviderCode)
		}
	}
	return GetProvidersOutput{
		Product:           in.Product,
		ActiveProviders:   active,
		InactiveProviders: inactive,
		ActiveCount:       len(active),
		TotalCount:        len(all),
		Meta:              Meta{DataSource: ds, QueriedAt: time.Now()},
	}, nil
}
