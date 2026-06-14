package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"opsone/internal/agent"
	"opsone/internal/domain"
	"opsone/internal/output"
	"opsone/internal/store"
	"opsone/internal/threshold"
	"opsone/internal/tools"
)

type scopeAction struct {
	kind        string // routing | maintenance
	plan        agent.RoutingPlanJSON
	worst       agent.ScopeContext
	eval        threshold.Result
}

// ensureScopeSuggestion persists a routing plan or maintenance recommendation when any
// active provider breaches 1 of 5 snapshot metrics (§9.0 — per provider, không gom scope).
func (s *Server) ensureScopeSuggestion(
	ctx context.Context,
	product, sku string,
	providers map[string]float64,
	th store.ProductThreshold,
) error {
	action, err := s.resolveScopeAction(ctx, product, sku, providers, th)
	if err != nil || action == nil {
		return err
	}

	hasPlan, err := s.DB.HasPendingRoutingPlan(ctx, product, sku)
	if err != nil {
		return err
	}
	if hasPlan && action.kind == "routing" {
		return s.DB.UpdatePendingRoutingPlanForScope(ctx, nil, product, sku, action.plan)
	}

	switch action.kind {
	case "routing":
		hasMaint, err := s.DB.HasPendingMaintenanceRecommendation(ctx, product, sku)
		if err != nil || hasMaint {
			return err
		}
		_, err = s.DB.InsertRoutingPlan(ctx, nil, product, action.plan.Scope, sku, action.plan, "pending_approve")
		return err
	case "maintenance":
		_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
		has, err := s.DB.HasRecentRecommendation(ctx, product, sku, "maintenance")
		if err != nil || has {
			return err
		}
		detail := action.eval.SuggestedActionReason
		if detail == "" {
			detail = output.MaintenanceDetail(product, action.worst.ProviderCode, action.eval.BreachReasons, nil)
		}
		return s.DB.InsertRecommendation(ctx, nil, nil, product, sku, "maintenance", detail)
	default:
		return nil
	}
}

// scopeSuggestionFromSnapshot builds synthetic plan/maintenance from a precomputed scope snapshot (no extra metric queries).
func (s *Server) scopeSuggestionFromSnapshot(
	ctx context.Context,
	caches overviewCaches,
	product, sku string,
	snap scopeSnapshot,
	maintainedProviders []string,
) (pendingPlan map[string]any, pendingMaint map[string]any) {
	action, err := s.resolveScopeActionFromSnapshot(ctx, caches, product, sku, snap, maintainedProviders, true)
	if err != nil || action == nil {
		return nil, nil
	}
	switch action.kind {
	case "routing":
		return s.routingPlanResponse(ctx, product, sku, action.plan), nil
	case "maintenance":
		pm, ok := s.maintenanceMapFromScopeAction(ctx, caches, product, sku, snap, maintainedProviders, action)
		if !ok {
			return nil, nil
		}
		return nil, pm
	default:
		return nil, nil
	}
}

// refreshPendingRoutingFromSnapshot recomputes a pending DB plan from live metrics (no ShouldAct gate).
func (s *Server) refreshPendingRoutingFromSnapshot(
	ctx context.Context,
	caches overviewCaches,
	product, sku string,
	snap scopeSnapshot,
	maintainedProviders []string,
) map[string]any {
	if !snap.AnyBreached || !snap.HasWorst {
		return nil
	}
	action, err := s.resolveScopeActionFromSnapshot(ctx, caches, product, sku, snap, maintainedProviders, false)
	if err != nil || action == nil {
		return nil
	}
	if action.kind == "maintenance" {
		_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
		return nil
	}
	return s.routingPlanResponse(ctx, product, sku, action.plan)
}

// prioritizeMaintenanceSuggestion cancels stale routing plans and returns maintenance when every
// routable provider breaches (SKURoutingDecision=maintenance) — không phụ thuộc ShouldAct.
func (s *Server) prioritizeMaintenanceSuggestion(
	ctx context.Context,
	caches overviewCaches,
	product, sku string,
	snap scopeSnapshot,
	maintainedProviders []string,
	existingMaint map[string]any,
) (map[string]any, bool) {
	if existingMaint != nil {
		_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
		return existingMaint, true
	}
	if !snap.AnyBreached || !snap.HasWorst {
		return nil, false
	}
	th, ok := caches.threshold(product)
	if !ok {
		return nil, false
	}
	routing := providersFromSnapshot(snap)
	pc := agent.ProductContext{
		Scopes:              snap.ScopeContexts,
		MaintainedProviders: maintainedProviderSet(maintainedProviders),
	}
	kind, _ := agent.SKURoutingDecision(pc, sku, routing, th)
	if kind != "maintenance" || !maintenanceSuggestionStillNeeded(snap, routing, maintainedProviders, th) {
		return nil, false
	}
	action, err := s.resolveScopeActionFromSnapshot(ctx, caches, product, sku, snap, maintainedProviders, false)
	if err != nil || action == nil || action.kind != "maintenance" {
		return nil, false
	}
	pm, ok := s.maintenanceMapFromScopeAction(ctx, caches, product, sku, snap, maintainedProviders, action)
	if !ok {
		return nil, false
	}
	_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
	return pm, true
}

func (s *Server) maintenanceMapFromScopeAction(
	ctx context.Context,
	caches overviewCaches,
	product, sku string,
	snap scopeSnapshot,
	maintainedProviders []string,
	action *scopeAction,
) (map[string]any, bool) {
	if action == nil || action.kind != "maintenance" {
		return nil, false
	}
	if s.maintenanceSuggestionSuppressedAfterDismiss(ctx, product, sku, caches.latestCycleID) {
		return nil, false
	}
	th, ok := caches.threshold(product)
	if ok && !maintenanceSuggestionStillNeeded(snap, providersFromSnapshot(snap), maintainedProviders, th) {
		return nil, false
	}
	reason := action.eval.SuggestedActionReason
	if reason == "" {
		reason = output.MaintenanceDetail(product, action.worst.ProviderCode, action.eval.BreachReasons, nil)
	}
	pm := map[string]any{
		"reason":      reason,
		"action_type": "maintenance",
		"suggested":   true,
	}
	if isSkuWideMaintenanceReason(reason) {
		pm["scope_level"] = true
	} else {
		pm["provider_code"] = action.worst.ProviderCode
	}
	return pm, true
}

func (s *Server) routingPlanResponse(ctx context.Context, product, sku string, plan agent.RoutingPlanJSON) map[string]any {
	if s.routingProposalSuppressedAfterReject(ctx, product, sku, plan) {
		return nil
	}
	hasPlan, err := s.DB.HasPendingRoutingPlan(ctx, product, sku)
	if err != nil {
		return syntheticPendingPlanMap(product, sku, plan)
	}
	if hasPlan {
		if err := s.DB.UpdatePendingRoutingPlanForScope(ctx, nil, product, sku, plan); err != nil {
			return syntheticPendingPlanMap(product, sku, plan)
		}
		if row, ok, err := s.DB.GetPendingRoutingPlanForScope(ctx, product, sku); err == nil && ok {
			return routingPlanJSON(row)
		}
	} else {
		// No plan in DB yet — insert so chat commands can also see this proposal.
		// (dashboard polls would otherwise return a synthetic id=0 plan invisible to chat)
		hasMaint, mErr := s.DB.HasPendingMaintenanceRecommendation(ctx, product, sku)
		if mErr == nil && !hasMaint {
			if _, iErr := s.DB.InsertRoutingPlan(ctx, nil, product, plan.Scope, sku, plan, "pending_approve"); iErr == nil {
				if row, ok, err := s.DB.GetPendingRoutingPlanForScope(ctx, product, sku); err == nil && ok {
					return routingPlanJSON(row)
				}
			}
		}
	}
	return syntheticPendingPlanMap(product, sku, plan)
}

func (s *Server) resolveScopeActionFromSnapshot(
	ctx context.Context,
	caches overviewCaches,
	product, sku string,
	snap scopeSnapshot,
	maintainedProviders []string,
	requireShouldAct bool,
) (*scopeAction, error) {
	if !snap.HasWorst || snap.Worst.Threshold == nil {
		return nil, nil
	}
	if requireShouldAct && !snap.ShouldAct {
		return nil, nil
	}
	if !requireShouldAct && !snap.AnyBreached {
		return nil, nil
	}
	th, ok := caches.threshold(product)
	if !ok {
		return nil, nil
	}
	pc, err := s.buildProductContextFromSnapshot(ctx, caches, product, sku, snap.ScopeContexts, maintainedProviders)
	if err != nil {
		return nil, err
	}
	action := buildScopeActionReadonly(
		caches, product, sku, providersFromSnapshot(snap), th, snap.Worst, *snap.Worst.Threshold, pc,
	)
	if action == nil {
		return nil, nil
	}
	return action, nil
}

// buildScopeActionReadonly computes routing/maintenance from snapshot only — no DB writes/extra queries.
func buildScopeActionReadonly(
	caches overviewCaches,
	product, sku string,
	providers map[string]float64,
	th store.ProductThreshold,
	worst agent.ScopeContext,
	eval threshold.Result,
	pc agent.ProductContext,
) *scopeAction {
	skuKind, skuReason := agent.SKURoutingDecision(pc, sku, providers, th)
	if skuKind == "maintenance" {
		eval.SuggestedAction = "maintenance"
		eval.SuggestedActionReason = skuReason
		return &scopeAction{kind: "maintenance", worst: worst, eval: eval}
	}
	eval.SuggestedAction = "routing"
	eval.SuggestedActionReason = skuReason
	reason := output.RoutingPlanReason(product, worst.ProviderCode, nil)
	if len(eval.BreachReasons) > 0 {
		reason = fmt.Sprintf("%s — %s", reason, eval.BreachReasons[0])
	}
	plan := agent.BuildRoutingPlan(pc, worst, reason, th)
	return &scopeAction{kind: "routing", plan: plan, worst: worst, eval: eval}
}

// maintenanceWarrantedFromSnapshot is true when live metrics still require SKU maintenance (not routing).
func maintenanceWarrantedFromSnapshot(
	snap scopeSnapshot,
	sku string,
	routing map[string]float64,
	maintainedProviders []string,
	th store.ProductThreshold,
) bool {
	if !snap.AnyBreached || !snap.HasWorst {
		return false
	}
	pc := agent.ProductContext{
		Scopes:              snap.ScopeContexts,
		MaintainedProviders: maintainedProviderSet(maintainedProviders),
	}
	kind, _ := agent.SKURoutingDecision(pc, sku, routing, th)
	if kind != "maintenance" {
		return false
	}
	return maintenanceSuggestionStillNeeded(snap, routing, maintainedProviders, th)
}

// maintenanceSuggestionStillNeeded is false when every breached routing provider is already in active maintenance.
func maintenanceSuggestionStillNeeded(
	snap scopeSnapshot,
	routing map[string]float64,
	maintainedProviders []string,
	th store.ProductThreshold,
) bool {
	maintained := map[string]struct{}{}
	for _, p := range maintainedProviders {
		maintained[p] = struct{}{}
	}
	for provider, pct := range routing {
		if pct <= 0 {
			continue
		}
		raw, ok := snap.ProviderMetrics[provider].(map[string]any)
		if !ok {
			continue
		}
		success, _ := raw["success_pct"].(float64)
		pending, _ := raw["pending_pct"].(float64)
		fail, _ := raw["fail_pct"].(float64)
		pTxn := metricTxnFromSnapshot(raw, "pending_txn")
		fTxn := metricTxnFromSnapshot(raw, "fail_txn")
		if !store.ScopeBreachedFromSnapshot(success, pending, fail, pTxn, fTxn, th) {
			continue
		}
		if _, inMaint := maintained[provider]; !inMaint {
			return true
		}
	}
	return false
}

func metricTxnFromSnapshot(raw map[string]any, key string) uint {
	if v, ok := raw[key].(uint); ok {
		return v
	}
	if v, ok := raw[key].(float64); ok {
		return uint(math.Round(v))
	}
	return 0
}

func providersFromSnapshot(snap scopeSnapshot) map[string]float64 {
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

func (s *Server) buildScopeAction(
	ctx context.Context,
	product, sku string,
	providers map[string]float64,
	th store.ProductThreshold,
	worst agent.ScopeContext,
	eval threshold.Result,
	kind string,
	buildPC func() (agent.ProductContext, error),
) (*scopeAction, error) {
	if kind == "" {
		kind, _ = suggestActionFromDB(ctx, s.DB, product, worst.ProviderCode, sku)
	}
	pc, err := buildPC()
	if err != nil {
		return nil, err
	}
	skuKind, skuReason := agent.SKURoutingDecision(pc, sku, providers, th)
	if skuKind == "maintenance" {
		eval.SuggestedAction = "maintenance"
		eval.SuggestedActionReason = skuReason
		_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
		return &scopeAction{kind: "maintenance", worst: worst, eval: eval}, nil
	}
	hasMaint, _ := s.DB.HasPendingMaintenanceRecommendation(ctx, product, sku)
	if hasMaint {
		return nil, nil
	}
	if kind != "routing" {
		return &scopeAction{kind: "maintenance", worst: worst, eval: eval}, nil
	}
	reason := output.RoutingPlanReason(product, worst.ProviderCode, nil)
	if len(eval.BreachReasons) > 0 {
		reason = fmt.Sprintf("%s — %s", reason, eval.BreachReasons[0])
	}
	plan := agent.BuildRoutingPlan(pc, worst, reason, th)
	return &scopeAction{kind: "routing", plan: plan, worst: worst, eval: eval}, nil
}

func maintainedProviderSet(providers []string) map[string]bool {
	out := make(map[string]bool, len(providers))
	for _, p := range providers {
		out[p] = true
	}
	return out
}

func (s *Server) buildProductContextFromSnapshot(
	ctx context.Context,
	caches overviewCaches,
	product, sku string,
	scopes []agent.ScopeContext,
	maintainedProviders []string,
) (agent.ProductContext, error) {
	routing, prov, err := caches.productRouting(ctx, s.DB, product)
	if err != nil {
		return agent.ProductContext{}, err
	}
	filtered := make([]agent.ScopeContext, 0, len(scopes))
	for _, sc := range scopes {
		if sc.SKUCode == sku {
			filtered = append(filtered, sc)
		}
	}
	return agent.ProductContext{
		Product:             domain.Product{ProductCode: product},
		Providers:           prov,
		Routing:             routing,
		Scopes:              filtered,
		MaintainedProviders: maintainedProviderSet(maintainedProviders),
	}, nil
}

func (s *Server) resolveScopeAction(
	ctx context.Context,
	product, sku string,
	providers map[string]float64,
	th store.ProductThreshold,
) (*scopeAction, error) {
	worst, ok, err := s.worstProviderInScope(ctx, product, sku, providers, th)
	if err != nil || !ok {
		return nil, err
	}
	if worst.Threshold == nil {
		return nil, nil
	}
	eval := *worst.Threshold
	kind := eval.SuggestedAction
	return s.buildScopeAction(ctx, product, sku, providers, th, worst, eval, kind, func() (agent.ProductContext, error) {
		return s.buildProductContextForScope(ctx, product, sku, providers, th)
	})
}

func syntheticPendingPlanMap(product, sku string, plan agent.RoutingPlanJSON) map[string]any {
	raw, _ := json.Marshal(plan)
	return map[string]any{
		"id":           0,
		"product_code": product,
		"sku_code":     sku,
		"scope":        plan.Scope,
		"status":       "pending_approve",
		"suggested":    true,
		"plan":         roundPlanJSON(raw),
	}
}

// scopeAnyProviderBreached is true when any provider with routing_pct > 0 violates 1 of 5 snapshot metrics.
func (s *Server) scopeAnyProviderBreached(
	ctx context.Context,
	product, sku string,
	providers map[string]float64,
	th store.ProductThreshold,
) bool {
	_, ok, _ := s.worstProviderInScope(ctx, product, sku, providers, th)
	return ok
}

func (s *Server) worstProviderInScope(
	ctx context.Context,
	product, sku string,
	providers map[string]float64,
	th store.ProductThreshold,
) (agent.ScopeContext, bool, error) {
	settings, err := s.DB.GetAgentSettings(ctx)
	if err != nil {
		return agent.ScopeContext{}, false, err
	}
	since := metricsSince(th.MetricsWindowMin)

	var worst agent.ScopeContext
	worstScore := -1.0
	found := false

	for provider, pct := range providers {
		if pct <= 0 {
			continue
		}
		m, ok, err := s.DB.GetMetricsInWindow(ctx, settings.DataSource, product, sku, provider, since)
		if err != nil || !ok {
			continue
		}
		pendingTxn := uint(math.Round(float64(m.TotalTransactions) * m.PendingRate / 100))
		failTxn := uint(math.Round(float64(m.TotalTransactions) * m.FailRate / 100))
		if !store.ScopeBreachedFromSnapshot(m.SuccessRate, m.PendingRate, m.FailRate, pendingTxn, failTxn, th) {
			continue
		}
		reasons := store.SnapshotBreachReasons(m.SuccessRate, m.PendingRate, m.FailRate, pendingTxn, failTxn, th)
		action, actionReason := suggestActionFromDB(ctx, s.DB, product, provider, sku)
		eval := threshold.Result{
			ProductCode:             product,
			Breached:                true,
			BreachReasons:           reasons,
			ShouldAct:               true,
			ShouldAlertMode:         true,
			SuggestedAction:         action,
			SuggestedActionReason:   actionReason,
		}
		score := m.FailRate*2 + m.PendingRate + float64(pendingTxn) + float64(failTxn)*0.5
		if score > worstScore {
			worstScore = score
			worst = agent.ScopeContext{
				ProductCode:  product,
				SKUCode:      sku,
				ProviderCode: provider,
				Metrics: &tools.GetMetricsOutput{
					SuccessRate:       m.SuccessRate,
					PendingRate:       m.PendingRate,
					FailRate:          m.FailRate,
					TotalTransactions: m.TotalTransactions,
				},
				Threshold: &eval,
				State:     "INCIDENT",
			}
			found = true
		}
	}
	return worst, found, nil
}

func suggestActionFromDB(ctx context.Context, db *store.DB, product, badProvider, sku string) (string, string) {
	th, err := db.GetProductThreshold(ctx, product)
	if err != nil {
		return "maintenance", "Không đọc được ngưỡng sản phẩm"
	}
	routing, err := db.GetRoutingForScope(ctx, product, sku)
	if err != nil {
		return "maintenance", "Không đọc được routing"
	}
	settings, err := db.GetAgentSettings(ctx)
	if err != nil {
		return "maintenance", "Không đọc được cấu hình agent"
	}
	windowMin := th.MetricsWindowMin
	if windowMin <= 0 {
		windowMin = 15
	}
	since := metricsSince(windowMin)

	active := 0
	healthy := 0
	for _, row := range routing {
		if row.TrafficPct <= 0 {
			continue
		}
		active++
		if row.ProviderCode == badProvider {
			continue
		}
		m, ok, _ := db.GetMetricsInWindow(ctx, settings.DataSource, product, sku, row.ProviderCode, since)
		if !ok {
			continue
		}
		pendingTxn := uint(math.Round(float64(m.TotalTransactions) * m.PendingRate / 100))
		failTxn := uint(math.Round(float64(m.TotalTransactions) * m.FailRate / 100))
		if !store.ScopeBreachedFromSnapshot(m.SuccessRate, m.PendingRate, m.FailRate, pendingTxn, failTxn, th) {
			healthy++
		}
	}
	if active <= 1 {
		return "maintenance", fmt.Sprintf("Chỉ %d provider đang routing — không thể chuyển traffic", active)
	}
	if healthy >= 1 {
		return "routing", fmt.Sprintf("%d provider routing; %d provider trong ngưỡng — có thể chuyển traffic", active, healthy)
	}
	return "maintenance", "Tất cả provider đang routing đều vi phạm ngưỡng — đề xuất bảo trì SKU"
}

func (s *Server) buildProductContextForScope(
	ctx context.Context,
	product, sku string,
	providers map[string]float64,
	th store.ProductThreshold,
) (agent.ProductContext, error) {
	reg := tools.NewRegistry(s.DB, s.Notify)
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
	since := metricsSince(th.MetricsWindowMin)

	var allScopes []agent.ScopeContext
	for provider, pct := range providers {
		if pct <= 0 {
			continue
		}
		m, ok, err := s.DB.GetMetricsInWindow(ctx, settings.DataSource, product, sku, provider, since)
		if err != nil || !ok {
			continue
		}
		pendingTxn := uint(math.Round(float64(m.TotalTransactions) * m.PendingRate / 100))
		failTxn := uint(math.Round(float64(m.TotalTransactions) * m.FailRate / 100))
		breached := store.ScopeBreachedFromSnapshot(m.SuccessRate, m.PendingRate, m.FailRate, pendingTxn, failTxn, th)
		var eval threshold.Result
		if breached {
			action, actionReason := suggestActionFromDB(ctx, s.DB, product, provider, sku)
			eval = threshold.Result{
				ProductCode:           product,
				Breached:              true,
				BreachReasons:         store.SnapshotBreachReasons(m.SuccessRate, m.PendingRate, m.FailRate, pendingTxn, failTxn, th),
				ShouldAct:             true,
				SuggestedAction:       action,
				SuggestedActionReason: actionReason,
			}
		}
		allScopes = append(allScopes, agent.ScopeContext{
			ProductCode:  product,
			SKUCode:      sku,
			ProviderCode: provider,
			Metrics: &tools.GetMetricsOutput{
				SuccessRate:       m.SuccessRate,
				PendingRate:       m.PendingRate,
				FailRate:          m.FailRate,
				TotalTransactions: m.TotalTransactions,
			},
			Threshold: &eval,
		})
	}

	return agent.ProductContext{
		Product:   domain.Product{ProductCode: product},
		Providers: prov,
		Routing:   routing,
		Scopes:    allScopes,
	}, nil
}

func metricsSince(windowMin int) time.Time {
	if windowMin <= 0 {
		windowMin = 15
	}
	return time.Now().Add(-time.Duration(windowMin) * time.Minute)
}

func routingMapsEqual(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if math.Abs(v-b[k]) > 0.05 {
			return false
		}
	}
	return true
}

// routingProposalSuppressedByReject skips re-showing the same synthetic plan after admin reject (no time window).
func routingProposalSuppressedByReject(latest store.RoutingPlanRow, plan agent.RoutingPlanJSON) bool {
	if latest.Status != "rejected" {
		return false
	}
	var stored agent.RoutingPlanJSON
	if err := json.Unmarshal(latest.PlanJSON, &stored); err != nil {
		return false
	}
	return routingMapsEqual(stored.Proposed, plan.Proposed)
}

func (s *Server) routingProposalSuppressedAfterReject(ctx context.Context, product, sku string, plan agent.RoutingPlanJSON) bool {
	latest, ok, err := s.DB.GetLatestRoutingPlanForScope(ctx, product, sku)
	if err != nil || !ok {
		return false
	}
	return routingProposalSuppressedByReject(latest, plan)
}

// scopeAutoApplyAllowed reports whether poll auto-apply may run (ShouldAct or force maintenance).
func scopeAutoApplyAllowed(
	snap scopeSnapshot,
	sku string,
	routing map[string]float64,
	maintainedProviders []string,
	th store.ProductThreshold,
) bool {
	if snap.ShouldAct {
		return true
	}
	if !snap.AnyBreached || !snap.HasWorst {
		return false
	}
	pc := agent.ProductContext{
		Scopes:              snap.ScopeContexts,
		MaintainedProviders: maintainedProviderSet(maintainedProviders),
	}
	return agent.ShouldForceAutoMaintenance(pc, sku, routing, th) ||
		agent.ShouldForceAutoMaintenanceAllProviders(pc, sku, routing, th) ||
		agent.ShouldForceAutoRouting(pc, sku, routing, th)
}

// autoApplyScopeFromSnapshot applies routing/maintenance when scope is in auto mode (§9.5.2).
func (s *Server) autoApplyScopeFromSnapshot(
	ctx context.Context,
	caches overviewCaches,
	product, sku string,
	snap scopeSnapshot,
	maintainedProviders []string,
	inRecoveryGrace bool,
) (bool, error) {
	if !snap.HasWorst || !snap.AnyBreached {
		return false, nil
	}
	th, ok := caches.threshold(product)
	if !ok {
		return false, nil
	}
	pc, err := s.buildProductContextFromSnapshot(ctx, caches, product, sku, snap.ScopeContexts, maintainedProviders)
	if err != nil {
		return false, err
	}
	routing := providersFromSnapshot(snap)
	forceMaint := agent.ShouldForceAutoMaintenance(pc, sku, routing, th)
	forceAllMaint := agent.ShouldForceAutoMaintenanceAllProviders(pc, sku, routing, th)
	forceRouting := agent.ShouldForceAutoRouting(pc, sku, routing, th)
	if inRecoveryGrace && !forceMaint && !forceAllMaint {
		return false, nil
	}
	if inRecoveryGrace {
		forceRouting = false
	}
	if !snap.ShouldAct && !forceMaint && !forceAllMaint && !forceRouting {
		return false, nil
	}
	requireShouldAct := snap.ShouldAct && !forceMaint && !forceAllMaint && !forceRouting
	action, err := s.resolveScopeActionFromSnapshot(ctx, caches, product, sku, snap, maintainedProviders, requireShouldAct)
	if err != nil || action == nil {
		if (forceMaint || forceAllMaint) && snap.Worst.Threshold != nil {
			return s.autoApplyMaintenanceForScope(ctx, product, sku, snap, maintainedProviders, snap.Worst, *snap.Worst.Threshold)
		}
		return false, err
	}
	switch action.kind {
	case "routing":
		if routingMapsEqual(action.plan.Current, action.plan.Proposed) {
			return s.autoApplyMaintenanceForScope(ctx, product, sku, snap, maintainedProviders, action.worst, action.eval)
		}
		_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
		prod, err := s.DB.GetProductByCode(ctx, product)
		if err != nil {
			return false, err
		}
		incidentID, _ := s.DB.FindOpenIncidentForScope(ctx, product, sku, nil)
		var incPtr *string
		if incidentID != "" {
			incPtr = &incidentID
		}
		out, err := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
			Product:     product,
			ServiceType: string(prod.ServiceType),
			Scope:       action.plan.Scope,
			SKU:         sku,
			Routing:     action.plan.Proposed,
			TriggerType: "auto",
			ExecutedBy:  "opsone-api",
			Reason:      action.plan.Reason,
			IncidentID:  incPtr,
		})
		if err != nil {
			return false, err
		}
		if out.Applied {
			_, _ = s.DB.InsertRoutingPlan(ctx, nil, product, action.plan.Scope, sku, action.plan, "executed")
		}
		return out.Applied, nil
	case "maintenance":
		return s.autoApplyMaintenanceForScope(ctx, product, sku, snap, maintainedProviders, action.worst, action.eval)
	default:
		return false, nil
	}
}

func (s *Server) autoApplyMaintenanceForScope(
	ctx context.Context,
	product, sku string,
	snap scopeSnapshot,
	maintainedProviders []string,
	worst agent.ScopeContext,
	eval threshold.Result,
) (bool, error) {
	th, err := s.DB.GetProductThreshold(ctx, product)
	if err != nil {
		return false, err
	}
	if !maintenanceSuggestionStillNeeded(snap, providersFromSnapshot(snap), maintainedProviders, th) {
		return false, nil
	}
	detail := eval.SuggestedActionReason
	if detail == "" {
		detail = output.MaintenanceDetail(product, worst.ProviderCode, eval.BreachReasons, nil)
	}
	providers, err := s.DB.GetRoutingForScope(ctx, product, sku)
	if err != nil {
		return false, err
	}
	targets := maintenanceTargetsForScope(detail, worst.ProviderCode, providers)
	if len(targets) == 0 {
		return false, nil
	}
	defaultMin := maintenanceDefaultDurationMin(ctx, s.DB)
	startsAt := time.Now()
	endsAt := startsAt.Add(time.Duration(defaultMin) * time.Minute)
	_, err = applyMaintenanceTargets(
		ctx, s.Tools, product, sku, "opsone-api",
		fmt.Sprintf("auto maintenance scope %s/%s", product, sku),
		targets, startsAt, endsAt,
	)
	return err == nil, err
}

func (s *Server) pendingMaintenanceForScope(ctx context.Context, product, sku string) map[string]any {
	rec, ok, err := s.DB.LatestPendingMaintenanceForScope(ctx, product, sku)
	if err != nil || !ok {
		return nil
	}
	return map[string]any{
		"id":            rec.ID,
		"sku_code":      rec.SKUCode,
		"provider_code": rec.ProviderCode,
		"reason":        rec.Detail,
		"action_type":   "maintenance",
	}
}
