package api

import (
	"context"
	"fmt"
	"time"

	"opsone/internal/agent"
	"opsone/internal/domain"
	"opsone/internal/store"
	"opsone/internal/tools"
)

// applyScopeReopenRouting restores scope routing using baseline biz + cycle metrics (§8.6.3 / §9.0).
// Sets recovery_apply_cycle_id so poll auto-apply waits for the next agent cycle.
func (s *Server) applyScopeReopenRouting(
	ctx context.Context,
	product, sku, by, reason string,
) (tools.UpdateRoutingOutput, map[string]float64, error) {
	rows, err := s.DB.GetRoutingForScope(ctx, product, sku)
	if err != nil {
		return tools.UpdateRoutingOutput{}, nil, err
	}
	if len(rows) == 0 {
		return tools.UpdateRoutingOutput{}, nil, fmt.Errorf("Không tìm thấy routing cho scope")
	}
	baseline := make(map[string]float64, len(rows))
	current := make(map[string]float64, len(rows))
	for _, row := range rows {
		baseline[row.ProviderCode] = row.BaselinePct
		current[row.ProviderCode] = row.TrafficPct
	}

	th, err := s.DB.GetProductThreshold(ctx, product)
	if err != nil {
		return tools.UpdateRoutingOutput{}, nil, err
	}
	pc, err := s.buildProductContextForReopen(ctx, product, sku, current, th)
	if err != nil {
		return tools.UpdateRoutingOutput{}, nil, err
	}
	if reason == "" {
		reason = fmt.Sprintf("Mở lại scope %s/%s — baseline biz (metric chu kỳ Agent)", product, sku)
	}
	_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
	plan := agent.BuildReopenRoutingPlan(pc, baseline, th, reason)

	prod, err := s.DB.GetProductByCode(ctx, product)
	if err != nil {
		return tools.UpdateRoutingOutput{}, nil, err
	}
	scope := "sku"
	if sku == "" {
		scope = "provider"
	}
	out, err := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     product,
		ServiceType: string(prod.ServiceType),
		Scope:       scope,
		SKU:         sku,
		Routing:     plan.Proposed,
		TriggerType: "manual_baseline",
		ExecutedBy:  by,
		Reason:      plan.Reason,
	})
	if err != nil {
		return tools.UpdateRoutingOutput{}, nil, err
	}
	applyCycle, _ := s.DB.LatestCompletedCycleID(ctx)
	if applyCycle > 0 {
		_ = s.DB.MarkRecoveryStart(ctx, product, sku, applyCycle)
	}
	_ = s.DB.SetPendingRestore(ctx, product, sku, false, by)
	return out, plan.Proposed, nil
}

func (s *Server) buildProductContextForReopen(
	ctx context.Context,
	product, sku string,
	current map[string]float64,
	th store.ProductThreshold,
) (agent.ProductContext, error) {
	reg := tools.NewRegistry(s.DB)
	routing, err := reg.GetRouting(ctx, tools.GetRoutingInput{Product: product})
	if err != nil {
		return agent.ProductContext{}, err
	}
	prov, err := reg.GetProviders(ctx, tools.GetProvidersInput{Product: product})
	if err != nil {
		return agent.ProductContext{}, err
	}
	settings, err := s.DB.GetAgentSettings(ctx)
	if err != nil {
		return agent.ProductContext{}, err
	}
	latestCycle, _ := s.DB.LatestCompletedCycleID(ctx)
	windowMin := th.MetricsWindowMin
	if windowMin <= 0 {
		windowMin = 15
	}
	since := time.Now().Add(-time.Duration(windowMin) * time.Minute)

	var scopes []agent.ScopeContext
	for provider := range current {
		sc := agent.ScopeContext{
			ProductCode:  product,
			SKUCode:      sku,
			ProviderCode: provider,
		}
		if latestCycle > 0 {
			hist, err := s.DB.GetRecentScopeHistory(ctx, product, sku, provider, latestCycle+1, 1)
			if err == nil && len(hist) > 0 {
				p := hist[0]
				sc.Metrics = historyPointToMetrics(p)
				scopes = append(scopes, sc)
				continue
			}
		}
		m, ok, err := s.DB.GetMetricsInWindow(ctx, settings.DataSource, product, sku, provider, since)
		if err != nil || !ok {
			scopes = append(scopes, sc)
			continue
		}
		sc.Metrics = &tools.GetMetricsOutput{
			SuccessRate:       m.SuccessRate,
			PendingRate:       m.PendingRate,
			FailRate:          m.FailRate,
			TotalTransactions: m.TotalTransactions,
		}
		scopes = append(scopes, sc)
	}

	return agent.ProductContext{
		Product:   domain.Product{ProductCode: product},
		Providers: prov,
		Routing:   routing,
		Scopes:    scopes,
	}, nil
}

func historyPointToMetrics(p store.ScopeHistoryPoint) *tools.GetMetricsOutput {
	return &tools.GetMetricsOutput{
		SuccessRate:       p.SuccessRate,
		PendingRate:       p.PendingRate,
		FailRate:          p.FailRate,
		TotalTransactions: p.TotalTransactions,
	}
}

func (s *Server) inReopenRecoveryGrace(ctx context.Context, product, sku string, currentCycleID uint64) bool {
	applyCycle, ok, err := s.DB.GetRecoveryApplyCycle(ctx, product, sku)
	if err != nil || !ok || applyCycle == 0 {
		return false
	}
	return currentCycleID <= applyCycle
}