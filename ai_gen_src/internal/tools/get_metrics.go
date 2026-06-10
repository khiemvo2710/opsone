package tools

import (
	"context"
	"math"
	"time"
)

// GetMetricsInput §6.1.
type GetMetricsInput struct {
	Product  string
	Provider string
	SKU      string
	Window   string
}

// GetMetricsOutput §6.1.
type GetMetricsOutput struct {
	SuccessRate       float64   `json:"success_rate"`
	PendingRate       float64   `json:"pending_rate"`
	FailRate          float64   `json:"fail_rate"`
	TotalTransactions uint      `json:"total_transactions"`
	FailTxnCount      uint      `json:"fail_txn_count"`
	RecordedAt        time.Time `json:"recorded_at"`
	Meta              Meta      `json:"meta"`
}

// GetMetrics returns latest metrics in window (§6.1).
func (r *Registry) GetMetrics(ctx context.Context, in GetMetricsInput) (GetMetricsOutput, error) {
	if in.Product == "" || in.Provider == "" {
		return GetMetricsOutput{}, newErr("invalid_input", "Thiếu product hoặc provider")
	}
	dur, winLabel, err := ParseWindow(in.Window)
	if err != nil {
		return GetMetricsOutput{}, err
	}
	ds, err := r.dataSource(ctx)
	if err != nil {
		return GetMetricsOutput{}, err
	}

	p, err := r.DB.GetProductByCode(ctx, in.Product)
	if err != nil {
		return GetMetricsOutput{}, newErr("product_not_found", "Không tìm thấy sản phẩm")
	}
	if p.RoutingMode == "sku" && in.SKU == "" {
		return GetMetricsOutput{}, newErr("sku_required", "Thẻ/topup data bắt buộc có sku")
	}

	since := time.Now().Add(-dur)
	m, ok, err := r.DB.GetMetricsInWindow(ctx, ds, in.Product, in.SKU, in.Provider, since)
	if err != nil {
		return GetMetricsOutput{}, err
	}
	if !ok {
		return GetMetricsOutput{}, newErr("no_data", "Không có metric trong cửa sổ thời gian")
	}

	failTxn := uint(math.Round(float64(m.TotalTransactions) * m.FailRate / 100))
	return GetMetricsOutput{
		SuccessRate:       m.SuccessRate,
		PendingRate:       m.PendingRate,
		FailRate:          m.FailRate,
		TotalTransactions: m.TotalTransactions,
		FailTxnCount:      failTxn,
		RecordedAt:        m.RecordedAt,
		Meta: Meta{
			DataSource: ds,
			QueriedAt:  time.Now(),
			Window:     winLabel,
		},
	}, nil
}
