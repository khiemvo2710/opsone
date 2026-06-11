package tools

import (
	"time"

	"opsone/internal/store"
)

// MaintenanceInWindow matches dashboard overview filter (§9.0).
func MaintenanceInWindow(starts, ends, now time.Time) bool {
	return !ends.Before(now) && !starts.After(now)
}

// BuildMaintenanceOutput classifies rows like dashboard + §6.8.
func BuildMaintenanceOutput(rows []store.MaintenanceWindow, now time.Time, provider, sku string) GetMaintenanceOutput {
	out := GetMaintenanceOutput{Meta: Meta{QueriedAt: now}}
	for _, w := range rows {
		if provider != "" && w.ProviderCode != provider {
			continue
		}
		if sku != "" && w.SKUCode != sku {
			continue
		}
		item := MaintenanceItem{
			MaintenanceID: w.MaintenanceID,
			ProductCode:   w.ProductCode,
			ProviderCode:  w.ProviderCode,
			SKUCode:       w.SKUCode,
			StartsAt:      w.StartsAt,
			EndsAt:        w.EndsAt,
			Status:        w.Status,
		}
		if w.Reason.Valid {
			item.Reason = w.Reason.String
		}
		if MaintenanceInWindow(w.StartsAt, w.EndsAt, now) {
			item.RemainingMinutes = int(w.EndsAt.Sub(now).Minutes())
			if item.RemainingMinutes < 1 {
				item.RemainingMinutes = 1
			}
			out.Active = append(out.Active, item)
		} else if w.StartsAt.After(now) {
			out.Scheduled = append(out.Scheduled, item)
		}
	}
	return out
}

// BuildMaintenanceOutputFromRows converts dashboard list rows (status=active query).
func BuildMaintenanceOutputFromRows(rows []store.MaintenanceRow, now time.Time, provider, sku string) GetMaintenanceOutput {
	windows := make([]store.MaintenanceWindow, 0, len(rows))
	for _, r := range rows {
		windows = append(windows, store.MaintenanceWindow{
			MaintenanceID: r.MaintenanceID,
			ProductCode:   r.ProductCode,
			ProviderCode:  r.ProviderCode,
			SKUCode:       r.SKUCode,
			StartsAt:      r.StartsAt,
			EndsAt:        r.EndsAt,
			Status:        r.Status,
			Reason:        r.Reason,
		})
	}
	return BuildMaintenanceOutput(windows, now, provider, sku)
}

// FilterMaintenanceByProvider keeps only one routing provider in tool/chat output.
func FilterMaintenanceByProvider(out GetMaintenanceOutput, provider string) GetMaintenanceOutput {
	if provider == "" {
		return out
	}
	filtered := GetMaintenanceOutput{Meta: out.Meta}
	for _, item := range out.Active {
		if item.ProviderCode == provider {
			filtered.Active = append(filtered.Active, item)
		}
	}
	for _, item := range out.Scheduled {
		if item.ProviderCode == provider {
			filtered.Scheduled = append(filtered.Scheduled, item)
		}
	}
	return filtered
}
