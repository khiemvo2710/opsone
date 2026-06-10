package mock_test

import (
	"math/rand"
	"testing"

	"opsone/internal/mock"
)

func TestEsaleDegradingDropsSuccess(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	state := make(map[string]float64)

	var first, last float64
	for i := 0; i < 5; i++ {
		r := computeRatesExported("esale_degrading", "TOPUP_VINA", "", "ESALE", rng, state)
		if i == 0 {
			first = r.Success
		}
		last = r.Success
	}
	if last >= first {
		t.Errorf("expected success to drop: first=%.2f last=%.2f", first, last)
	}
}

func TestNormalScenarioStable(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	state := make(map[string]float64)
	r := computeRatesExported("normal", "ZING", "20000", "IMEDIA", rng, state)
	if r.Success < 90 || r.Success > 100 {
		t.Errorf("success out of range: %.2f", r.Success)
	}
}

// computeRatesExported wraps unexported computeRates via scenario.go test hook.
func computeRatesExported(scenario, product, sku, provider string, rng *rand.Rand, state map[string]float64) mock.Rates {
	return mock.ComputeRatesForTest(scenario, product, sku, provider, rng, state)
}
