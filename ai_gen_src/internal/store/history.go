package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ScopeHistoryPoint is one prior metric sample for trend / consecutive breach.
type ScopeHistoryPoint struct {
	CycleID     uint64
	RecordedAt  time.Time
	SuccessRate float64
	PendingRate float64
	FailRate    float64
}

func scopeHistoryKey(product, sku, provider string) string {
	return product + "\x00" + sku + "\x00" + provider
}

// LoadAgentHistoryThroughCycle loads recent history rows per scope through latestCycle (one query for dashboard).
func (db *DB) LoadAgentHistoryThroughCycle(ctx context.Context, throughCycleID uint64, maxPerScope int) (map[string][]ScopeHistoryPoint, error) {
	if maxPerScope <= 0 {
		maxPerScope = 5
	}
	const query = `
		SELECT product_code, sku_code, provider_code, cycle_id, recorded_at, success_rate, pending_rate, fail_rate
		FROM agent_analysis_history
		WHERE cycle_id <= ?
		ORDER BY product_code, sku_code, provider_code, cycle_id DESC`
	rows, err := db.QueryContext(ctx, query, throughCycleID)
	if err != nil {
		return nil, fmt.Errorf("load agent history: %w", err)
	}
	defer rows.Close()
	out := make(map[string][]ScopeHistoryPoint)
	counts := make(map[string]int)
	for rows.Next() {
		var product, sku, provider string
		var p ScopeHistoryPoint
		if err := rows.Scan(&product, &sku, &provider, &p.CycleID, &p.RecordedAt, &p.SuccessRate, &p.PendingRate, &p.FailRate); err != nil {
			return nil, err
		}
		key := scopeHistoryKey(product, sku, provider)
		if counts[key] >= maxPerScope {
			continue
		}
		out[key] = append(out[key], p)
		counts[key]++
	}
	return out, rows.Err()
}

// ScopeConsecutiveBreachFromHistory counts consecutive breaches using preloaded history (no per-scope DB).
func ScopeConsecutiveBreachFromHistory(
	product, sku, provider string,
	th ProductThreshold,
	liveBreached bool,
	latestCycle uint64,
	history map[string][]ScopeHistoryPoint,
) (consecutive int, shouldAct bool) {
	if !liveBreached {
		return 0, false
	}
	required := th.ConsecutiveCyclesRequired
	if required <= 0 {
		required = 2
	}
	if latestCycle == 0 {
		return 1, 1 >= required
	}
	prior := 0
	for _, p := range history[scopeHistoryKey(product, sku, provider)] {
		if p.CycleID == latestCycle {
			continue
		}
		if p.CycleID < latestCycle {
			if ScopeBreachedFromRates(p.SuccessRate, p.PendingRate, p.FailRate, th) {
				prior++
			} else {
				break
			}
		}
	}
	consecutive = prior + 1
	return consecutive, consecutive >= required
}

// GetRecentScopeHistory returns last N history rows before current cycle (§3 step 3).
func (db *DB) GetRecentScopeHistory(ctx context.Context, product, sku, provider string, beforeCycleID uint64, limit int) ([]ScopeHistoryPoint, error) {
	if limit <= 0 {
		limit = 5
	}
	const query = `
		SELECT cycle_id, recorded_at, success_rate, pending_rate, fail_rate
		FROM agent_analysis_history
		WHERE product_code = ? AND sku_code = ? AND provider_code = ?
		  AND cycle_id < ?
		ORDER BY recorded_at DESC
		LIMIT ?`
	rows, err := db.QueryContext(ctx, query, product, sku, provider, beforeCycleID, limit)
	if err != nil {
		return nil, fmt.Errorf("recent scope history: %w", err)
	}
	defer rows.Close()
	var out []ScopeHistoryPoint
	for rows.Next() {
		var p ScopeHistoryPoint
		if err := rows.Scan(&p.CycleID, &p.RecordedAt, &p.SuccessRate, &p.PendingRate, &p.FailRate); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ScopeBreachedFromRates checks rate-only thresholds (history lacks txn counts).
func ScopeBreachedFromRates(success, pending, fail float64, th ProductThreshold) bool {
	if success < th.SuccessRateMinPct {
		return true
	}
	if pending > th.PendingRateMaxPct {
		return true
	}
	if fail > th.FailRateMaxPct {
		return true
	}
	return false
}

// SnapshotBreachReasons lists human-readable breach reasons (khớp ScopeBreachedFromSnapshot / UI §9.0).
func SnapshotBreachReasons(success, pending, fail float64, pendingTxn, failTxn uint, th ProductThreshold) []string {
	var reasons []string
	if success <= th.SuccessRateMinPct {
		reasons = append(reasons, fmt.Sprintf("Tỷ lệ thành công %.1f%% dưới ngưỡng %.1f%%", success, th.SuccessRateMinPct))
	}
	if pending >= th.PendingRateMaxPct {
		reasons = append(reasons, fmt.Sprintf("Tỷ lệ pending %.1f%% vượt ngưỡng %.1f%%", pending, th.PendingRateMaxPct))
	}
	if fail >= th.FailRateMaxPct {
		reasons = append(reasons, fmt.Sprintf("Tỷ lệ lỗi %.1f%% vượt ngưỡng %.1f%%", fail, th.FailRateMaxPct))
	}
	if th.PendingTxnCountMax > 0 && pendingTxn >= th.PendingTxnCountMax {
		reasons = append(reasons, fmt.Sprintf("Số GD pending %d vượt ngưỡng %d", pendingTxn, th.PendingTxnCountMax))
	}
	if th.FailTxnCountMax > 0 && failTxn >= th.FailTxnCountMax {
		reasons = append(reasons, fmt.Sprintf("Số GD fail %d vượt ngưỡng %d", failTxn, th.FailTxnCountMax))
	}
	return reasons
}

// ScopeBreachedFromSnapshot checks all 5 dashboard metrics (%, GD pending/fail) — khớp UI §9.0.
func ScopeBreachedFromSnapshot(success, pending, fail float64, pendingTxn, failTxn uint, th ProductThreshold) bool {
	if success <= th.SuccessRateMinPct {
		return true
	}
	if pending >= th.PendingRateMaxPct {
		return true
	}
	if fail >= th.FailRateMaxPct {
		return true
	}
	if th.PendingTxnCountMax > 0 && pendingTxn >= th.PendingTxnCountMax {
		return true
	}
	if th.FailTxnCountMax > 0 && failTxn >= th.FailTxnCountMax {
		return true
	}
	return false
}

// ScopeBreached checks if rates violate product thresholds (§7.4).
func ScopeBreached(success, pending, fail float64, totalTxn uint, errEvents uint, th ProductThreshold) bool {
	if ScopeBreachedFromRates(success, pending, fail, th) {
		return true
	}
	failTxn := uint(float64(totalTxn) * fail / 100)
	if th.FailTxnCountMax > 0 && failTxn >= th.FailTxnCountMax {
		return true
	}
	pendingTxn := uint(float64(totalTxn) * pending / 100)
	if th.PendingTxnCountMax > 0 && pendingTxn >= th.PendingTxnCountMax {
		return true
	}
	if errEvents > th.ErrorEventCountMax {
		return true
	}
	return false
}

// HistoryPointBreachedAtCycle reports whether history for one cycle violates thresholds (rates only).
func (db *DB) HistoryPointBreachedAtCycle(ctx context.Context, product, sku, provider string, cycleID uint64, th ProductThreshold) (bool, error) {
	const query = `
		SELECT success_rate, pending_rate, fail_rate
		FROM agent_analysis_history
		WHERE cycle_id = ? AND product_code = ? AND sku_code = ? AND provider_code = ?
		LIMIT 1`
	var success, pending, fail float64
	err := db.QueryRowContext(ctx, query, cycleID, product, sku, provider).Scan(&success, &pending, &fail)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ScopeBreachedFromRates(success, pending, fail, th), nil
}

// ScopeConsecutiveBreachLive counts consecutive breached agent cycles including live snapshot (§1.2 / §2.3.2).
// When live is not breached, returns (0, false) — scope returns to green on dashboard.
func (db *DB) ScopeConsecutiveBreachLive(ctx context.Context, product, sku, provider string, th ProductThreshold, liveBreached bool) (consecutive int, shouldAct bool) {
	if !liveBreached {
		return 0, false
	}
	required := th.ConsecutiveCyclesRequired
	if required <= 0 {
		required = 2
	}
	latestCycle, err := db.LatestCompletedCycleID(ctx)
	if err != nil || latestCycle == 0 {
		return 1, 1 >= required
	}
	prior, err := db.CountConsecutiveBreaches(ctx, product, sku, provider, latestCycle, th)
	if err != nil {
		return 1, 1 >= required
	}
	consecutive = prior + 1
	return consecutive, consecutive >= required
}

// CountConsecutiveBreaches counts prior consecutive breached cycles for scope (§1.2).
func (db *DB) CountConsecutiveBreaches(ctx context.Context, product, sku, provider string, beforeCycleID uint64, th ProductThreshold) (int, error) {
	points, err := db.GetRecentScopeHistory(ctx, product, sku, provider, beforeCycleID, th.ConsecutiveCyclesRequired+2)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, p := range points {
		if ScopeBreachedFromRates(p.SuccessRate, p.PendingRate, p.FailRate, th) {
			count++
		} else {
			break
		}
	}
	return count, nil
}

// HealthStatusRow for health_status_product insert.
type HealthStatusRow struct {
	ProductCode   string
	HealthStatus  string
	HealthSummary string
}

// InsertHealthStatusProducts batch insert per-product health (§3.1).
func (db *DB) InsertHealthStatusProducts(ctx context.Context, cycleID uint64, rows []HealthStatusRow) error {
	const query = `
		INSERT INTO health_status_product (cycle_id, product_code, health_status, health_summary)
		VALUES (?, ?, ?, ?)`
	for _, r := range rows {
		_, err := db.ExecContext(ctx, query, cycleID, r.ProductCode, r.HealthStatus, r.HealthSummary)
		if err != nil {
			return fmt.Errorf("insert health_status_product: %w", err)
		}
	}
	return nil
}

// AgentStateRow for agent_state_history insert.
type AgentStateRow struct {
	ProductCode      string
	SKUCode          string
	State            string
	PrevState        *string
	TransitionReason string
}

// InsertAgentStateHistory writes state transitions (§3.1).
func (db *DB) InsertAgentStateHistory(ctx context.Context, cycleID uint64, rows []AgentStateRow) error {
	const query = `
		INSERT INTO agent_state_history (cycle_id, product_code, sku_code, state, prev_state, transition_reason)
		VALUES (?, ?, ?, ?, ?, ?)`
	for _, r := range rows {
		_, err := db.ExecContext(ctx, query, cycleID, r.ProductCode, r.SKUCode, r.State, r.PrevState, r.TransitionReason)
		if err != nil {
			return fmt.Errorf("insert agent_state_history: %w", err)
		}
	}
	return nil
}

// ScopeStateRow is product×sku state from one analysis cycle.
type ScopeStateRow struct {
	ProductCode string
	SKUCode     string
	State       string
}

// ListAgentStateByCycle returns scope states recorded for a cycle.
func (db *DB) ListAgentStateByCycle(ctx context.Context, cycleID uint64) ([]ScopeStateRow, error) {
	const query = `
		SELECT product_code, sku_code, state
		FROM agent_state_history
		WHERE cycle_id = ?
		ORDER BY product_code, sku_code`
	rows, err := db.QueryContext(ctx, query, cycleID)
	if err != nil {
		return nil, fmt.Errorf("list agent state by cycle: %w", err)
	}
	defer rows.Close()
	var out []ScopeStateRow
	for rows.Next() {
		var r ScopeStateRow
		if err := rows.Scan(&r.ProductCode, &r.SKUCode, &r.State); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// StateToHealthStatus maps agent scope state to dashboard health color.
func StateToHealthStatus(state string) string {
	switch state {
	case "INCIDENT":
		return "red"
	case "WARNING", "RECOVERING", "MAINTENANCE_ACTIVE":
		return "yellow"
	default:
		return "green"
	}
}

// GetLatestScopeState returns most recent state for product×sku scope.
func (db *DB) GetLatestScopeState(ctx context.Context, product, sku string) (string, bool, error) {
	const query = `
		SELECT state FROM agent_state_history
		WHERE product_code = ? AND sku_code = ?
		ORDER BY recorded_at DESC
		LIMIT 1`
	var state string
	err := db.QueryRowContext(ctx, query, product, sku).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return state, true, nil
}
