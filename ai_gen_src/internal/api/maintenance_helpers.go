package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"opsone/internal/domain"
	"opsone/internal/store"
	"opsone/internal/tools"
)

const skuWideMaintenanceMarker = "Tất cả provider đang routing"

func isSkuWideMaintenanceReason(reason string) bool {
	return strings.Contains(reason, skuWideMaintenanceMarker)
}

func maintenanceTargetsForScope(reason, providerCode string, routing []domain.RoutingPct) []string {
	if isSkuWideMaintenanceReason(reason) {
		var out []string
		for _, row := range routing {
			if row.TrafficPct > 0 {
				out = append(out, row.ProviderCode)
			}
		}
		return out
	}
	if providerCode != "" {
		return []string{providerCode}
	}
	return nil
}

func maintenanceTargetsFromRecommendation(rec store.PendingRecommendation, routing []domain.RoutingPct) []string {
	return maintenanceTargetsForScope(rec.Detail, rec.ProviderCode, routing)
}

func pendingMaintenanceToMap(rec store.PendingRecommendation) map[string]any {
	if rec.ID == 0 || strings.Contains(rec.Detail, "DISMISSED:") {
		return nil
	}
	out := map[string]any{
		"id":          rec.ID,
		"sku_code":    rec.SKUCode,
		"reason":      rec.Detail,
		"action_type": "maintenance",
	}
	if isSkuWideMaintenanceReason(rec.Detail) {
		out["scope_level"] = true
	} else if rec.ProviderCode != "" {
		out["provider_code"] = rec.ProviderCode
	}
	return out
}

func validateMaintenanceWindow(startsAt, endsAt time.Time) error {
	if !endsAt.After(startsAt) {
		return fmt.Errorf("ends_at phải sau starts_at")
	}
	return nil
}

func parseMaintenanceWindow(startsAtRaw, endsAtRaw string, durationMin int) (time.Time, time.Time, error) {
	if startsAtRaw != "" && endsAtRaw != "" {
		startsAt, err := parseFlexibleTime(startsAtRaw)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("starts_at không hợp lệ")
		}
		endsAt, err := parseFlexibleTime(endsAtRaw)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("ends_at không hợp lệ")
		}
		if err := validateMaintenanceWindow(startsAt, endsAt); err != nil {
			return time.Time{}, time.Time{}, err
		}
		return startsAt, endsAt, nil
	}
	if durationMin <= 0 {
		durationMin = 60
	}
	startsAt := time.Now()
	endsAt := startsAt.Add(time.Duration(durationMin) * time.Minute)
	if err := validateMaintenanceWindow(startsAt, endsAt); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return startsAt, endsAt, nil
}

func applyMaintenanceTargets(
	ctx context.Context,
	toolsReg *tools.Registry,
	product, sku, by, reason string,
	targets []string,
	startsAt, endsAt time.Time,
) ([]map[string]any, error) {
	var applied []map[string]any
	for i, provider := range targets {
		out, err := toolsReg.SetMaintenance(ctx, tools.SetMaintenanceInput{
			Product:     product,
			Provider:    provider,
			SKU:         sku,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
			TriggerType: "admin_manual",
			Reason:      reason,
			Status:      "active",
			Seq:         i,
		})
		if err != nil {
			for _, prev := range applied {
				mid, _ := prev["maintenance_id"].(string)
				if mid != "" {
					_ = toolsReg.DB.CancelMaintenanceByID(ctx, mid, by)
				}
			}
			return nil, err
		}
		applied = append(applied, map[string]any{
			"maintenance_id": out.MaintenanceID,
			"provider_code":  provider,
			"status":         out.Status,
		})
	}
	return applied, nil
}

func parseFlexibleTime(raw string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q", raw)
}
