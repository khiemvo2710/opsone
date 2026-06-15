package store

import (
	"context"
	"database/sql"
	"time"
)

// MetricBaseline holds aggregated baseline for a product×sku×provider at one hour-of-week.
type MetricBaseline struct {
	ProductCode    string
	SKUCode        string
	ProviderCode   string
	HourOfWeek     int
	AvgSuccessRate float64
	AvgPendingRate float64
	AvgFailRate    float64
	StddevSuccess  float64
	SampleCount    int
}

// hourOfWeek returns 0-167: (Mon=0..Sun=6)*24 + hour.
func hourOfWeek(t time.Time) int {
	dow := int(t.Weekday()) // 0=Sun
	if dow == 0 {
		dow = 7
	}
	return (dow-1)*24 + t.Hour()
}

// GetBaseline returns the baseline for the current hour of week.
// Returns nil, nil when no baseline exists yet.
func (db *DB) GetBaseline(ctx context.Context, product, sku, provider string) (*MetricBaseline, error) {
	how := hourOfWeek(time.Now())
	const q = `SELECT avg_success_rate, avg_pending_rate, avg_fail_rate, stddev_success, sample_count
               FROM metric_baseline_hourly
               WHERE product_code=? AND sku_code=? AND provider_code=? AND hour_of_week=?`
	var b MetricBaseline
	b.ProductCode = product
	b.SKUCode = sku
	b.ProviderCode = provider
	b.HourOfWeek = how
	err := db.QueryRowContext(ctx, q, product, sku, provider, how).Scan(
		&b.AvgSuccessRate, &b.AvgPendingRate, &b.AvgFailRate, &b.StddevSuccess, &b.SampleCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// AggregateBaselines recomputes hour-of-week baselines from the last lookbackDays of history.
func (db *DB) AggregateBaselines(ctx context.Context, lookbackDays int) error {
	if lookbackDays <= 0 {
		lookbackDays = 14
	}
	// DAYOFWEEK: 1=Sun,2=Mon,...,7=Sat → map to 0=Mon..6=Sun
	const q = `
	INSERT INTO metric_baseline_hourly
	  (product_code, sku_code, provider_code, hour_of_week,
	   avg_success_rate, avg_pending_rate, avg_fail_rate, stddev_success, sample_count)
	SELECT
	  product_code, sku_code, provider_code,
	  (CASE DAYOFWEEK(recorded_at)
	     WHEN 2 THEN 0 WHEN 3 THEN 1 WHEN 4 THEN 2
	     WHEN 5 THEN 3 WHEN 6 THEN 4 WHEN 7 THEN 5 ELSE 6
	   END) * 24 + HOUR(recorded_at) AS hour_of_week,
	  ROUND(AVG(success_rate), 2),
	  ROUND(AVG(pending_rate), 2),
	  ROUND(AVG(fail_rate),    2),
	  ROUND(IFNULL(STDDEV(success_rate), 0), 2),
	  COUNT(*)
	FROM agent_analysis_history
	WHERE recorded_at >= DATE_SUB(NOW(), INTERVAL ? DAY)
	GROUP BY product_code, sku_code, provider_code, hour_of_week
	ON DUPLICATE KEY UPDATE
	  avg_success_rate = VALUES(avg_success_rate),
	  avg_pending_rate = VALUES(avg_pending_rate),
	  avg_fail_rate    = VALUES(avg_fail_rate),
	  stddev_success   = VALUES(stddev_success),
	  sample_count     = VALUES(sample_count)
	`
	_, err := db.ExecContext(ctx, q, lookbackDays)
	return err
}

// BaselineDeviation returns how many stddevs current success_rate is below the baseline.
// Returns (0, false) if baseline is insufficient (sample_count < 5).
func BaselineDeviation(currentSuccess float64, b *MetricBaseline) (deviations float64, hasBaseline bool) {
	if b == nil || b.SampleCount < 5 {
		return 0, false
	}
	stddev := b.StddevSuccess
	if stddev < 1.0 {
		stddev = 1.0 // floor to avoid division by near-zero
	}
	return (b.AvgSuccessRate - currentSuccess) / stddev, true
}
