package agent

import (
	"fmt"
	"math"

	"opsone/internal/store"
)

// ScopeBreached reports whether scope metrics violate any of the 5 product thresholds.
func ScopeBreached(sc ScopeContext, th store.ProductThreshold) bool {
	if sc.Threshold != nil && sc.Threshold.Breached {
		return true
	}
	return scopeMetricsBreached(sc, th)
}

// scopeMetricsBreached evaluates live metrics only (không dùng cờ Threshold cũ).
func scopeMetricsBreached(sc ScopeContext, th store.ProductThreshold) bool {
	if sc.Metrics == nil {
		return false
	}
	m := sc.Metrics
	pTxn := uint(math.Round(float64(m.TotalTransactions) * m.PendingRate / 100))
	fTxn := uint(math.Round(float64(m.TotalTransactions) * m.FailRate / 100))
	return store.ScopeBreachedFromSnapshot(m.SuccessRate, m.PendingRate, m.FailRate, pTxn, fTxn, th)
}

func maintainedProvidersFromScopes(scopes []ScopeContext) map[string]bool {
	out := make(map[string]bool)
	for _, sc := range scopes {
		if sc.Maintenance != nil && len(sc.Maintenance.Active) > 0 {
			out[sc.ProviderCode] = true
		}
	}
	return out
}

func maintainedProvidersForPC(pc ProductContext) map[string]bool {
	if len(pc.MaintainedProviders) > 0 {
		return pc.MaintainedProviders
	}
	return maintainedProvidersFromScopes(pc.Scopes)
}

// SKURoutingDecision chooses routing vs maintenance for one SKU scope (§9.0).
// Maintenance when every routable provider (routing_pct > 0, không đang BT) breaches thresholds.
func SKURoutingDecision(pc ProductContext, sku string, routing map[string]float64, th store.ProductThreshold) (action, reason string) {
	maintained := maintainedProvidersForPC(pc)
	active := 0
	healthy := 0
	for provider, pct := range routing {
		if pct <= 0 {
			continue
		}
		if maintained[provider] {
			continue
		}
		active++
		sc := findScope(pc, sku, provider)
		if sc.Metrics == nil {
			continue
		}
		if !scopeMetricsBreached(sc, th) {
			healthy++
		}
	}
	if active == 0 {
		return "maintenance", "Không còn provider routing khả dụng ngoài bảo trì"
	}
	if active <= 1 {
		return "maintenance", fmt.Sprintf("Chỉ %d provider đang routing — không thể chuyển traffic", active)
	}
	if healthy == 0 {
		return "maintenance", "Tất cả provider đang routing đều vi phạm ngưỡng — đề xuất bảo trì SKU"
	}
	return "routing", fmt.Sprintf("%d provider routing; %d provider trong ngưỡng — có thể chuyển traffic", active, healthy)
}

func findScope(pc ProductContext, sku, provider string) ScopeContext {
	for _, sc := range pc.Scopes {
		if sc.SKUCode == sku && sc.ProviderCode == provider {
			return sc
		}
	}
	return ScopeContext{SKUCode: sku, ProviderCode: provider}
}

func breachedProvidersForSKU(pc ProductContext, sku string, th store.ProductThreshold) map[string]bool {
	out := make(map[string]bool)
	for _, sc := range pc.Scopes {
		if sc.SKUCode != sku {
			continue
		}
		if scopeMetricsBreached(sc, th) {
			out[sc.ProviderCode] = true
		}
	}
	return out
}
