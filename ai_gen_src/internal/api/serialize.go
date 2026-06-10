package api

import (
	"encoding/json"
	"math"

	"opsone/internal/domain"
	"opsone/internal/store"
)

func roundPlanJSON(raw json.RawMessage) map[string]any {
	var plan map[string]any
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil
	}
	roundPctMap(plan, "current_pct")
	roundPctMap(plan, "proposed_pct")
	return plan
}

func roundPctMap(plan map[string]any, key string) {
	m, ok := plan[key].(map[string]any)
	if !ok {
		return
	}
	for k, v := range m {
		if f, ok := v.(float64); ok {
			m[k] = math.Round(f*10) / 10
		}
	}
}

func incidentJSON(r store.IncidentRow) map[string]any {
	m := map[string]any{
		"id":            r.ID,
		"incident_id":   r.IncidentID,
		"severity":      r.Severity,
		"product_code":  r.ProductCode,
		"provider_code": r.ProviderCode,
		"sku_code":      r.SKUCode,
		"status":        r.Status,
		"created_at":    r.CreatedAt,
	}
	if r.CycleID.Valid {
		m["cycle_id"] = r.CycleID.Int64
	}
	if r.Summary.Valid {
		m["summary"] = r.Summary.String
	}
	if r.HandledBy.Valid {
		m["handled_by"] = r.HandledBy.String
	}
	if r.HandledAt.Valid {
		m["handled_at"] = r.HandledAt.Time
	}
	if r.ResolutionAction.Valid {
		m["resolution_action"] = r.ResolutionAction.String
	}
	return m
}

func routingPlanJSON(r store.RoutingPlanRow) map[string]any {
	m := map[string]any{
		"id":           r.ID,
		"product_code": r.ProductCode,
		"scope":        r.Scope,
		"sku_code":     r.SKUCode,
		"status":       r.Status,
		"created_at":   r.CreatedAt,
	}
	if r.CycleID.Valid {
		m["cycle_id"] = r.CycleID.Int64
	}
	if len(r.PlanJSON) > 0 {
		m["plan"] = roundPlanJSON(r.PlanJSON)
	}
	return m
}

func agentChangeJSON(r store.AgentChangeListItem) map[string]any {
	return map[string]any{
		"id":             r.ID,
		"product_code":   r.ProductCode,
		"scope":          r.Scope,
		"sku_code":       r.SKUCode,
		"trigger_type":   r.TriggerType,
		"change_status":  r.ChangeStatus,
		"routing_before": parseRoutingMap(r.RoutingBefore),
		"routing_after":  parseRoutingMap(r.RoutingAfter),
		"created_at":     r.CreatedAt,
	}
}

func parseRoutingMap(raw json.RawMessage) map[string]float64 {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]float64
	_ = json.Unmarshal(raw, &m)
	return m
}

func maintenanceJSON(r store.MaintenanceRow) map[string]any {
	m := map[string]any{
		"maintenance_id": r.MaintenanceID,
		"product_code":   r.ProductCode,
		"provider_code":  r.ProviderCode,
		"sku_code":       r.SKUCode,
		"starts_at":      r.StartsAt,
		"ends_at":        r.EndsAt,
		"status":         r.Status,
	}
	if r.Reason.Valid {
		m["reason"] = r.Reason.String
	}
	return m
}

func notificationJSON(r store.NotificationRow) map[string]any {
	m := map[string]any{
		"id":            r.ID,
		"dedupe_key":    r.DedupeKey,
		"product_code":  r.ProductCode,
		"provider_code": r.ProviderCode,
		"subject":       r.Subject,
		"status":        r.Status,
		"created_at":    r.CreatedAt,
	}
	if r.SentAt.Valid {
		m["sent_at"] = r.SentAt.Time
	}
	return m
}

func productJSON(p domain.Product) map[string]any {
	return map[string]any{
		"id":           p.ID,
		"product_code": p.ProductCode,
		"label":        p.Label,
		"service_type": p.ServiceType,
		"routing_mode": p.RoutingMode,
		"enabled":      p.Enabled,
	}
}

func thresholdJSON(t store.ProductThreshold) map[string]any {
	return map[string]any{
		"product_code":                t.ProductCode,
		"enabled":                     t.Enabled,
		"success_rate_min_pct":        t.SuccessRateMinPct,
		"pending_rate_max_pct":        t.PendingRateMaxPct,
		"fail_rate_max_pct":           t.FailRateMaxPct,
		"fail_txn_count_max":          t.FailTxnCountMax,
		"pending_txn_count_max":       t.PendingTxnCountMax,
		"error_event_count_max":       t.ErrorEventCountMax,
		"metrics_window_min":          t.MetricsWindowMin,
		"consecutive_cycles_required": t.ConsecutiveCyclesRequired,
		"alert_email_enabled":         t.AlertEmailEnabled,
	}
}

func configJSON(s store.AgentSettingsFull) map[string]any {
	return map[string]any{
		"scheduler_enabled":      s.SchedulerEnabled,
		"scheduler_interval_min": s.SchedulerIntervalMin,
		"data_source":            s.DataSource,
		"mock_enabled":           s.MockEnabled,
		"mock_interval_min":      s.MockIntervalMin,
		"mock_scenario":          s.MockScenario,
		"agent_locale":           s.AgentLocale,
	}
}
