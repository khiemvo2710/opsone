package api

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"opsone/internal/agent"
	"opsone/internal/notify"
	"opsone/internal/output"
	"opsone/internal/store"
	"opsone/internal/threshold"
	"opsone/internal/tools"
)

type overviewCaches struct {
	settings         store.AgentSettings
	thresholds       map[string]store.ProductThreshold
	metrics          map[string]store.MetricWindowResult
	sinceByWindow    map[int]time.Time
	routingByProd    map[string]tools.GetRoutingOutput
	providersByProd  map[string]tools.GetProvidersOutput
	latestCycleID    uint64
	historyByScope   map[string][]store.ScopeHistoryPoint
	notify           *notify.Service
}

type scopeSnapshot struct {
	ProviderMetrics     map[string]any
	LiveMetrics         map[string]any
	Health              string
	AnyBreached         bool
	ShouldAct           bool
	ConsecutiveBreaches int
	Worst               agent.ScopeContext
	HasWorst            bool
	ScopeContexts       []agent.ScopeContext
}

func (s *Server) newOverviewCaches(ctx context.Context) (overviewCaches, error) {
	settings, err := s.DB.GetAgentSettings(ctx)
	if err != nil {
		return overviewCaches{}, err
	}
	thresholds, err := s.DB.ListAllProductThresholds(ctx)
	if err != nil {
		return overviewCaches{}, err
	}
	windowMin := 15
	for _, th := range thresholds {
		if th.MetricsWindowMin > windowMin {
			windowMin = th.MetricsWindowMin
		}
	}
	since := time.Now().Add(-time.Duration(windowMin) * time.Minute)
	metrics, err := s.DB.LoadLatestMetricsSince(ctx, settings.DataSource, since)
	if err != nil {
		return overviewCaches{}, err
	}
	maxHist := 4
	for _, th := range thresholds {
		if need := th.ConsecutiveCyclesRequired + 2; need > maxHist {
			maxHist = need
		}
	}
	latestCycle, _ := s.DB.LatestCompletedCycleID(ctx)
	historyByScope := map[string][]store.ScopeHistoryPoint{}
	if latestCycle > 0 {
		if h, err := s.DB.LoadAgentHistoryThroughCycle(ctx, latestCycle, maxHist); err == nil {
			historyByScope = h
		}
	}
	return overviewCaches{
		settings:        settings,
		thresholds:      thresholds,
		metrics:         metrics,
		sinceByWindow:   map[int]time.Time{windowMin: since},
		routingByProd:   make(map[string]tools.GetRoutingOutput),
		providersByProd: make(map[string]tools.GetProvidersOutput),
		latestCycleID:   latestCycle,
		historyByScope:  historyByScope,
		notify:          s.Notify,
	}, nil
}

func (c *overviewCaches) metric(product, sku, provider string) (store.MetricWindowResult, bool) {
	m, ok := c.metrics[product+"\x00"+sku+"\x00"+provider]
	return m, ok
}

func (c *overviewCaches) productRouting(ctx context.Context, db *store.DB, product string) (tools.GetRoutingOutput, tools.GetProvidersOutput, error) {
	if r, ok := c.routingByProd[product]; ok {
		return r, c.providersByProd[product], nil
	}
	reg := tools.NewRegistry(db, c.notify)
	routing, err := reg.GetRouting(ctx, tools.GetRoutingInput{Product: product})
	if err != nil {
		return tools.GetRoutingOutput{}, tools.GetProvidersOutput{}, err
	}
	prov, err := reg.GetProviders(ctx, tools.GetProvidersInput{Product: product})
	if err != nil {
		return tools.GetRoutingOutput{}, tools.GetProvidersOutput{}, err
	}
	c.routingByProd[product] = routing
	c.providersByProd[product] = prov
	return routing, prov, nil
}

func (c *overviewCaches) since(windowMin int) time.Time {
	if windowMin <= 0 {
		windowMin = 15
	}
	if t, ok := c.sinceByWindow[windowMin]; ok {
		return t
	}
	t := time.Now().Add(-time.Duration(windowMin) * time.Minute)
	c.sinceByWindow[windowMin] = t
	return t
}

func (c *overviewCaches) threshold(product string) (store.ProductThreshold, bool) {
	th, ok := c.thresholds[product]
	return th, ok
}

func (s *Server) buildScopeSnapshot(
	ctx context.Context,
	c overviewCaches,
	product, sku string,
	providers map[string]float64,
	maintainedProviders []string,
) scopeSnapshot {
	snap := scopeSnapshot{
		ProviderMetrics: make(map[string]any, len(providers)),
		Health:          "green",
	}

	th, ok := c.threshold(product)
	if !ok || !th.Enabled {
		for provider, pct := range providers {
			snap.ProviderMetrics[provider] = map[string]any{"routing_pct": roundPct(pct)}
		}
		return snap
	}

	maintainedSet := maintainedProviderSet(maintainedProviders)
	activeRouting := 0
	for _, pct := range providers {
		if pct > 0 {
			activeRouting++
		}
	}
	skuWideMaint := activeRouting > 0 && len(maintainedProviders) >= activeRouting

	var totalW, wSucc, wPend, wFail float64
	var pendingTxn, failTxn uint
	worstScore := -1.0

	for provider, pct := range providers {
		underMaint := skuWideMaint || maintainedSet[provider]
		item := map[string]any{"routing_pct": roundPct(pct)}
		if pct <= 0 || underMaint {
			if underMaint {
				item["routing_pct"] = 0.0
				item["under_maintenance"] = true
			}
			item["success_pct"] = 0.0
			item["pending_pct"] = 0.0
			item["fail_pct"] = 0.0
			item["pending_txn"] = 0
			item["fail_txn"] = 0
			snap.ProviderMetrics[provider] = item
			continue
		}

		m, hasMetric := c.metric(product, sku, provider)
		if !hasMetric {
			snap.ProviderMetrics[provider] = item
			continue
		}

		pTxn := uint(math.Round(float64(m.TotalTransactions) * m.PendingRate / 100))
		fTxn := uint(math.Round(float64(m.TotalTransactions) * m.FailRate / 100))
		item["success_pct"] = roundPct(m.SuccessRate)
		item["pending_pct"] = roundPct(m.PendingRate)
		item["fail_pct"] = roundPct(m.FailRate)
		item["pending_txn"] = pTxn
		item["fail_txn"] = fTxn
		snap.ProviderMetrics[provider] = item

		totalW += pct
		wSucc += pct * m.SuccessRate
		wPend += pct * m.PendingRate
		wFail += pct * m.FailRate
		pendingTxn += pTxn
		failTxn += fTxn

		breached := store.ScopeBreachedFromSnapshot(m.SuccessRate, m.PendingRate, m.FailRate, pTxn, fTxn, th)
		sc := agent.ScopeContext{
			ProductCode:  product,
			SKUCode:      sku,
			ProviderCode: provider,
			Metrics: &tools.GetMetricsOutput{
				SuccessRate:       m.SuccessRate,
				PendingRate:       m.PendingRate,
				FailRate:          m.FailRate,
				TotalTransactions: m.TotalTransactions,
			},
		}
		if breached {
			snap.AnyBreached = true
			sc.State = "WARNING"
			reasons := store.SnapshotBreachReasons(m.SuccessRate, m.PendingRate, m.FailRate, pTxn, fTxn, th)
			sc.Threshold = &threshold.Result{
				ProductCode:   product,
				Breached:      true,
				BreachReasons: reasons,
			}
			score := m.FailRate*2 + m.PendingRate + float64(pTxn) + float64(fTxn)*0.5
			if score > worstScore {
				worstScore = score
				snap.Worst = sc
				snap.HasWorst = true
			}
		} else {
			sc.Threshold = &threshold.Result{ProductCode: product, Breached: false}
		}
		snap.ScopeContexts = append(snap.ScopeContexts, sc)
	}

	if snap.AnyBreached {
		required := th.ConsecutiveCyclesRequired
		if required <= 0 {
			required = 2
		}
		maxConsec := 0
		shouldAct := false
		for _, sc := range snap.ScopeContexts {
			if sc.Threshold == nil || !sc.Threshold.Breached {
				continue
			}
			consec, act := store.ScopeConsecutiveBreachFromHistory(
				product, sku, sc.ProviderCode, th, true, c.latestCycleID, c.historyByScope,
			)
			if act {
				shouldAct = true
			}
			if consec > maxConsec {
				maxConsec = consec
			}
		}
		snap.ConsecutiveBreaches = maxConsec
		if !shouldAct && snap.HasWorst {
			pc := agent.ProductContext{
				Scopes:              snap.ScopeContexts,
				MaintainedProviders: maintainedProviderSet(maintainedProviders),
			}
			if agent.ShouldForceAutoMaintenance(pc, sku, providers, th) ||
				agent.ShouldForceAutoMaintenanceAllProviders(pc, sku, providers, th) {
				shouldAct = true
			}
		}
		snap.ShouldAct = shouldAct
		if snap.HasWorst {
			if snap.Worst.Threshold != nil {
				snap.Worst.Threshold.ConsecutiveBreachCycles = maxConsec
				snap.Worst.Threshold.RequiredCycles = required
				snap.Worst.Threshold.ShouldAct = shouldAct
			}
			if shouldAct {
				snap.Health = "red"
				snap.Worst.State = "INCIDENT"
			} else {
				snap.Health = "yellow"
				snap.Worst.State = "WARNING"
			}
			pc := agent.ProductContext{
				Scopes:              snap.ScopeContexts,
				MaintainedProviders: maintainedProviderSet(maintainedProviders),
			}
			action, reason := agent.SKURoutingDecision(pc, sku, providers, th)
			if snap.Worst.Threshold != nil {
				snap.Worst.Threshold.SuggestedAction = action
				snap.Worst.Threshold.SuggestedActionReason = reason
			}
		} else {
			snap.Health = "yellow"
		}
	}
	if totalW > 0 {
		snap.LiveMetrics = map[string]any{
			"success_pct": roundPct(wSucc / totalW),
			"pending_pct": roundPct(wPend / totalW),
			"fail_pct":    roundPct(wFail / totalW),
			"pending_txn": pendingTxn,
			"fail_txn":    failTxn,
		}
	}
	return snap
}

// liveMaintenanceReason rebuilds suggestion text from current snapshot metrics (khớp cột provider trên UI).
func liveMaintenanceReason(product, sku string, snap scopeSnapshot, pm map[string]any, th store.ProductThreshold) string {
	if snap.HasWorst && snap.Worst.Threshold != nil {
		if snap.Worst.Threshold.SuggestedAction == "maintenance" {
			if r := snap.Worst.Threshold.SuggestedActionReason; strings.Contains(r, "Tất cả provider") {
				return r
			}
		}
	}
	provider, _ := pm["provider_code"].(string)
	if provider == "" {
		pc := agent.ProductContext{Scopes: snap.ScopeContexts}
		if kind, reason := agent.SKURoutingDecision(pc, sku, providersFromSnapshotForReason(snap), th); kind == "maintenance" && reason != "" {
			return reason
		}
		if raw, ok := pm["reason"].(string); ok && !strings.Contains(raw, "DISMISSED:") {
			return raw
		}
		return ""
	}
	if item, ok := snap.ProviderMetrics[provider].(map[string]any); ok {
		success, _ := item["success_pct"].(float64)
		pending, _ := item["pending_pct"].(float64)
		fail, _ := item["fail_pct"].(float64)
		pTxnF, _ := item["pending_txn"].(uint)
		fTxnF, _ := item["fail_txn"].(uint)
		pTxn := uint(pTxnF)
		fTxn := uint(fTxnF)
		if pTxnF == 0 {
			if v, ok := item["pending_txn"].(float64); ok {
				pTxn = uint(math.Round(v))
			}
		}
		if fTxnF == 0 {
			if v, ok := item["fail_txn"].(float64); ok {
				fTxn = uint(math.Round(v))
			}
		}
		reasons := store.SnapshotBreachReasons(success, pending, fail, pTxn, fTxn, th)
		if len(reasons) > 0 {
			return output.MaintenanceDetail(product, provider, reasons, nil)
		}
	}
	if raw, ok := pm["reason"].(string); ok && !strings.Contains(raw, "DISMISSED:") {
		return raw
	}
	return fmt.Sprintf("Đề xuất bảo trì %s — provider %s", product, provider)
}

func providersFromSnapshotForReason(snap scopeSnapshot) map[string]float64 {
	out := make(map[string]float64, len(snap.ProviderMetrics))
	for provider, raw := range snap.ProviderMetrics {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if v, ok := m["routing_pct"].(float64); ok {
			out[provider] = v
		}
	}
	return out
}

func thresholdsJSON(thresholds map[string]store.ProductThreshold) map[string]any {
	out := make(map[string]any, len(thresholds))
	for code, th := range thresholds {
		out[code] = map[string]any{
			"product_code":               th.ProductCode,
			"enabled":                    th.Enabled,
			"success_rate_min_pct":       th.SuccessRateMinPct,
			"pending_rate_max_pct":       th.PendingRateMaxPct,
			"fail_rate_max_pct":          th.FailRateMaxPct,
			"fail_txn_count_max":         th.FailTxnCountMax,
			"pending_txn_count_max":      th.PendingTxnCountMax,
			"error_event_count_max":      th.ErrorEventCountMax,
			"metrics_window_min":         th.MetricsWindowMin,
			"consecutive_cycles_required": th.ConsecutiveCyclesRequired,
			"alert_email_enabled":        th.AlertEmailEnabled,
		}
	}
	return out
}
