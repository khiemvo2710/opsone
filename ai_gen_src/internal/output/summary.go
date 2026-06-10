package output

import (
	"opsone/internal/config"
	"opsone/internal/rules"
)

// SummarizeIncident returns incident summary; uses template fallback when LLM unavailable (§7.6).
func SummarizeIncident(cfg config.Config, product, provider, sku string, success, fail float64, evidence []rules.RuleResult, breachReasons []string) (summary, headline string) {
	if cfg.LLMAPIURL != "" {
		// Phase 4: template fallback; LLM integration in later phase when API configured.
	}
	return IncidentSummary(product, provider, sku, success, fail, evidence, breachReasons)
}
