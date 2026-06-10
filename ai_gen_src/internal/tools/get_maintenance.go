package tools

import (
	"context"
	"time"
)

// MaintenanceItem active or scheduled window.
type MaintenanceItem struct {
	MaintenanceID    string    `json:"maintenance_id"`
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
func (r *Registry) GetMaintenance(ctx context.Context, in GetMaintenanceInput) (GetMaintenanceOutput, error) {
	if in.Product == "" || in.Provider == "" {
		return GetMaintenanceOutput{}, newErr("invalid_input", "Thiếu product hoặc provider")
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
	out := GetMaintenanceOutput{Meta: Meta{DataSource: ds, QueriedAt: now}}
	for _, w := range rows {
		item := MaintenanceItem{
			MaintenanceID: w.MaintenanceID,
			StartsAt:      w.StartsAt,
			EndsAt:        w.EndsAt,
			Status:        w.Status,
		}
		if w.Reason.Valid {
			item.Reason = w.Reason.String
		}
		if w.Status == "active" && w.EndsAt.After(now) {
			item.RemainingMinutes = int(w.EndsAt.Sub(now).Minutes())
			out.Active = append(out.Active, item)
		} else if w.Status == "scheduled" {
			out.Scheduled = append(out.Scheduled, item)
		}
	}
	return out, nil
}
