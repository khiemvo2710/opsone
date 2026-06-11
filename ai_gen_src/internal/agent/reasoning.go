package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"opsone/internal/output"
	"opsone/internal/rules"
	"opsone/internal/store"
	"opsone/internal/tools"
)

// Reasoner runs rules + generates outputs (Phase 4, §7–§8).
type Reasoner struct {
	DB    *store.DB
	Rules rules.Engine
}

// NewReasoner creates a reasoner.
func NewReasoner(db *store.DB) *Reasoner {
	return &Reasoner{DB: db, Rules: rules.Engine{}}
}

// Process runs rules engine and persists Incident / RoutingPlan / Recommendation.
func (r *Reasoner) Process(ctx context.Context, cycleID uint64, products []ProductContext) ([]ProductContext, error) {
	day := time.Now()
	for i := range products {
		pc := &products[i]
		th, err := r.DB.GetProductThreshold(ctx, pc.Product.ProductCode)
		if err != nil {
			return nil, err
		}

		pd := toProductData(*pc)
		var productEvidence []rules.RuleResult
		for j := range pc.Scopes {
			sc := &pc.Scopes[j]
			hist, err := r.DB.GetRecentScopeHistory(ctx, sc.ProductCode, sc.SKUCode, sc.ProviderCode, cycleID, 5)
			if err != nil {
				return nil, err
			}
			sin := rules.ScopeInput{
				Scope:         toScopeData(*sc),
				History:       hist,
				TrafficPct:    trafficPct(*pc, *sc),
				SuccessMinPct: th.SuccessRateMinPct,
				FailMaxPct:    th.FailRateMaxPct,
			}
			sc.RuleEvidence = r.Rules.EvaluateScope(sin)
			productEvidence = append(productEvidence, sc.RuleEvidence...)
		}

		pc.ProductRuleEvidence = r.Rules.EvaluateProduct(rules.ProductInput{
			Product:       pd,
			SuccessMinPct: th.SuccessRateMinPct,
			ScopeEvidence: productEvidence,
		})

		if err := r.emitOutputs(ctx, cycleID, day, pc); err != nil {
			return nil, err
		}
		r.refreshProductHealth(pc)
	}
	return products, nil
}

func toScopeData(s ScopeContext) rules.ScopeData {
	d := rules.ScopeData{
		ProductCode:  s.ProductCode,
		ServiceType:  s.ServiceType,
		SKUCode:      s.SKUCode,
		ProviderCode: s.ProviderCode,
	}
	if s.Metrics != nil {
		d.SuccessRate = s.Metrics.SuccessRate
		d.PendingRate = s.Metrics.PendingRate
		d.FailRate = s.Metrics.FailRate
	}
	if s.Revenue != nil {
		d.RevenueLastHour = s.Revenue.RevenueLastHour
	}
	if s.TopErrors != nil && len(s.TopErrors.Errors) > 0 {
		d.TopErrorCode = s.TopErrors.Errors[0].Error
		d.TopErrorCount = s.TopErrors.Errors[0].Count
	}
	if s.Threshold != nil {
		d.Breached = s.Threshold.Breached
		d.HealthyBackupCount = s.Threshold.HealthyBackupCount
	}
	return d
}

func toProductData(pc ProductContext) rules.ProductData {
	pd := rules.ProductData{
		ServiceType:  string(pc.Product.ServiceType),
		RoutingMode:  string(pc.Product.RoutingMode),
		Routing:      pc.Routing.Routing,
		RoutingBySKU: pc.Routing.RoutingBySKU,
	}
	for _, s := range pc.Scopes {
		pd.Scopes = append(pd.Scopes, toScopeData(s))
	}
	return pd
}

func (r *Reasoner) emitOutputs(ctx context.Context, cycleID uint64, day time.Time, pc *ProductContext) error {
	cyclePtr := &cycleID
	worst := worstScope(pc)
	if worst == nil || worst.Threshold == nil {
		return nil
	}
	thRes := worst.Threshold
	if !thRes.Breached {
		return nil
	}

	allEvidence := append(pc.ProductRuleEvidence, collectScopeEvidence(pc)...)

	prodTh, err := r.DB.GetProductThreshold(ctx, pc.Product.ProductCode)
	if err != nil {
		return err
	}
	routing := currentRouting(*pc, *worst)
	skuAction, skuReason := SKURoutingDecision(*pc, worst.SKUCode, routing, prodTh)
	effectiveAction := thRes.SuggestedAction
	if skuAction == "maintenance" {
		effectiveAction = "maintenance"
	}
	forceMaint := thRes.Breached && ShouldForceAutoMaintenance(*pc, worst.SKUCode, routing, prodTh)
	forceAllMaint := thRes.Breached && ShouldForceAutoMaintenanceAllProviders(*pc, worst.SKUCode, routing, prodTh)
	forceRouting := thRes.Breached && ShouldForceAutoRouting(*pc, worst.SKUCode, routing, prodTh)

	inReopenGrace := false
	if applyCycle, ok, err := r.DB.GetRecoveryApplyCycle(ctx, pc.Product.ProductCode, worst.SKUCode); err == nil && ok {
		if cycleID <= applyCycle {
			inReopenGrace = true
		}
	}

	if thRes.ShouldAct || forceMaint || forceAllMaint || forceRouting {
		existingID, err := r.DB.FindOpenIncidentForScope(ctx, pc.Product.ProductCode, worst.SKUCode, nil)
		if err != nil {
			return err
		}
		var incID string
		if existingID != "" {
			incID = existingID
		} else {
			incID, err = r.DB.NextIncidentID(ctx, day)
			if err != nil {
				return err
			}
			m := worst.Metrics
			var failBefore *float64
			if hist, err := r.DB.GetRecentScopeHistory(ctx, worst.ProductCode, worst.SKUCode, worst.ProviderCode, cycleID, 1); err == nil && len(hist) > 0 {
				f := hist[0].FailRate
				failBefore = &f
			}
			sev := output.SeverityFromEvidence("medium", allEvidence)
			summary, _ := output.IncidentSummary(
				pc.Product.ProductCode, worst.ProviderCode, worst.SKUCode,
				m.SuccessRate, m.FailRate, allEvidence, thRes.BreachReasons,
			)
			if err := r.DB.InsertIncident(ctx, store.IncidentInsert{
				IncidentID: incID, CycleID: cyclePtr, Severity: sev,
				ProductCode: pc.Product.ProductCode, ProviderCode: worst.ProviderCode, SKUCode: worst.SKUCode,
				SuccessAfter: &m.SuccessRate, FailAfter: &m.FailRate,
				FailBefore: failBefore, Summary: summary,
			}); err != nil {
				return err
			}
		}
		pc.LastIncidentID = incID

		switch effectiveAction {
		case "routing":
			if inReopenGrace {
				break
			}
			hasMaint, err := r.DB.HasPendingMaintenanceRecommendation(ctx, pc.Product.ProductCode, worst.SKUCode)
			if err != nil {
				return err
			}
			if hasMaint {
				break
			}
			plan := BuildRoutingPlan(*pc, *worst, output.RoutingPlanReason(pc.Product.ProductCode, worst.ProviderCode, allEvidence), prodTh)
			scope := plan.Scope
			scopeAuto, _ := r.DB.ResolveEffectiveScopeAuto(ctx, pc.Product.ProductCode, worst.SKUCode)
			autoApply := store.ShouldAutoApplyScope(scopeAuto, time.Now())
			hasPending, err := r.DB.HasPendingRoutingPlan(ctx, pc.Product.ProductCode, worst.SKUCode)
			if err != nil {
				return err
			}
			if hasPending {
				if autoApply {
					_ = r.DB.CancelPendingRoutingPlansForScope(ctx, pc.Product.ProductCode, worst.SKUCode)
					if err := r.applyRoutingPlanAuto(ctx, cyclePtr, pc, worst, plan, &pc.LastIncidentID); err != nil {
						return err
					}
					pc.HealthSummary = fmt.Sprintf("%s — Đã tự động áp dụng routing", productDisplayName(*pc))
					pc.HealthStatus = "yellow"
					routing = currentRouting(*pc, *worst)
					if ShouldForceAutoMaintenance(*pc, worst.SKUCode, routing, prodTh) {
						detail := skuReason
						if detail == "" {
							detail = output.MaintenanceDetail(pc.Product.ProductCode, worst.ProviderCode, thRes.BreachReasons)
						}
						if err := r.applyMaintenanceAuto(ctx, pc, worst, detail); err != nil {
							return err
						}
						pc.HealthSummary = fmt.Sprintf("%s — Đã tự động bảo trì %s", productDisplayName(*pc), worst.ProviderCode)
						pc.HealthStatus = "red"
					}
				} else {
					if err := r.DB.UpdatePendingRoutingPlanForScope(ctx, cyclePtr, pc.Product.ProductCode, worst.SKUCode, plan); err != nil {
						return err
					}
					pc.HealthSummary = fmt.Sprintf("%s — Kế hoạch routing chờ duyệt", productDisplayName(*pc))
					pc.HealthStatus = "yellow"
				}
				break
			}
			if autoApply {
				if err := r.applyRoutingPlanAuto(ctx, cyclePtr, pc, worst, plan, &pc.LastIncidentID); err != nil {
					return err
				}
				pc.HealthSummary = fmt.Sprintf("%s — Đã tự động áp dụng routing", productDisplayName(*pc))
				pc.HealthStatus = "yellow"
				routing = currentRouting(*pc, *worst)
				if ShouldForceAutoMaintenance(*pc, worst.SKUCode, routing, prodTh) {
					detail := skuReason
					if detail == "" {
						detail = output.MaintenanceDetail(pc.Product.ProductCode, worst.ProviderCode, thRes.BreachReasons)
					}
					if err := r.applyMaintenanceAuto(ctx, pc, worst, detail); err != nil {
						return err
					}
					pc.HealthSummary = fmt.Sprintf("%s — Đã tự động bảo trì %s", productDisplayName(*pc), worst.ProviderCode)
					pc.HealthStatus = "red"
				}
				break
			}
			_, err = r.DB.InsertRoutingPlan(ctx, cyclePtr, pc.Product.ProductCode, scope, worst.SKUCode, plan, "pending_approve")
			if err != nil {
				return err
			}
			pc.HealthSummary = fmt.Sprintf("%s — Kế hoạch routing chờ duyệt", productDisplayName(*pc))
			pc.HealthStatus = "yellow"
		case "maintenance":
			if inReopenGrace && !forceMaint && !forceAllMaint {
				break
			}
			detail := output.MaintenanceDetail(pc.Product.ProductCode, worst.ProviderCode, thRes.BreachReasons)
			if strings.Contains(skuReason, "Tất cả provider") {
				detail = skuReason
			}
			_ = r.DB.CancelPendingRoutingPlansForScope(ctx, pc.Product.ProductCode, worst.SKUCode)
			scopeAuto, _ := r.DB.ResolveEffectiveScopeAuto(ctx, pc.Product.ProductCode, worst.SKUCode)
			if store.ShouldAutoApplyScope(scopeAuto, time.Now()) {
				if err := r.applyMaintenanceAuto(ctx, pc, worst, detail); err != nil {
					return err
				}
				pc.HealthSummary = fmt.Sprintf("%s — Đã tự động bảo trì %s", productDisplayName(*pc), worst.ProviderCode)
				pc.HealthStatus = "red"
				break
			}
			inc := pc.LastIncidentID
			var incPtr *string
			if inc != "" {
				incPtr = &inc
			}
			if err := r.DB.InsertRecommendation(ctx, cyclePtr, incPtr, pc.Product.ProductCode, worst.SKUCode, "maintenance", detail); err != nil {
				return err
			}
			pc.HealthSummary = fmt.Sprintf("%s — Đề xuất bảo trì %s", productDisplayName(*pc), worst.ProviderCode)
			pc.HealthStatus = "red"
		}
	} else if thRes.Breached {
		has, err := r.DB.HasRecentRecommendation(ctx, pc.Product.ProductCode, worst.SKUCode, "monitor")
		if err != nil {
			return err
		}
		if !has {
			detail := output.MonitorRecommendation(pc.Product.ProductCode)
			if err := r.DB.InsertRecommendation(ctx, cyclePtr, nil, pc.Product.ProductCode, worst.SKUCode, "monitor", detail); err != nil {
				return err
			}
		}
		pc.HealthStatus = "yellow"
		pc.HealthSummary = output.MonitorRecommendation(productDisplayName(*pc))
	}
	return nil
}

func maintenanceTargetsFromRouting(detail, providerCode string, routing map[string]float64) []string {
	if strings.Contains(detail, "Tất cả provider đang routing") {
		var out []string
		for p, pct := range routing {
			if pct > 0 {
				out = append(out, p)
			}
		}
		return out
	}
	if providerCode != "" {
		return []string{providerCode}
	}
	return nil
}

func (r *Reasoner) applyMaintenanceAuto(ctx context.Context, pc *ProductContext, worst *ScopeContext, detail string) error {
	routing := currentRouting(*pc, *worst)
	targets := maintenanceTargetsFromRouting(detail, worst.ProviderCode, routing)
	if len(targets) == 0 {
		return nil
	}
	settings, err := r.DB.GetAgentSettings(ctx)
	if err != nil {
		return err
	}
	durationMin := store.NormalizeMaintenanceDefaultDurationMin(settings.MaintenanceDefaultDurationMin)
	startsAt := time.Now()
	endsAt := startsAt.Add(time.Duration(durationMin) * time.Minute)
	reg := tools.NewRegistry(r.DB)
	reason := fmt.Sprintf("auto maintenance: %s", detail)
	for i, provider := range targets {
		_, err := reg.SetMaintenance(ctx, tools.SetMaintenanceInput{
			Product:     pc.Product.ProductCode,
			Provider:    provider,
			SKU:         worst.SKUCode,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
			TriggerType: "auto",
			Reason:      reason,
			Status:      "active",
			Seq:         i,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reasoner) applyRoutingPlanAuto(
	ctx context.Context,
	cycleID *uint64,
	pc *ProductContext,
	worst *ScopeContext,
	plan RoutingPlanJSON,
	incidentID *string,
) error {
	reg := tools.NewRegistry(r.DB)
	out, err := reg.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     pc.Product.ProductCode,
		ServiceType: string(pc.Product.ServiceType),
		Scope:       plan.Scope,
		SKU:         worst.SKUCode,
		Routing:     plan.Proposed,
		TriggerType: "auto",
		ExecutedBy:  "opsone-agent",
		Reason:      plan.Reason,
		CycleID:     cycleID,
		IncidentID:  incidentID,
	})
	if err != nil {
		return err
	}
	if !out.Applied {
		return nil
	}
	_, err = r.DB.InsertRoutingPlan(ctx, cycleID, pc.Product.ProductCode, plan.Scope, worst.SKUCode, plan, "executed")
	return err
}

func (r *Reasoner) refreshProductHealth(pc *ProductContext) {
	if pc.HealthStatus == "" {
		pc.HealthStatus, pc.HealthSummary, pc.State = aggregateProductHealth(*pc)
	}
}

func worstScope(pc *ProductContext) *ScopeContext {
	var worst *ScopeContext
	rank := 0
	for i := range pc.Scopes {
		s := &pc.Scopes[i]
		if s.Skipped || s.Metrics == nil || s.Threshold == nil || !s.Threshold.Breached {
			continue
		}
		r := scopeRank(s)
		if r > rank {
			rank = r
			worst = s
		}
	}
	return worst
}

func scopeRank(s *ScopeContext) int {
	if s.State == "INCIDENT" {
		return 3
	}
	if s.State == "WARNING" {
		return 2
	}
	return 1
}

func collectScopeEvidence(pc *ProductContext) []rules.RuleResult {
	var out []rules.RuleResult
	for _, s := range pc.Scopes {
		out = append(out, s.RuleEvidence...)
	}
	return out
}

func trafficPct(pc ProductContext, s ScopeContext) float64 {
	if pc.Routing.RoutingBySKU != nil && s.SKUCode != "" {
		if m, ok := pc.Routing.RoutingBySKU[s.SKUCode]; ok {
			return m[s.ProviderCode]
		}
	}
	if pc.Routing.Routing != nil {
		return pc.Routing.Routing[s.ProviderCode]
	}
	return 0
}
