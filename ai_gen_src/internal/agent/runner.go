package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"opsone/internal/catalog"
	"opsone/internal/config"
	"opsone/internal/domain"
	"opsone/internal/notify"
	"opsone/internal/store"
)

// Runner executes Agent Core pipeline: collect context per routing_mode (§3, Phase 3).
type Runner struct {
	DB           *store.DB
	Collector    *Collector
	Notify       *notify.Service
	DashboardURL string // Public URL for email deep-links (from DASHBOARD_URL env)
}

// NewRunner creates the agent core runner.
// n and cfg are optional (nil/zero-value safe) — useful in tests.
func NewRunner(db *store.DB, args ...any) *Runner {
	var n *notify.Service
	var dashURL = "http://localhost:5173"
	for _, a := range args {
		switch v := a.(type) {
		case *notify.Service:
			n = v
		case notify.Service:
			n = &v
		case config.Config:
			if v.DashboardURL != "" {
				dashURL = v.DashboardURL
			}
		}
	}
	return &Runner{DB: db, Collector: NewCollector(db, n), Notify: n, DashboardURL: dashURL}
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

	reasoner := NewReasoner(r.DB, r.Notify, r.DashboardURL)
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

	// Send notifications for breaches
	go func() {
		if r.Notify == nil {
			return
		}
		for _, pc := range productCtxs {
			// Check product-level threshold
			th, _ := r.DB.GetProductThreshold(context.Background(), pc.Product.ProductCode)
			
			for _, sc := range pc.Scopes {
				if sc.Threshold == nil || !sc.Threshold.Breached {
					continue
				}
				
				if th.AlertEmailEnabled {
					action := "Đang theo dõi — chưa đủ điều kiện can thiệp tự động"
					if sc.Threshold.ShouldAct {
						action = "Đang xử lý: " + sc.Threshold.SuggestedAction
					}
					deepLink := fmt.Sprintf("%s/dashboard?product=%s", r.DashboardURL, pc.Product.ProductCode)
					// Hourly dedup bucket: at most 1 breach email per scope per hour.
					hourBucket := time.Now().UTC().Format("2006010215")
					dedupeKey := fmt.Sprintf("%s:%s:%s:breach:%s", pc.Product.ProductCode, sc.ProviderCode, sc.SKUCode, hourBucket)

					_ = r.Notify.SendIfNeeded(context.Background(), notify.EmailParams{
						Product:       pc.Product.Label,
						Provider:      sc.ProviderCode,
						SKU:           sc.SKUCode,
						HealthStatus:  pc.HealthStatus,
						TriggerEvent:  "breach",
						SuccessRate:   sc.Metrics.SuccessRate,
						PendingRate:   sc.Metrics.PendingRate,
						FailRate:      sc.Metrics.FailRate,
						BreachReasons: sc.Threshold.BreachReasons,
						ActionSummary: action,
						CycleID:       &cycleID,
						DeepLinkURL:   deepLink,
						DedupeKey:     dedupeKey,
					})
				}
			}
		}
	}()

	if err := r.DB.FinishAnalysisCycleFull(ctx, cycleID, time.Now(), cycleHealth, cycleSummary, decision); err != nil {
		return result, err
	}

	// Baseline anomaly check (log only — no breach override)
	go func() {
		bctx, bcancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer bcancel()
		for _, pc := range productCtxs {
			for _, sc := range pc.Scopes {
				if sc.Metrics == nil || sc.Skipped {
					continue
				}
				baseline, err := r.DB.GetBaseline(bctx, sc.ProductCode, sc.SKUCode, sc.ProviderCode)
				if err != nil || baseline == nil {
					continue
				}
				dev, ok := store.BaselineDeviation(sc.Metrics.SuccessRate, baseline)
				if ok && dev >= 3.0 {
					log.Printf("ANOMALY %s/%s/%s: success=%.1f%% baseline=%.1f%% dev=%.1f σ",
						sc.ProductCode, sc.SKUCode, sc.ProviderCode,
						sc.Metrics.SuccessRate, baseline.AvgSuccessRate, dev)
				}
			}
		}
	}()

	// Auto-recovery verification
	go r.checkPendingRecoveries(context.Background())

	// Baseline aggregation: run every 60 cycles (≈ every hour when interval=1min)
	if cycleID%60 == 0 {
		go func() {
			bctx, bcancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer bcancel()
			job := &BaselineJob{DB: r.DB}
			job.RunOnce(bctx)
		}()
	}

	log.Printf("agent core: cycle=%d products=%d history=%d health=%s decision=%s",
		cycleID, len(productCtxs), len(historyRows), cycleHealth, decision)
	return result, nil
}

// checkPendingRecoveries verifies pending routing action outcomes.
func (r *Runner) checkPendingRecoveries(ctx context.Context) {
	pending, err := r.DB.ListPendingVerify(ctx)
	if err != nil || len(pending) == 0 {
		return
	}
	for _, v := range pending {
		postSuccess, postPending, found := r.DB.FindCurrentMetricsForScope(ctx, v.ProductCode, v.SKUCode)
		if !found {
			continue
		}
		status := "no_change"
		if postSuccess-v.PreSuccessRate >= 3.0 {
			status = "improved"
		} else if v.PreSuccessRate-postSuccess >= 5.0 {
			status = "degraded"
		}
		escalated := status == "degraded"
		if err := r.DB.UpdateVerifyResult(ctx, v.ID, postSuccess, postPending, status, escalated); err != nil {
			log.Printf("recovery_verify: update error: %v", err)
			continue
		}
		if escalated {
			log.Printf("RECOVERY_FAILED: %s/%s post=%.1f%% pre=%.1f%% — escalating",
				v.ProductCode, v.SKUCode, postSuccess, v.PreSuccessRate)
			_ = r.Notify.SendIfNeeded(ctx, notify.EmailParams{
				Product:       v.ProductCode,
				SKU:           v.SKUCode,
				HealthStatus:  "red",
				TriggerEvent:  "recovery_failed",
				SuccessRate:   postSuccess,
				PendingRate:   postPending,
				ActionSummary: fmt.Sprintf("Routing thay đổi KHÔNG cải thiện: thành công %.0f%% → %.0f%%", v.PreSuccessRate, postSuccess),
				DedupeKey:     fmt.Sprintf("recovery_fail:%s:%s:%d", v.ProductCode, v.SKUCode, v.ID),
			})
		}
	}
}

// RunBlocking runs scheduler until ctx cancelled; skips overlapping runs.
func (r *Runner) RunBlocking(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 1 * time.Minute
	}

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

	// Run immediately on startup, then re-read interval from DB each tick.
	run()
	for {
		// Re-read interval from DB so UI changes take effect without restart.
		if settings, err := r.DB.GetAgentSettings(ctx); err == nil {
			if d := IntervalFromSettings(settings); d > 0 {
				interval = d
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
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
			redProducts = append(redProducts, catalog.ProductDisplayLabel(p.Product))
		case "yellow":
			if health != "red" {
				health = "yellow"
				decision = "warning"
			}
		}
	}

	if len(redProducts) > 0 {
		summary = fmt.Sprintf("Sự cố: %s", strings.Join(redProducts, ", "))
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
