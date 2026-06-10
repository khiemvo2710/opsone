package rules

import "opsone/internal/store"

// RuleResult is one rule evaluation output (§7.3.1).
type RuleResult struct {
	RuleID          string         `json:"rule_id"`
	Triggered       bool           `json:"triggered"`
	Tag             string         `json:"tag"`
	MessageVi       string         `json:"message_vi"`
	SeverityBump    int            `json:"severity_bump"`
	SuggestedAction string         `json:"suggested_action,omitempty"`
	Evidence        map[string]any `json:"evidence,omitempty"`
}

// ScopeData is metric snapshot for rule evaluation (decoupled from agent package).
type ScopeData struct {
	ProductCode         string
	ServiceType         string
	SKUCode             string
	ProviderCode        string
	SuccessRate         float64
	PendingRate         float64
	FailRate            float64
	RevenueLastHour     uint64
	Breached            bool
	HealthyBackupCount  int
	TopErrorCode        string
	TopErrorCount       uint
}

// ProductData for SKU-level rules.
type ProductData struct {
	ServiceType  string
	RoutingMode  string
	Scopes       []ScopeData
	RoutingBySKU map[string]map[string]float64
	Routing      map[string]float64
}

// ScopeInput is input for per-scope rules (R1–R6, R9).
type ScopeInput struct {
	Scope         ScopeData
	History       []store.ScopeHistoryPoint
	TrafficPct    float64
	SuccessMinPct float64
	FailMaxPct    float64
	GoodThreshold float64
	RevenueMinVND uint64
}

// ProductInput is input for product-level SKU rules (R7, R8).
type ProductInput struct {
	Product       ProductData
	SuccessMinPct float64
	ScopeEvidence []RuleResult
}
