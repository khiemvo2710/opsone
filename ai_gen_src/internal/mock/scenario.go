package mock

import (
	"math"
	"math/rand"
)

// Rates holds success/pending/fail percentages (sum ≈ 100).
type Rates struct {
	Success float64
	Pending float64
	Fail    float64
}

func (r Rates) normalized() Rates {
	s := r.Success + r.Pending + r.Fail
	if s <= 0 {
		return Rates{Success: 97, Pending: 2, Fail: 1}
	}
	return Rates{
		Success: round2(r.Success / s * 100),
		Pending: round2(r.Pending / s * 100),
		Fail:    round2(r.Fail / s * 100),
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// baselineRates returns stable rates with ±2% noise (§4.5.2).
func baselineRates(rng *rand.Rand) Rates {
	success := 97.0 + rng.Float64()*4 - 2 // 95-99
	fail := 1.0 + rng.Float64()*2         // 1-3
	pending := 100 - success - fail
	if pending < 0 {
		pending = 0
	}
	return Rates{Success: success, Pending: pending, Fail: fail}.normalized()
}

// scenarioKey identifies a scope for degradation state.
func scenarioKey(product, sku, provider string) string {
	return product + "|" + sku + "|" + provider
}

// computeRates applies scenario to a scope; state map persists ESALE degradation.
func computeRates(scenario string, product, sku, provider string, rng *rand.Rand, state map[string]float64) Rates {
	switch scenario {
	case "esale_degrading":
		if isDegradeTarget(product, sku, provider) {
			key := scenarioKey(product, sku, provider)
			cur, ok := state[key]
			if !ok {
				cur = 97.0
			}
			drop := 3.0 + rng.Float64()*2 // 3-5%
			cur = math.Max(50, cur-drop)
			state[key] = cur
			fail := math.Min(40, 100-cur-2)
			pending := 100 - cur - fail
			if pending < 0 {
				pending = 0
			}
			return Rates{Success: round2(cur), Pending: round2(pending), Fail: round2(fail)}.normalized()
		}
		return baselineRates(rng)
	default:
		return baselineRates(rng)
	}
}

func isDegradeTarget(product, sku, provider string) bool {
	if provider != "ESALE" {
		return false
	}
	if product == "TOPUP_VINA" && sku == "" {
		return true
	}
	if product == "ZING" && sku == "20000" {
		return true
	}
	return false
}

// ComputeRatesForTest exposes scenario logic for unit tests.
func ComputeRatesForTest(scenario string, product, sku, provider string, rng *rand.Rand, state map[string]float64) Rates {
	return computeRates(scenario, product, sku, provider, rng, state)
}

// errorCounts derives mock error stats from fail rate.
func errorCounts(failRate float64, totalTxn uint, rng *rand.Rand) (code3004, code22 uint) {
	failTxn := uint(float64(totalTxn) * failRate / 100)
	if failTxn == 0 {
		return 0, 0
	}
	split := 0.6 + rng.Float64()*0.3
	code3004 = uint(float64(failTxn) * split)
	code22 = failTxn - code3004
	return code3004, code22
}
