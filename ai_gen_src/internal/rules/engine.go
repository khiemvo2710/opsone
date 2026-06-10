package rules

import (
	"fmt"
	"sort"

	"opsone/internal/store"
)

type ruleFn func(ScopeInput) RuleResult
type productRuleFn func(ProductInput) RuleResult

type ruleDef struct {
	id       string
	priority int
	services []string
	fn       ruleFn
}

type productRuleDef struct {
	id       string
	priority int
	services []string
	fn       productRuleFn
}

// Engine runs deterministic rules §7.3.
type Engine struct{}

// EvaluateScope runs rules 1–6, 9 for one scope when breached.
func (e *Engine) EvaluateScope(in ScopeInput) []RuleResult {
	if !in.Scope.Breached {
		return nil
	}

	svc := in.Scope.ServiceType
	defs := []ruleDef{
		{id: "R5_PROVIDER_OVERLOAD", priority: 30, services: allSvc(), fn: r5ProviderOverload},
		{id: "R6_BACKUP_HEALTHIER", priority: 30, services: allSvc(), fn: r6BackupHealthier},
		{id: "R3_FAIL_SPIKE", priority: 20, services: allSvc(), fn: r3FailSpike},
		{id: "R4_TOP_ERROR_SHIFT", priority: 15, services: allSvc(), fn: r4TopErrorShift},
		{id: "R1_SUCCESS_DECLINE", priority: 10, services: allSvc(), fn: r1SuccessDecline},
		{id: "R2_PENDING_RISE", priority: 10, services: allSvc(), fn: r2PendingRise},
	}

	var out []RuleResult
	for _, d := range sortRules(defs, svc) {
		r := d.fn(in)
		if r.Triggered {
			out = append(out, r)
		}
	}
	if len(out) > 0 {
		r9 := r9RevenuePriority(in, out)
		if r9.Triggered {
			out = append(out, r9)
		}
	}
	return out
}

// EvaluateProduct runs SKU rules 7–8 at product level.
func (e *Engine) EvaluateProduct(in ProductInput) []RuleResult {
	if in.Product.RoutingMode != "sku" {
		return nil
	}
	if !productBreached(in.Product) {
		return nil
	}
	defs := []productRuleDef{
		{id: "R7_SKU_QUALITY_DIVERGENCE", priority: 25, services: []string{"card", "topup_data"}, fn: r7SKUQualityDivergence},
		{id: "R8_SKU_TRAFFIC_IMBALANCE", priority: 25, services: []string{"card", "topup_data"}, fn: r8SKUTrafficImbalance},
	}
	var out []RuleResult
	for _, d := range sortProductRules(defs, in.Product.ServiceType) {
		r := d.fn(in)
		if r.Triggered {
			out = append(out, r)
		}
	}
	return out
}

func productBreached(p ProductData) bool {
	for _, s := range p.Scopes {
		if s.Breached {
			return true
		}
	}
	return false
}

func allSvc() []string {
	return []string{"card", "topup_data", "topup"}
}

func sortRules(defs []ruleDef, svc string) []ruleDef {
	var filtered []ruleDef
	for _, d := range defs {
		if appliesTo(d.services, svc) {
			filtered = append(filtered, d)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].priority > filtered[j].priority })
	return filtered
}

func sortProductRules(defs []productRuleDef, svc string) []productRuleDef {
	var filtered []productRuleDef
	for _, d := range defs {
		if appliesTo(d.services, svc) {
			filtered = append(filtered, d)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].priority > filtered[j].priority })
	return filtered
}

func appliesTo(list []string, svc string) bool {
	for _, s := range list {
		if s == svc {
			return true
		}
	}
	return false
}

func r1SuccessDecline(in ScopeInput) RuleResult {
	h := orderedHistory(in.History)
	if len(h) < 2 {
		return RuleResult{RuleID: "R1_SUCCESS_DECLINE"}
	}
	s1, s2 := h[0].SuccessRate, h[1].SuccessRate
	delta := s2 - s1
	if s1 < s2 && delta >= 5 {
		return RuleResult{
			RuleID: "R1_SUCCESS_DECLINE", Triggered: true, Tag: "success_decline",
			MessageVi:    fmt.Sprintf("Tỷ lệ thành công giảm %.0f%% trong 2 chu kỳ", delta),
			SeverityBump: 1,
			Evidence:     map[string]any{"delta": delta, "success_now": s1},
		}
	}
	return RuleResult{RuleID: "R1_SUCCESS_DECLINE"}
}

func r2PendingRise(in ScopeInput) RuleResult {
	h := orderedHistory(in.History)
	if len(h) < 2 {
		return RuleResult{RuleID: "R2_PENDING_RISE"}
	}
	p1, p2 := h[0].PendingRate, h[1].PendingRate
	if p1 > p2 {
		return RuleResult{
			RuleID: "R2_PENDING_RISE", Triggered: true, Tag: "pending_rise",
			MessageVi:    fmt.Sprintf("Pending tăng %.0f%% (2 chu kỳ liên tiếp)", p1),
			SeverityBump: 1,
			Evidence:     map[string]any{"pending_now": p1},
		}
	}
	return RuleResult{RuleID: "R2_PENDING_RISE"}
}

func r3FailSpike(in ScopeInput) RuleResult {
	fail := in.Scope.FailRate
	var prev float64
	if len(in.History) > 0 {
		prev = in.History[0].FailRate
	}
	failMax := in.FailMaxPct
	if failMax <= 0 {
		failMax = 10
	}
	delta := fail - prev
	if fail > failMax || delta >= 5 {
		return RuleResult{
			RuleID: "R3_FAIL_SPIKE", Triggered: true, Tag: "fail_spike",
			MessageVi:    fmt.Sprintf("Tỷ lệ lỗi %.0f%% (Δ +%.0f%% so chu kỳ trước)", fail, delta),
			SeverityBump: 2,
			Evidence:     map[string]any{"fail_now": fail, "delta": delta},
		}
	}
	return RuleResult{RuleID: "R3_FAIL_SPIKE"}
}

func r4TopErrorShift(in ScopeInput) RuleResult {
	if in.Scope.TopErrorCode == "" {
		return RuleResult{RuleID: "R4_TOP_ERROR_SHIFT"}
	}
	return RuleResult{
		RuleID: "R4_TOP_ERROR_SHIFT", Triggered: true, Tag: "top_error_shift",
		MessageVi:    fmt.Sprintf("Lỗi đầu bảng %s (%d lượt)", in.Scope.TopErrorCode, in.Scope.TopErrorCount),
		SeverityBump: 1,
		Evidence:     map[string]any{"code": in.Scope.TopErrorCode, "count": in.Scope.TopErrorCount},
	}
}

func r5ProviderOverload(in ScopeInput) RuleResult {
	pct := in.TrafficPct
	successMin := in.SuccessMinPct
	if successMin <= 0 {
		successMin = 80
	}
	if pct >= 60 && in.Scope.SuccessRate < successMin {
		return RuleResult{
			RuleID: "R5_PROVIDER_OVERLOAD", Triggered: true, Tag: "provider_overload",
			MessageVi: fmt.Sprintf("Provider %s gánh %.0f%% nhưng success %.0f%%", in.Scope.ProviderCode, pct, in.Scope.SuccessRate),
			SeverityBump: 2, SuggestedAction: "routing",
			Evidence:     map[string]any{"provider": in.Scope.ProviderCode, "pct": pct, "success": in.Scope.SuccessRate},
		}
	}
	return RuleResult{RuleID: "R5_PROVIDER_OVERLOAD"}
}

func r6BackupHealthier(in ScopeInput) RuleResult {
	if in.Scope.HealthyBackupCount < 1 {
		return RuleResult{RuleID: "R6_BACKUP_HEALTHIER"}
	}
	return RuleResult{
		RuleID: "R6_BACKUP_HEALTHIER", Triggered: true, Tag: "backup_healthier",
		MessageVi: fmt.Sprintf("Có %d provider backup healthy — có thể chuyển traffic khỏi %s",
			in.Scope.HealthyBackupCount, in.Scope.ProviderCode),
		SeverityBump: 1, SuggestedAction: "routing",
		Evidence:     map[string]any{"healthy_backup_count": in.Scope.HealthyBackupCount},
	}
}

func r9RevenuePriority(in ScopeInput, prior []RuleResult) RuleResult {
	minRev := in.RevenueMinVND
	if minRev == 0 {
		minRev = 50_000_000
	}
	if in.Scope.RevenueLastHour >= minRev {
		return RuleResult{
			RuleID: "R9_REVENUE_PRIORITY", Triggered: true, Tag: "high_revenue_impact",
			MessageVi:    fmt.Sprintf("Doanh thu ảnh hưởng %d VND/h — ưu tiên xử lý cao", in.Scope.RevenueLastHour),
			SeverityBump: 1,
			Evidence:     map[string]any{"revenue": in.Scope.RevenueLastHour},
		}
	}
	return RuleResult{RuleID: "R9_REVENUE_PRIORITY"}
}

func r7SKUQualityDivergence(in ProductInput) RuleResult {
	thMin := in.SuccessMinPct
	if thMin <= 0 {
		thMin = 80
	}
	var badSKU, okSKU string
	for _, s := range in.Product.Scopes {
		if s.SuccessRate < thMin && badSKU == "" {
			badSKU = s.SKUCode
		} else if s.SuccessRate >= thMin+10 {
			okSKU = s.SKUCode
		}
	}
	if badSKU != "" && okSKU != "" {
		return RuleResult{
			RuleID: "R7_SKU_QUALITY_DIVERGENCE", Triggered: true, Tag: "sku_quality_divergence",
			MessageVi:    fmt.Sprintf("SKU %s xấu nhưng SKU khác cùng provider vẫn ổn", badSKU),
			SeverityBump: 1, SuggestedAction: "routing",
			Evidence:     map[string]any{"sku_bad": badSKU},
		}
	}
	return RuleResult{RuleID: "R7_SKU_QUALITY_DIVERGENCE"}
}

func r8SKUTrafficImbalance(in ProductInput) RuleResult {
	thMin := in.SuccessMinPct
	if thMin <= 0 {
		thMin = 80
	}
	for _, s := range in.Product.Scopes {
		pct := trafficPctForScope(in.Product, s)
		if pct >= 70 && s.SuccessRate < thMin {
			return RuleResult{
				RuleID: "R8_SKU_TRAFFIC_IMBALANCE", Triggered: true, Tag: "sku_traffic_imbalance",
				MessageVi: fmt.Sprintf("SKU %s dồn %.0f%% vào %s đang lỗi", s.SKUCode, pct, s.ProviderCode),
				SeverityBump: 1, SuggestedAction: "routing",
				Evidence:     map[string]any{"sku": s.SKUCode, "pct": pct, "provider": s.ProviderCode},
			}
		}
	}
	return RuleResult{RuleID: "R8_SKU_TRAFFIC_IMBALANCE"}
}

func orderedHistory(h []store.ScopeHistoryPoint) []store.ScopeHistoryPoint {
	if len(h) <= 2 {
		return h
	}
	return h[:2]
}

func trafficPctForScope(p ProductData, s ScopeData) float64 {
	if p.RoutingBySKU != nil && s.SKUCode != "" {
		if m, ok := p.RoutingBySKU[s.SKUCode]; ok {
			return m[s.ProviderCode]
		}
	}
	if p.Routing != nil {
		return p.Routing[s.ProviderCode]
	}
	return 0
}
