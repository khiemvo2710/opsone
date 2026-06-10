package threshold

import (
	"context"
	"fmt"
	"math"
	"time"

	"opsone/internal/store"
)

// MetricInput scope metrics for threshold evaluation.
type MetricInput struct {
	Product      string
	Provider     string
	SKU          string
	SuccessRate  float64
	PendingRate  float64
	FailRate     float64
	TotalTxn     uint
	ErrorEvents  uint
}

// Evaluator compares metrics vs product thresholds (§7.4).
type Evaluator struct {
	DB *store.DB
}

// Result is EvaluateThresholds output (§7.4).
type Result struct {
	ProductCode               string   `json:"product_code"`
	Breached                  bool     `json:"breached"`
	BreachReasons             []string `json:"breach_reasons"`
	ConsecutiveBreachCycles   int      `json:"consecutive_breach_cycles"`
	RequiredCycles            int      `json:"required_cycles"`
	ShouldAct                 bool     `json:"should_act"`
	ShouldAlertEmail          bool     `json:"should_alert_email"`
	ShouldAlertMode           bool     `json:"should_alert_mode"`
	ActiveProviderCount       int      `json:"active_provider_count"`
	HealthyBackupCount        int      `json:"healthy_backup_count"`
	SuggestedAction           string   `json:"suggested_action"`
	SuggestedActionReason     string   `json:"suggested_action_reason"`
}

// EvaluateThresholds checks metrics against product thresholds and suggests action (§7.4).
func (e *Evaluator) EvaluateThresholds(ctx context.Context, in MetricInput, consecutiveBreach int) (Result, error) {
	th, err := e.DB.GetProductThreshold(ctx, in.Product)
	if err != nil {
		return Result{}, err
	}
	if !th.Enabled {
		return Result{ProductCode: in.Product, Breached: false}, nil
	}

	failTxn := uint(math.Round(float64(in.TotalTxn) * in.FailRate / 100))
	pendingTxn := uint(math.Round(float64(in.TotalTxn) * in.PendingRate / 100))
	reasons := breachReasons(in, th, failTxn, pendingTxn)

	activeCount, err := e.DB.CountActiveProviders(ctx, in.Product)
	if err != nil {
		return Result{}, err
	}

	healthyBackup, err := e.countHealthyBackups(ctx, in, th)
	if err != nil {
		return Result{}, err
	}

	breached := len(reasons) > 0
	res := Result{
		ProductCode:             in.Product,
		Breached:                breached,
		BreachReasons:           reasons,
		ConsecutiveBreachCycles: consecutiveBreach,
		RequiredCycles:          th.ConsecutiveCyclesRequired,
		ActiveProviderCount:     activeCount,
		HealthyBackupCount:      healthyBackup,
		ShouldAlertEmail:        breached && th.AlertEmailEnabled,
		ShouldAlertMode:         breached,
	}

	if breached {
		required := th.ConsecutiveCyclesRequired
		if required <= 0 {
			required = 2
		}
		res.RequiredCycles = required
		res.ConsecutiveBreachCycles = consecutiveBreach
		res.SuggestedAction, res.SuggestedActionReason = suggestAction(activeCount, healthyBackup)
		if consecutiveBreach >= required {
			res.ShouldAct = true
		}
	}
	return res, nil
}

func breachReasons(in MetricInput, th store.ProductThreshold, failTxn, pendingTxn uint) []string {
	var reasons []string
	if in.SuccessRate <= th.SuccessRateMinPct {
		reasons = append(reasons, fmt.Sprintf("Tỷ lệ thành công %.1f%% dưới ngưỡng %.1f%%", in.SuccessRate, th.SuccessRateMinPct))
	}
	if in.PendingRate >= th.PendingRateMaxPct {
		reasons = append(reasons, fmt.Sprintf("Tỷ lệ pending %.1f%% vượt ngưỡng %.1f%%", in.PendingRate, th.PendingRateMaxPct))
	}
	if in.FailRate >= th.FailRateMaxPct {
		reasons = append(reasons, fmt.Sprintf("Tỷ lệ lỗi %.1f%% vượt ngưỡng %.1f%%", in.FailRate, th.FailRateMaxPct))
	}
	if th.FailTxnCountMax > 0 && failTxn >= th.FailTxnCountMax {
		reasons = append(reasons, fmt.Sprintf("Số GD fail %d vượt ngưỡng %d", failTxn, th.FailTxnCountMax))
	}
	if th.PendingTxnCountMax > 0 && pendingTxn >= th.PendingTxnCountMax {
		reasons = append(reasons, fmt.Sprintf("Số GD pending %d vượt ngưỡng %d", pendingTxn, th.PendingTxnCountMax))
	}
	if in.ErrorEvents > th.ErrorEventCountMax {
		reasons = append(reasons, fmt.Sprintf("Số sự kiện lỗi %d vượt ngưỡng %d", in.ErrorEvents, th.ErrorEventCountMax))
	}
	return reasons
}

func suggestAction(activeCount, healthyBackup int) (action, reason string) {
	if activeCount <= 1 {
		return "maintenance", fmt.Sprintf("Chỉ %d provider active — không thể routing", activeCount)
	}
	if healthyBackup >= 1 {
		return "routing", fmt.Sprintf("%d provider active; %d backup healthy — có thể chuyển traffic", activeCount, healthyBackup)
	}
	return "maintenance", "Nhiều provider active nhưng không có backup healthy"
}

func (e *Evaluator) countHealthyBackups(ctx context.Context, in MetricInput, th store.ProductThreshold) (int, error) {
	routing, err := e.DB.GetRoutingForScope(ctx, in.Product, in.SKU)
	if err != nil {
		return 0, err
	}
	settings, err := e.DB.GetAgentSettings(ctx)
	if err != nil {
		return 0, err
	}
	windowMin := th.MetricsWindowMin
	if windowMin <= 0 {
		windowMin = 15
	}
	since := time.Now().Add(-time.Duration(windowMin) * time.Minute)
	count := 0
	for _, row := range routing {
		if row.ProviderCode == in.Provider || row.TrafficPct <= 0 {
			continue
		}
		m, ok, err := e.DB.GetMetricsInWindow(ctx, settings.DataSource, in.Product, in.SKU, row.ProviderCode, since)
		if err != nil {
			return 0, err
		}
		if !ok {
			continue
		}
		pendingTxn := uint(math.Round(float64(m.TotalTransactions) * m.PendingRate / 100))
		failTxn := uint(math.Round(float64(m.TotalTransactions) * m.FailRate / 100))
		if !store.ScopeBreachedFromSnapshot(m.SuccessRate, m.PendingRate, m.FailRate, pendingTxn, failTxn, th) {
			count++
		}
	}
	return count, nil
}

// LoadScopeMetrics loads latest metrics for EvaluateThresholds helper.
func (e *Evaluator) LoadScopeMetrics(ctx context.Context, product, sku, provider string) (MetricInput, bool, error) {
	settings, err := e.DB.GetAgentSettings(ctx)
	if err != nil {
		return MetricInput{}, false, err
	}
	th, err := e.DB.GetProductThreshold(ctx, product)
	if err != nil {
		return MetricInput{}, false, err
	}
	windowMin := th.MetricsWindowMin
	if windowMin <= 0 {
		windowMin = 15
	}
	since := time.Now().Add(-time.Duration(windowMin) * time.Minute)
	m, ok, err := e.DB.GetMetricsInWindow(ctx, settings.DataSource, product, sku, provider, since)
	if err != nil {
		return MetricInput{}, false, err
	}
	if !ok {
		return MetricInput{}, false, nil
	}
	errEvents, err := e.DB.SumErrorEventsAtRecordedAt(ctx, settings.DataSource, product, sku, provider, m.RecordedAt)
	if err != nil {
		return MetricInput{}, false, err
	}
	return MetricInput{
		Product:     product,
		Provider:    provider,
		SKU:         sku,
		SuccessRate: m.SuccessRate,
		PendingRate: m.PendingRate,
		FailRate:    m.FailRate,
		TotalTxn:    m.TotalTransactions,
		ErrorEvents: errEvents,
	}, true, nil
}
