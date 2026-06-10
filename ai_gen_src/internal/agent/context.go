package agent

import (
	"time"

	"opsone/internal/domain"
	"opsone/internal/rules"
	"opsone/internal/threshold"
	"opsone/internal/tools"
)

// ScopeContext is one product×sku×provider unit ready for Reasoning (§3, Phase 3).
type ScopeContext struct {
	ProductCode  string `json:"product_code"`
	ServiceType  string `json:"service_type"`
	SKUCode      string `json:"sku_code"`
	ProviderCode string `json:"provider_code"`
	Skipped      bool   `json:"skipped"`
	SkipReason   string `json:"skip_reason,omitempty"`

	Metrics     *tools.GetMetricsOutput     `json:"metrics,omitempty"`
	TopErrors   *tools.GetTopErrorsOutput   `json:"top_errors,omitempty"`
	Revenue     *tools.GetRevenueOutput     `json:"revenue,omitempty"`
	Maintenance *tools.GetMaintenanceOutput `json:"maintenance,omitempty"`
	Threshold    *threshold.Result           `json:"threshold,omitempty"`
	State        string                      `json:"state"` // NORMAL | WARNING | INCIDENT
	RuleEvidence []rules.RuleResult          `json:"rule_evidence,omitempty"`
}

// ProductContext aggregates scopes for one product after routing_mode loop (§3).
type ProductContext struct {
	Product      domain.Product           `json:"product"`
	Providers    tools.GetProvidersOutput `json:"providers"`
	Routing      tools.GetRoutingOutput   `json:"routing"`
	Scopes       []ScopeContext           `json:"scopes"`
	// MaintainedProviders: providers with active maintenance — excluded from routing targets.
	MaintainedProviders map[string]bool `json:"maintained_providers,omitempty"`
	HealthStatus string                   `json:"health_status"` // green | yellow | red
	HealthSummary       string             `json:"health_summary"`
	State               string             `json:"state"`
	ProductRuleEvidence []rules.RuleResult `json:"product_rule_evidence,omitempty"`
	LastIncidentID      string             `json:"last_incident_id,omitempty"`
}

// CycleContext is the full pipeline input for Reasoning Engine (Phase 4).
type CycleContext struct {
	CycleID    uint64           `json:"cycle_id"`
	StartedAt  time.Time        `json:"started_at"`
	DataSource string           `json:"data_source"`
	Products   []ProductContext `json:"products"`
	HealthStatus string         `json:"health_status"`
	Decision   string           `json:"decision"`
}
