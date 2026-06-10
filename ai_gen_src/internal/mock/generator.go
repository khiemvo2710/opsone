package mock

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"opsone/internal/catalog"
	"opsone/internal/store"
)

// Generator produces mock_metrics / mock_error_stats each tick (§4.5).
type Generator struct {
	DB    *store.DB
	Rng   *rand.Rand
	state map[string]float64 // esale_degrading success baseline per scope
}

// NewGenerator creates a generator with optional seed (0 = time-based).
func NewGenerator(db *store.DB, seed int64) *Generator {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Generator{
		DB:    db,
		Rng:   rand.New(rand.NewSource(seed)),
		state: make(map[string]float64),
	}
}

// RunOnce executes one mock generation cycle.
func (g *Generator) RunOnce(ctx context.Context) (int, error) {
	settings, err := g.DB.GetAgentSettings(ctx)
	if err != nil {
		return 0, err
	}
	if !settings.MockEnabled || settings.DataSource != "mock" {
		log.Println("mock: skipped (disabled or data_source != mock)")
		return 0, nil
	}

	now := time.Now()
	runID, err := g.DB.StartMockGeneratorRun(ctx, now, settings.MockScenario)
	if err != nil {
		return 0, err
	}

	scopes, err := catalog.ListMetricScopes(ctx, g.DB)
	if err != nil {
		_ = g.DB.FinishMockGeneratorRun(ctx, runID, time.Now(), 0, err.Error())
		return 0, err
	}

	suppressed, err := g.DB.ListInWindowMaintenanceMetricScopes(ctx, now)
	if err != nil {
		_ = g.DB.FinishMockGeneratorRun(ctx, runID, time.Now(), 0, err.Error())
		return 0, err
	}

	var metrics []store.MockMetricRow
	var errors []store.MockErrorRow
	var skippedMaint int

	for _, sc := range scopes {
		if _, inMaint := suppressed[store.MetricScopeKey(sc.ProductCode, sc.SKUCode, sc.ProviderCode)]; inMaint {
			skippedMaint++
			continue
		}
		rates := computeRates(settings.MockScenario, sc.ProductCode, sc.SKUCode, sc.ProviderCode, g.Rng, g.state)
		totalTxn := uint(800 + g.Rng.Intn(400))
		revenue := uint64(50_000_000 + g.Rng.Int63n(100_000_000))

		metrics = append(metrics, store.MockMetricRow{
			RecordedAt:        now,
			ProductCode:       sc.ProductCode,
			SKUCode:           sc.SKUCode,
			ProviderCode:      sc.ProviderCode,
			SuccessRate:       rates.Success,
			PendingRate:       rates.Pending,
			FailRate:          rates.Fail,
			TotalTransactions: totalTxn,
			RevenueLastHour:   revenue,
			GeneratorRunID:    runID,
		})

		c3004, c22 := errorCounts(rates.Fail, totalTxn, g.Rng)
		if c3004 > 0 {
			errors = append(errors, store.MockErrorRow{
				RecordedAt: now, ProductCode: sc.ProductCode, ProviderCode: sc.ProviderCode,
				SKUCode: sc.SKUCode, ErrorCode: "-3004", ErrorCount: c3004, GeneratorRunID: runID,
			})
		}
		if c22 > 0 {
			errors = append(errors, store.MockErrorRow{
				RecordedAt: now, ProductCode: sc.ProductCode, ProviderCode: sc.ProviderCode,
				SKUCode: sc.SKUCode, ErrorCode: "-22", ErrorCount: c22, GeneratorRunID: runID,
			})
		}
	}

	if err := g.DB.InsertMockMetrics(ctx, metrics); err != nil {
		_ = g.DB.FinishMockGeneratorRun(ctx, runID, time.Now(), 0, err.Error())
		return 0, err
	}
	if err := g.DB.InsertMockErrors(ctx, errors); err != nil {
		_ = g.DB.FinishMockGeneratorRun(ctx, runID, time.Now(), 0, err.Error())
		return 0, err
	}

	retention := settings.MockRetentionHours
	if retention <= 0 {
		retention = 24
	}
	cutoff := now.Add(-time.Duration(retention) * time.Hour)
	if err := g.DB.PurgeMockMetricsOlderThan(ctx, cutoff); err != nil {
		log.Printf("mock: retention purge warning: %v", err)
	}

	if err := g.DB.FinishMockGeneratorRun(ctx, runID, time.Now(), len(metrics), ""); err != nil {
		return len(metrics), err
	}

	if skippedMaint > 0 {
		log.Printf("mock: generated %d metric rows (%d scopes skipped — maintenance) scenario=%s",
			len(metrics), skippedMaint, settings.MockScenario)
	} else {
		log.Printf("mock: generated %d metric rows scenario=%s", len(metrics), settings.MockScenario)
	}
	return len(metrics), nil
}

// RunBlocking starts ticker until ctx cancelled.
func (g *Generator) RunBlocking(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if _, err := g.RunOnce(ctx); err != nil {
		return fmt.Errorf("mock initial run: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := g.RunOnce(ctx); err != nil {
				log.Printf("mock: run error: %v", err)
			}
		}
	}
}
