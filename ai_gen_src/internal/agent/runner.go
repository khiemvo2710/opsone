package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"opsone/internal/domain"
	"opsone/internal/store"
)

// Runner executes Agent Core pipeline: collect context per routing_mode (§3, Phase 3).
type Runner struct {
	DB        *store.DB
	Collector *Collector
}

// NewRunner creates the agent core runner.
func NewRunner(db *store.DB) *Runner {
	return &Runner{DB: db, Collector: NewCollector(db)}
}

// RunOnce executes one agent cycle: collect → persist history/health/state.
func (r *Runner) RunOnce(ctx context.Context) (CycleContext, error) {
	settings, err := r.DB.GetAgentSettings(ctx)
	if err != nil {
		return CycleContext{}, err
	}
	if !settings.SchedulerEnabled {
		log.Println("agent: scheduler disabled, skip")
		return CycleContext{}, nil
	}

	now := time.Now()
	cycleID, err := r.DB.StartAnalysisCycle(ctx, now, settings.DataSource)
	if err != nil {
		return CycleContext{}, err
	}

	products, err := r.DB.ListProducts(ctx, true)
	if err != nil {
		_ = r.DB.FailAnalysisCycle(ctx, cycleID, time.Now())
		return CycleContext{}, err
	}

	windowMin := 15
	if len(products) > 0 {
		if th, err := r.DB.GetProductThreshold(ctx, products[0].ProductCode); err == nil && th.MetricsWindowMin > 0 {
			windowMin = th.MetricsWindowMin
		}
	}
	window := fmt.Sprintf("%dm", windowMin)

	productCtxs, err := r.Collector.CollectAll(ctx, products, cycleID, window)
	if err != nil {
		_ = r.DB.FailAnalysisCycle(ctx, cycleID, time.Now())
		return CycleContext{}, err
	}

	reasoner := NewReasoner(r.DB)
	productCtxs, err = reasoner.Process(ctx, cycleID, productCtxs)
	if err != nil {
		_ = r.DB.FailAnalysisCycle(ctx, cycleID, time.Now())
		return CycleContext{}, fmt.Errorf("reasoning: %w", err)
	}

	var historyRows []store.AnalysisHistoryRow
	var healthRows []store.HealthStatusRow
	var stateRows []store.AgentStateRow

	for _, pc := range productCtxs {
		healthRows = append(healthRows, store.HealthStatusRow{
			ProductCode:   pc.Product.ProductCode,
			HealthStatus:  pc.HealthStatus,
			HealthSummary: pc.HealthSummary,
		})
		stateRows = append(stateRows, buildStateRows(pc)...)

		for _, s := range pc.Scopes {
			if s.Skipped || s.Metrics == nil {
				continue
			}
			historyRows = append(historyRows, store.AnalysisHistoryRow{
				RecordedAt:        now,
				ProductCode:       s.ProductCode,
				ServiceType:       s.ServiceType,
				SKUCode:           s.SKUCode,
				ProviderCode:      s.ProviderCode,
				SuccessRate:       s.Metrics.SuccessRate,
				PendingRate:       s.Metrics.PendingRate,
				FailRate:          s.Metrics.FailRate,
				TotalTransactions: s.Metrics.TotalTransactions,
			})
		}
	}

	if err := r.DB.InsertAnalysisHistory(ctx, cycleID, historyRows); err != nil {
		_ = r.DB.FailAnalysisCycle(ctx, cycleID, time.Now())
		return CycleContext{}, err
	}
	if err := r.DB.InsertHealthStatusProducts(ctx, cycleID, healthRows); err != nil {
		_ = r.DB.FailAnalysisCycle(ctx, cycleID, time.Now())
		return CycleContext{}, err
	}
	if err := r.DB.InsertAgentStateHistory(ctx, cycleID, stateRows); err != nil {
		_ = r.DB.FailAnalysisCycle(ctx, cycleID, time.Now())
		return CycleContext{}, err
	}

	cycleHealth, cycleSummary, decision := aggregateCycleHealth(productCtxs)
	result := CycleContext{
		CycleID:      cycleID,
		StartedAt:    now,
		DataSource:   settings.DataSource,
		Products:     productCtxs,
		HealthStatus: cycleHealth,
		Decision:     decision,
	}

	if err := r.DB.FinishAnalysisCycleFull(ctx, cycleID, time.Now(), cycleHealth, cycleSummary, decision); err != nil {
		return result, err
	}

	log.Printf("agent core: cycle=%d products=%d history=%d health=%s decision=%s",
		cycleID, len(productCtxs), len(historyRows), cycleHealth, decision)
	return result, nil
}

// RunBlocking runs scheduler until ctx cancelled; skips overlapping runs.
func (r *Runner) RunBlocking(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	running := make(chan struct{}, 1)
	run := func() {
		select {
		case running <- struct{}{}:
		default:
			log.Println("agent: previous cycle still running, skip tick")
			return
		}
		defer func() { <-running }()

		if _, err := r.RunOnce(ctx); err != nil {
			log.Printf("agent: cycle error: %v", err)
		}
	}

	run()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			run()
		}
	}
}

func buildStateRows(pc ProductContext) []store.AgentStateRow {
	if pc.Product.RoutingMode == domain.RoutingSKU {
		bySKU := map[string]string{}
		bySKUSummary := map[string]string{}
		for _, s := range pc.Scopes {
			st := s.State
			if st == "MAINTENANCE_ACTIVE" {
				st = "MAINTENANCE_ACTIVE"
			}
			if prev, ok := bySKU[s.SKUCode]; !ok || stateRank(st) > stateRank(prev) {
				bySKU[s.SKUCode] = st
				bySKUSummary[s.SKUCode] = fmt.Sprintf("%s/%s", s.SKUCode, s.ProviderCode)
			}
		}
		var rows []store.AgentStateRow
		for sku, st := range bySKU {
			rows = append(rows, store.AgentStateRow{
				ProductCode:      pc.Product.ProductCode,
				SKUCode:          sku,
				State:            mapStateToDB(st),
				TransitionReason: pc.HealthSummary,
			})
		}
		if len(rows) > 0 {
			return rows
		}
	}
	return []store.AgentStateRow{{
		ProductCode:      pc.Product.ProductCode,
		SKUCode:          "",
		State:            mapStateToDB(pc.State),
		TransitionReason: pc.HealthSummary,
	}}
}

func stateRank(s string) int {
	switch s {
	case "INCIDENT":
		return 3
	case "WARNING":
		return 2
	case "RECOVERING":
		return 2
	case "MAINTENANCE_ACTIVE":
		return 2
	default:
		return 1
	}
}

func aggregateCycleHealth(products []ProductContext) (health, summary, decision string) {
	health = "green"
	decision = "monitor"
	var redProducts []string

	for _, p := range products {
		switch p.HealthStatus {
		case "red":
			health = "red"
			decision = "incident"
			redProducts = append(redProducts, p.Product.ProductCode)
		case "yellow":
			if health != "red" {
				health = "yellow"
				decision = "warning"
			}
		}
	}

	if len(redProducts) > 0 {
		summary = fmt.Sprintf("Sự cố: %v", redProducts)
	} else if health == "yellow" {
		summary = "Một số sản phẩm vượt ngưỡng — đang theo dõi"
	} else {
		summary = "Hệ thống ổn định"
	}
	return health, summary, decision
}

func mapStateToDB(state string) string {
	switch state {
	case "INCIDENT":
		return "INCIDENT"
	case "WARNING":
		return "WARNING"
	case "RECOVERING":
		return "RECOVERING"
	case "MAINTENANCE_ACTIVE":
		return "MAINTENANCE_ACTIVE"
	default:
		return "NORMAL"
	}
}
