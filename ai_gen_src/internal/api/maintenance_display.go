package api

import (
	"math"
	"sort"
	"strings"
	"time"

	"opsone/internal/store"
)

func activeRoutingProviders(routing map[string]float64) []string {
	out := make([]string, 0, len(routing))
	for provider, pct := range routing {
		if pct > 0 {
			out = append(out, provider)
		}
	}
	sort.Strings(out)
	return out
}

// maintenanceOverview builds the dashboard maintenance chip.
// All providers with active maintenance windows are included regardless of routing %.
// scope_level is true when ALL providers in routing are under maintenance.
func maintenanceOverview(
	mws []store.MaintenanceRow,
	routing map[string]float64,
	sku string,
	now time.Time,
) map[string]any {
	if len(mws) == 0 {
		return nil
	}

	// Total provider count (for scope_level determination).
	totalProviders := len(routing)

	relevant := make([]store.MaintenanceRow, 0, len(mws))
	maintained := make([]string, 0)
	seen := make(map[string]struct{})
	for _, m := range mws {
		if m.EndsAt.Before(now) || m.StartsAt.After(now) {
			continue
		}
		relevant = append(relevant, m)
		if _, dup := seen[m.ProviderCode]; dup {
			continue
		}
		seen[m.ProviderCode] = struct{}{}
		maintained = append(maintained, m.ProviderCode)
	}
	if len(relevant) == 0 {
		return nil
	}
	sort.Strings(maintained)

	mw := relevant[0]
	remaining := int(math.Ceil(mw.EndsAt.Sub(now).Minutes()))
	if remaining < 1 {
		remaining = 1
	}
	for _, m := range relevant[1:] {
		r := int(math.Ceil(m.EndsAt.Sub(now).Minutes()))
		if r < 1 {
			r = 1
		}
		if r < remaining {
			remaining = r
			mw = m
		}
	}

	scopeLevel := totalProviders > 0 && len(maintained) >= totalProviders
	providerCode := ""
	switch {
	case scopeLevel:
		providerCode = sku
	case len(maintained) == 1:
		providerCode = maintained[0]
	default:
		providerCode = strings.Join(maintained, " + ")
	}

	maintIDs := make([]string, 0, len(relevant))
	seenID := map[string]struct{}{}
	for _, m := range relevant {
		if _, dup := seenID[m.MaintenanceID]; dup {
			continue
		}
		seenID[m.MaintenanceID] = struct{}{}
		maintIDs = append(maintIDs, m.MaintenanceID)
	}

	out := map[string]any{
		"maintenance_id":  mw.MaintenanceID,
		"maintenance_ids": maintIDs,
		"provider_code":   providerCode,
		"provider_codes":  maintained,
		"scope_level":     scopeLevel,
		"starts_at":       mw.StartsAt,
		"ends_at":         mw.EndsAt,
		"remaining_min":   remaining,
		"label_vi":        strings.Join(maintained, ", ") + " · " + mw.StartsAt.Format("15:04") + " → " + mw.EndsAt.Format("15:04"),
	}
	if mw.Reason.Valid {
		out["reason"] = mw.Reason.String
	}
	return out
}

func maintainedActiveProviders(mws []store.MaintenanceRow, routing map[string]float64, now time.Time) []string {
	activeSet := make(map[string]struct{}, len(routing))
	for provider, pct := range routing {
		if pct > 0 {
			activeSet[provider] = struct{}{}
		}
	}
	out := make([]string, 0)
	seen := make(map[string]struct{})
	for _, m := range mws {
		if m.EndsAt.Before(now) || m.StartsAt.After(now) {
			continue
		}
		if _, ok := activeSet[m.ProviderCode]; !ok {
			continue
		}
		if _, dup := seen[m.ProviderCode]; dup {
			continue
		}
		seen[m.ProviderCode] = struct{}{}
		out = append(out, m.ProviderCode)
	}
	sort.Strings(out)
	return out
}
