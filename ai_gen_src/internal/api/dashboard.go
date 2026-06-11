package api

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"opsone/internal/agent"
	"opsone/internal/store"
)

type scopeKey struct {
	product string
	sku     string
}

func (k scopeKey) maintKey() string {
	return store.MaintenanceScopeKey(k.product, k.sku)
}

func (s *Server) handleDashboardOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	routing, err := s.DB.ListAllRoutingConfig(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	meta, _ := s.DB.ProductMetaMap(ctx)

	caches, err := s.newOverviewCaches(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	now := time.Now()
	maintRows, _ := s.DB.ListMaintenanceWindows(ctx, "", "active", 100)
	maintByScope := map[scopeKey][]store.MaintenanceRow{}
	for _, m := range maintRows {
		if m.EndsAt.Before(now) || m.StartsAt.After(now) {
			continue
		}
		k := scopeKey{product: m.ProductCode, sku: m.SKUCode}
		maintByScope[k] = append(maintByScope[k], m)
	}

	planRows, _ := s.DB.ListPendingRoutingPlansPerScope(ctx)
	planByScope := map[scopeKey]store.RoutingPlanRow{}
	for _, p := range planRows {
		planByScope[scopeKey{product: p.ProductCode, sku: p.SKUCode}] = p
	}

	maintRecByScope, _ := s.DB.ListPendingMaintenanceByScope(ctx)
	scopeAutoByKey, _ := s.DB.ListScopeAutoConfig(ctx)

	scopes := map[scopeKey]map[string]float64{}
	baselines := map[scopeKey]map[string]float64{}
	scopeOrder := make([]scopeKey, 0)
	for _, row := range routing {
		k := scopeKey{product: row.ProductCode, sku: row.SKUCode}
		if scopes[k] == nil {
			scopes[k] = make(map[string]float64)
			baselines[k] = make(map[string]float64)
			scopeOrder = append(scopeOrder, k)
		}
		scopes[k][row.ProviderCode] = roundPct(row.TrafficPct)
		baselines[k][row.ProviderCode] = roundPct(row.BaselinePct)
	}

	outRows := make([]map[string]any, 0, len(scopeOrder))
	for _, k := range scopeOrder {
		var maintainedProviders []string
		if mws, ok := maintByScope[k]; ok {
			maintainedProviders = maintainedActiveProviders(mws, scopes[k], now)
		}

		snap := s.buildScopeSnapshot(ctx, caches, k.product, k.sku, scopes[k], maintainedProviders)

		m := meta[k.product]
		label := m.Label
		if label == "" {
			label = k.product
		}
		svcType := m.ServiceType
		item := map[string]any{
			"product_code":     k.product,
			"product_label":    label,
			"service_type":     svcType,
			"sku_code":         k.sku,
			"health_status":    snap.Health,
			"routing_pct":      scopes[k],
			"baseline_pct":     baselines[k],
			"provider_metrics": snap.ProviderMetrics,
		}
		if snap.LiveMetrics != nil {
			item["live_metrics"] = snap.LiveMetrics
		}

		productAutoKey := store.ScopeAutoMapKey(k.product, "")
		if ac, ok := scopeAutoByKey[productAutoKey]; ok {
			item["product_auto_action"] = ac.AutoAction
			if ac.WindowStart != "" {
				item["product_window_start"] = ac.WindowStart
			}
			if ac.WindowEnd != "" {
				item["product_window_end"] = ac.WindowEnd
			}
		} else {
			item["product_auto_action"] = "recommend_only"
		}

		if ac, ok := scopeAutoByKey[store.ScopeAutoMapKey(k.product, k.sku)]; ok {
			item["scope_auto_action"] = ac.AutoAction
			if ac.WindowStart != "" {
				item["scope_window_start"] = ac.WindowStart
			}
			if ac.WindowEnd != "" {
				item["scope_window_end"] = ac.WindowEnd
			}
		} else {
			item["scope_auto_action"] = "recommend_only"
		}

		effectiveAuto := store.ResolveEffectiveScopeAuto(scopeAutoByKey, k.product, k.sku)
		item["auto_action"] = effectiveAuto.AutoAction
		if effectiveAuto.WindowStart != "" {
			item["window_start"] = effectiveAuto.WindowStart
		}
		if effectiveAuto.WindowEnd != "" {
			item["window_end"] = effectiveAuto.WindowEnd
		}

		if mws, ok := maintByScope[k]; ok && len(mws) > 0 {
			if maint := maintenanceOverview(mws, scopes[k], k.sku, now); maint != nil {
				item["maintenance"] = maint
			}
		}

		th, hasTh := caches.threshold(k.product)
		dbMaint := pendingMaintenanceToMap(maintRecByScope[k.maintKey()])

		_, hasPendingPlan := planByScope[k]
		var livePlan, liveMaint map[string]any
		maintWarranted := false
		if snap.ShouldAct && hasTh {
			livePlan, liveMaint = s.scopeSuggestionFromSnapshot(ctx, caches, k.product, k.sku, snap, maintainedProviders)
			maintWarranted = maintenanceWarrantedFromSnapshot(snap, k.sku, scopes[k], maintainedProviders, th)
			if maintWarranted {
				livePlan = nil
				if hasPendingPlan {
					_ = s.DB.CancelPendingRoutingPlansForScope(ctx, k.product, k.sku)
					hasPendingPlan = false
				}
			}
		}
		inActiveMaint := item["maintenance"] != nil
		if !inActiveMaint && hasTh && hasPendingPlan && livePlan == nil && snap.AnyBreached && !maintWarranted {
			livePlan = s.refreshPendingRoutingFromSnapshot(ctx, caches, k.product, k.sku, snap, maintainedProviders)
		}
		if !inActiveMaint && hasPendingPlan && !snap.AnyBreached && !snap.ShouldAct {
			_ = s.DB.CancelPendingRoutingPlansForScope(ctx, k.product, k.sku)
			hasPendingPlan = false
		}

		if !inActiveMaint && hasTh {
			if pm, ok := s.prioritizeMaintenanceSuggestion(ctx, caches, k.product, k.sku, snap, maintainedProviders, liveMaint); ok {
				liveMaint = pm
				livePlan = nil
				maintWarranted = true
				hasPendingPlan = false
			}
		}

		manualApproval := !store.ShouldAutoApplyScope(effectiveAuto, now)

		if !manualApproval && hasTh && !inActiveMaint && snap.AnyBreached {
			inGrace := s.inReopenRecoveryGrace(ctx, k.product, k.sku, caches.latestCycleID)
			for pass := 0; pass < 2; pass++ {
				if !scopeAutoApplyAllowed(snap, k.sku, scopes[k], maintainedProviders, th) {
					break
				}
				applied, err := s.autoApplyScopeFromSnapshot(ctx, caches, k.product, k.sku, snap, maintainedProviders, inGrace)
				if err != nil || !applied {
					break
				}
				if rows, err := s.DB.GetRoutingForScope(ctx, k.product, k.sku); err == nil {
					updated := make(map[string]float64, len(rows))
					for _, row := range rows {
						updated[row.ProviderCode] = roundPct(row.TrafficPct)
					}
					scopes[k] = updated
					item["routing_pct"] = updated
				}
				hasPendingPlan = false
				livePlan = nil
				liveMaint = nil
				maintWarranted = false
				if mws, ok := maintByScope[k]; ok {
					maintainedProviders = maintainedActiveProviders(mws, scopes[k], now)
					if maint := maintenanceOverview(mws, scopes[k], k.sku, now); maint != nil {
						item["maintenance"] = maint
						inActiveMaint = true
						break
					}
				}
				snap = s.buildScopeSnapshot(ctx, caches, k.product, k.sku, scopes[k], maintainedProviders)
				item["health_status"] = snap.Health
				item["provider_metrics"] = snap.ProviderMetrics
			}
		}

		if manualApproval {
			switch {
			case !inActiveMaint && maintWarranted && liveMaint != nil:
				liveMaint["reason"] = liveMaintenanceReason(k.product, k.sku, snap, liveMaint, th)
				item["pending_maintenance"] = liveMaint
			case !inActiveMaint && maintWarranted && dbMaint != nil:
				dbMaint["reason"] = liveMaintenanceReason(k.product, k.sku, snap, dbMaint, th)
				item["pending_maintenance"] = dbMaint
			case livePlan != nil && !inActiveMaint:
				item["pending_plan"] = livePlan
			default:
				if hasPendingPlan && !maintWarranted {
					if plan, ok := planByScope[k]; ok {
						var planJSON agent.RoutingPlanJSON
						if err := json.Unmarshal(plan.PlanJSON, &planJSON); err != nil ||
							!s.routingProposalSuppressedAfterReject(ctx, k.product, k.sku, planJSON) {
							item["pending_plan"] = routingPlanJSON(plan)
						}
					}
				}
			}
		}

		outRows = append(outRows, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"updated_at":  time.Now(),
		"providers":   []string{"ESALE", "IMEDIA", "SHOPPAY"},
		"thresholds":  thresholdsJSON(caches.thresholds),
		"rows":        outRows,
	})
}

func roundPct(v float64) float64 {
	return math.Round(v*10) / 10
}
