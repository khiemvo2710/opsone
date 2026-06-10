package agent

import (
	"math"
	"sort"

	"opsone/internal/store"
)

// RoutingPlanJSON is stored in routing_plans.plan_json (§8.6.2).
type RoutingPlanJSON struct {
	Product         string             `json:"product"`
	Scope           string             `json:"scope"`
	SKU             string             `json:"sku,omitempty"`
	Reason          string             `json:"reason_vi"`
	Current         map[string]float64 `json:"current_pct"`
	Proposed        map[string]float64 `json:"proposed_pct"`
	SuccessRates    map[string]float64 `json:"success_rates"`
	ExpectedSuccess float64            `json:"expected_success_pct"`
}

// BuildRoutingPlan proposes hard cut for every breached provider on the SKU (§8.6.3 / §9.0).
func BuildRoutingPlan(pc ProductContext, badScope ScopeContext, reason string, th store.ProductThreshold) RoutingPlanJSON {
	current := currentRouting(pc, badScope)
	proposed := proposeShift(current, pc, badScope, th)
	success := successByProvider(pc, badScope)
	proposed = roundPctToInt100(proposed)
	enforceSingleHealthyFullRouting(proposed)
	exp := weightedSuccess(proposed, success)
	return RoutingPlanJSON{
		Product:         pc.Product.ProductCode,
		Scope:           scopeName(pc),
		SKU:             badScope.SKUCode,
		Reason:          reason,
		Current:         current,
		Proposed:        proposed,
		SuccessRates:    success,
		ExpectedSuccess: exp,
	}
}

func scopeName(pc ProductContext) string {
	if pc.Routing.Scope != "" {
		return pc.Routing.Scope
	}
	return "provider"
}

func currentRouting(pc ProductContext, s ScopeContext) map[string]float64 {
	out := make(map[string]float64)
	if pc.Routing.RoutingBySKU != nil && s.SKUCode != "" {
		for p, pct := range pc.Routing.RoutingBySKU[s.SKUCode] {
			out[p] = pct
		}
		return out
	}
	for p, pct := range pc.Routing.Routing {
		out[p] = pct
	}
	return out
}

func successByProvider(pc ProductContext, focus ScopeContext) map[string]float64 {
	out := make(map[string]float64)
	sku := focus.SKUCode
	for _, sc := range pc.Scopes {
		if sc.SKUCode != sku || sc.Metrics == nil {
			continue
		}
		out[sc.ProviderCode] = sc.Metrics.SuccessRate
	}
	return out
}

// proposeShift zeros every breached provider and redistributes traffic to healthy providers.
func proposeShift(current map[string]float64, pc ProductContext, bad ScopeContext, th store.ProductThreshold) map[string]float64 {
	breached := breachedProvidersForSKU(pc, bad.SKUCode, th)
	if len(breached) == 0 && bad.Threshold != nil && bad.Threshold.Breached {
		breached[bad.ProviderCode] = true
	}

	out := copyPctMap(current)
	var drop float64
	for p, pct := range current {
		if pct <= 0 || !breached[p] {
			continue
		}
		drop += pct
		out[p] = 0
	}
	if drop <= 0 {
		return out
	}
	return redistributeToHealthy(out, pc, bad, breached, drop)
}

func redistributeToHealthy(
	out map[string]float64,
	pc ProductContext,
	focus ScopeContext,
	breached map[string]bool,
	drop float64,
) map[string]float64 {
	if drop <= 0 {
		return out
	}
	sr := successByProvider(pc, focus)
	var good []string
	var goodSum float64
	maintained := maintainedProvidersForPC(pc)
	for p, pct := range out {
		if breached[p] || pct <= 0 || maintained[p] {
			continue
		}
		weight := sr[p]
		if weight <= 0 {
			weight = 1
		}
		good = append(good, p)
		goodSum += weight
	}
	if len(good) == 0 || goodSum == 0 {
		return out
	}
	remain := drop
	for i, p := range good {
		weight := sr[p]
		if weight <= 0 {
			weight = 1
		}
		add := drop * (weight / goodSum)
		if i == len(good)-1 {
			add = remain
		}
		out[p] += add
		remain -= add
	}
	normalizePct(out)
	return out
}

func weightedSuccess(pct, success map[string]float64) float64 {
	var sum float64
	for p, w := range pct {
		sum += w * success[p] / 100
	}
	return sum
}

func normalizePct(m map[string]float64) {
	var sum float64
	for _, v := range m {
		sum += v
	}
	if sum == 0 {
		return
	}
	for p := range m {
		m[p] = m[p] / sum * 100
	}
}

// roundPctToInt100 rounds each provider to whole % with sum exactly 100 (largest remainder).
func roundPctToInt100(m map[string]float64) map[string]float64 {
	if len(m) == 0 {
		return m
	}
	type entry struct {
		key       string
		base      int
		remainder float64
	}
	entries := make([]entry, 0, len(m))
	sum := 0
	for k, v := range m {
		if v < 0 {
			v = 0
		}
		base := int(math.Floor(v + 1e-9))
		entries = append(entries, entry{k, base, v - float64(base)})
		sum += base
	}
	if sum == 0 {
		return m
	}
	diff := 100 - sum
	sort.Slice(entries, func(i, j int) bool {
		if diff > 0 {
			return entries[i].remainder > entries[j].remainder
		}
		return entries[i].remainder < entries[j].remainder
	})
	n := len(entries)
	for i := 0; i < absInt(diff); i++ {
		idx := i % n
		if diff > 0 {
			entries[idx].base++
		} else if entries[idx].base > 0 {
			entries[idx].base--
		}
	}
	out := make(map[string]float64, len(entries))
	for _, e := range entries {
		out[e.key] = float64(e.base)
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func nonZeroRoutingProviders(m map[string]float64) (count int, sole string) {
	for p, v := range m {
		if v > 0 {
			count++
			sole = p
		}
	}
	return count, sole
}

// enforceSingleHealthyFullRouting keeps 100% on the only active provider after integer rounding.
func enforceSingleHealthyFullRouting(m map[string]float64) {
	if n, sole := nonZeroRoutingProviders(m); n == 1 {
		for p := range m {
			if p == sole {
				m[p] = 100
			} else {
				m[p] = 0
			}
		}
	}
}

func copyPctMap(m map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
