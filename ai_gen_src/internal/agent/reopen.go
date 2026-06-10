package agent

import (
	"fmt"

	"opsone/internal/store"
)

// BuildReopenRoutingPlan proposes routing after admin Mở lại provider / Mở lại dịch vụ.
// Metrics in pc.Scopes must come from agent cycle history (or mock window for all providers),
// not dashboard snapshot that zeroes routing_pct=0 providers.
func BuildReopenRoutingPlan(
	pc ProductContext,
	baseline map[string]float64,
	th store.ProductThreshold,
	reason string,
) RoutingPlanJSON {
	current := make(map[string]float64)
	if pc.Routing.RoutingBySKU != nil {
		sku := ""
		for _, sc := range pc.Scopes {
			sku = sc.SKUCode
			break
		}
		if sku != "" {
			current = currentRouting(pc, ScopeContext{SKUCode: sku})
		}
	}
	if len(current) == 0 {
		for p, pct := range pc.Routing.Routing {
			current[p] = pct
		}
	}
	proposed := roundPctToInt100(copyPctMap(baseline))
	sku := ""
	if len(pc.Scopes) > 0 {
		sku = pc.Scopes[0].SKUCode
	}
	success := successByProvider(pc, ScopeContext{SKUCode: sku})
	exp := weightedSuccess(proposed, success)
	if reason == "" {
		reason = fmt.Sprintf("Mở lại scope — trả routing về baseline biz (metric chu kỳ Agent)")
	}
	return RoutingPlanJSON{
		Product:         pc.Product.ProductCode,
		Scope:           scopeName(pc),
		SKU:             sku,
		Reason:          reason,
		Current:         roundPctToInt100(copyPctMap(current)),
		Proposed:        proposed,
		SuccessRates:    success,
		ExpectedSuccess: exp,
	}
}
