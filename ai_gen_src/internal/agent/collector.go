package agent

import (
	"context"
	"fmt"
	"strings"

	"opsone/internal/catalog"
	"opsone/internal/domain"
	"opsone/internal/notify"
	"opsone/internal/store"
	"opsone/internal/threshold"
	"opsone/internal/tools"
)

// Collector gathers tool outputs per routing_mode loop (§3 step 2).
type Collector struct {
	DB        *store.DB
	Tools     *tools.Registry
	Threshold *threshold.Evaluator
	Notify    *notify.Service
}

// NewCollector creates a context collector.
func NewCollector(db *store.DB, n *notify.Service) *Collector {
	return &Collector{
		DB:        db,
		Tools:     tools.NewRegistry(db, n),
		Threshold: &threshold.Evaluator{DB: db},
		Notify:    n,
	}
}

// CollectAll builds context for all enabled products (single catalog pass).
func (c *Collector) CollectAll(ctx context.Context, products []domain.Product, cycleID uint64, window string) ([]ProductContext, error) {
	allScopes, err := catalog.ListMetricScopes(ctx, c.DB)
	if err != nil {
		return nil, err
	}
	byProduct := make(map[string][]catalog.MetricScope)
	for _, sc := range allScopes {
		byProduct[sc.ProductCode] = append(byProduct[sc.ProductCode], sc)
	}

	var out []ProductContext
	for _, p := range products {
		pc, err := c.collectProductWithScopes(ctx, p, byProduct[p.ProductCode], cycleID, window)
		if err != nil {
			return nil, err
		}
		out = append(out, pc)
	}
	return out, nil
}

func (c *Collector) collectProductWithScopes(ctx context.Context, product domain.Product, scopes []catalog.MetricScope, cycleID uint64, window string) (ProductContext, error) {
	prov, err := c.Tools.GetProviders(ctx, tools.GetProvidersInput{Product: product.ProductCode})
	if err != nil {
		return ProductContext{}, fmt.Errorf("GetProviders %s: %w", product.ProductCode, err)
	}
	routing, err := c.Tools.GetRouting(ctx, tools.GetRoutingInput{Product: product.ProductCode})
	if err != nil {
		return ProductContext{}, fmt.Errorf("GetRouting %s: %w", product.ProductCode, err)
	}
	th, err := c.DB.GetProductThreshold(ctx, product.ProductCode)
	if err != nil {
		return ProductContext{}, err
	}

	var scopeCtxs []ScopeContext
	for _, sc := range scopes {
		// Determine current routing pct for this provider/SKU so collectScope
		// can skip breach evaluation when the provider has 0% traffic.
		routingPct := providerRoutingPct(routing, sc.SKUCode, sc.ProviderCode)
		sctx, err := c.collectScope(ctx, sc, cycleID, window, th, routingPct)
		if err != nil {
			return ProductContext{}, err
		}
		scopeCtxs = append(scopeCtxs, sctx)
	}
	if err := c.applyRecoveryOverlay(ctx, product.ProductCode, cycleID, scopeCtxs); err != nil {
		return ProductContext{}, err
	}
	pc := ProductContext{Product: product, Providers: prov, Routing: routing, Scopes: scopeCtxs}
	pc.HealthStatus, pc.HealthSummary, pc.State = aggregateProductHealth(pc)
	return pc, nil
}

// CollectProduct builds ProductContext for one enabled product.
func (c *Collector) CollectProduct(ctx context.Context, product domain.Product, cycleID uint64, window string) (ProductContext, error) {
	allScopes, err := catalog.ListMetricScopes(ctx, c.DB)
	if err != nil {
		return ProductContext{}, err
	}
	var scopes []catalog.MetricScope
	for _, sc := range allScopes {
		if sc.ProductCode == product.ProductCode {
			scopes = append(scopes, sc)
		}
	}
	return c.collectProductWithScopes(ctx, product, scopes, cycleID, window)
}

func (c *Collector) collectScope(ctx context.Context, sc catalog.MetricScope, cycleID uint64, window string, th store.ProductThreshold, routingPct float64) (ScopeContext, error) {
	out := ScopeContext{
		ProductCode:  sc.ProductCode,
		ServiceType:  string(sc.ServiceType),
		SKUCode:      sc.SKUCode,
		ProviderCode: sc.ProviderCode,
		State:        "NORMAL",
	}

	maint, err := c.Tools.GetMaintenance(ctx, tools.GetMaintenanceInput{
		Product: sc.ProductCode, Provider: sc.ProviderCode, SKU: sc.SKUCode,
	})
	if err != nil {
		return ScopeContext{}, err
	}
	out.Maintenance = &maint
	if len(maint.Active) > 0 {
		out.Skipped = true
		out.SkipReason = "maintenance_active"
		out.State = "MAINTENANCE_ACTIVE"
		return out, nil
	}

	// Skip breach evaluation for providers with 0% routing — they carry no live traffic
	// so their metrics cannot affect service quality. Creating incidents for them is noise.
	if routingPct == 0 {
		out.Skipped = true
		out.SkipReason = "zero_routing"
		return out, nil
	}

	metrics, err := c.Tools.GetMetrics(ctx, tools.GetMetricsInput{
		Product: sc.ProductCode, Provider: sc.ProviderCode, SKU: sc.SKUCode, Window: window,
	})
	if err != nil {
		if isNoDataErr(err) {
			out.Skipped = true
			out.SkipReason = "no_data"
			return out, nil
		}
		return ScopeContext{}, err
	}
	out.Metrics = &metrics

	topErrs, err := c.Tools.GetTopErrors(ctx, tools.GetTopErrorsInput{
		Product: sc.ProductCode, Provider: sc.ProviderCode, SKU: sc.SKUCode, Window: window, Limit: 5,
	})
	if err == nil {
		out.TopErrors = &topErrs
	}

	rev, err := c.Tools.GetRevenue(ctx, tools.GetRevenueInput{
		Product: sc.ProductCode, Provider: sc.ProviderCode, SKU: sc.SKUCode, Window: window,
	})
	if err == nil {
		out.Revenue = &rev
	}

	prior, err := c.DB.CountConsecutiveBreaches(ctx, sc.ProductCode, sc.SKUCode, sc.ProviderCode, cycleID, th)
	if err != nil {
		return ScopeContext{}, err
	}
	consecutive := prior
	settings, err := c.DB.GetAgentSettings(ctx)
	if err != nil {
		return ScopeContext{}, err
	}
	errEvents, _ := c.DB.SumErrorEventsAtRecordedAt(ctx, settings.DataSource, sc.ProductCode, sc.SKUCode, sc.ProviderCode, metrics.RecordedAt)
	if store.ScopeBreached(metrics.SuccessRate, metrics.PendingRate, metrics.FailRate, metrics.TotalTransactions, errEvents, th) {
		consecutive++
	}

	eval, err := c.Threshold.EvaluateThresholds(ctx, threshold.MetricInput{
		Product: sc.ProductCode, Provider: sc.ProviderCode, SKU: sc.SKUCode,
		SuccessRate: metrics.SuccessRate, PendingRate: metrics.PendingRate, FailRate: metrics.FailRate,
		TotalTxn: metrics.TotalTransactions, ErrorEvents: errEvents,
	}, consecutive)
	if err != nil {
		return ScopeContext{}, err
	}
	out.Threshold = &eval
	out.State = scopeStateFromThreshold(eval)
	return out, nil
}

func (c *Collector) applyRecoveryOverlay(ctx context.Context, productCode string, cycleID uint64, scopes []ScopeContext) error {
	skuCycles := make(map[string]int)
	for i := range scopes {
		sku := scopes[i].SKUCode
		if _, ok := skuCycles[sku]; ok {
			continue
		}
		applyCycle, ok, err := c.DB.GetRecoveryApplyCycle(ctx, productCode, sku)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		n, err := c.DB.CountAnalysisCyclesSince(ctx, applyCycle, cycleID)
		if err != nil {
			return err
		}
		skuCycles[sku] = n
	}

	var clearSKUs []string
	seenClear := make(map[string]bool)
	for i := range scopes {
		if scopes[i].Skipped {
			continue
		}
		cycles, tracked := skuCycles[scopes[i].SKUCode]
		if !tracked {
			continue
		}
		breached := scopes[i].Threshold != nil && scopes[i].Threshold.Breached
		newState, clear := overlayRecoveryState(scopes[i].State, breached, cycles)
		scopes[i].State = newState
		if clear && !seenClear[scopes[i].SKUCode] {
			seenClear[scopes[i].SKUCode] = true
			clearSKUs = append(clearSKUs, scopes[i].SKUCode)
		}
	}
	for _, sku := range clearSKUs {
		if err := c.DB.ClearRecoveryStart(ctx, productCode, sku); err != nil {
			return err
		}
	}
	return nil
}

func scopeStateFromThreshold(r threshold.Result) string {
	if !r.Breached {
		return "NORMAL"
	}
	if r.ConsecutiveBreachCycles >= r.RequiredCycles {
		return "INCIDENT"
	}
	return "WARNING"
}

func aggregateProductHealth(pc ProductContext) (healthStatus, summary, state string) {
	healthStatus = "green"
	state = "NORMAL"
	var issues []string

	for _, s := range pc.Scopes {
		if s.Skipped {
			continue
		}
		switch s.State {
		case "INCIDENT":
			healthStatus = "red"
			state = "INCIDENT"
			if s.Threshold != nil && len(s.Threshold.BreachReasons) > 0 {
				issues = append(issues, fmt.Sprintf("%s/%s: %s", scopeLabel(s), s.ProviderCode, s.Threshold.BreachReasons[0]))
			}
		case "RECOVERING":
			if healthStatus != "red" {
				healthStatus = "yellow"
				state = "RECOVERING"
			}
			issues = append(issues, fmt.Sprintf("%s/%s đang hồi phục sau routing", scopeLabel(s), s.ProviderCode))
		case "WARNING":
			if healthStatus != "red" {
				healthStatus = "yellow"
				state = "WARNING"
			}
			issues = append(issues, fmt.Sprintf("%s/%s vượt ngưỡng", scopeLabel(s), s.ProviderCode))
		case "MAINTENANCE_ACTIVE":
			if healthStatus == "green" {
				healthStatus = "yellow"
			}
		}
	}

	if len(issues) == 0 {
		return healthStatus, "Ổn định", state
	}
	if len(issues) > 2 {
		summary = fmt.Sprintf("%s — %d scope cần theo dõi", productDisplayName(pc), len(issues))
	} else {
		summary = strings.Join(issues, "; ")
	}
	return healthStatus, summary, state
}

func scopeLabel(s ScopeContext) string {
	if s.SKUCode != "" {
		return s.SKUCode
	}
	return s.ProductCode
}

func isNoDataErr(err error) bool {
	if te, ok := err.(*tools.ToolError); ok {
		return te.Code == "no_data"
	}
	return false
}

// providerRoutingPct returns the current traffic_pct for a given provider+SKU
// from the product's routing output. Returns 0 if not found.
func providerRoutingPct(routing tools.GetRoutingOutput, sku, provider string) float64 {
	if sku == "" {
		// provider-level routing (topup products)
		return routing.Routing[provider]
	}
	if skuMap, ok := routing.RoutingBySKU[sku]; ok {
		return skuMap[provider]
	}
	return 0
}
