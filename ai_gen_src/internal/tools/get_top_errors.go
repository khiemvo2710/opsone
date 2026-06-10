package tools

import (
	"context"
	"time"
)

// TopErrorItem §6.2.
type TopErrorItem struct {
	Error string `json:"error"`
	Count uint   `json:"count"`
}

// GetTopErrorsInput §6.2.
type GetTopErrorsInput struct {
	Product  string
	Provider string
	SKU      string
	Window   string
	Limit    int
}

// GetTopErrorsOutput §6.2.
type GetTopErrorsOutput struct {
	Errors []TopErrorItem `json:"errors"`
	Meta   Meta           `json:"meta"`
}

// GetTopErrors aggregates error codes in window (§6.2).
func (r *Registry) GetTopErrors(ctx context.Context, in GetTopErrorsInput) (GetTopErrorsOutput, error) {
	if in.Product == "" || in.Provider == "" {
		return GetTopErrorsOutput{}, newErr("invalid_input", "Thiếu product hoặc provider")
	}
	dur, winLabel, err := ParseWindow(in.Window)
	if err != nil {
		return GetTopErrorsOutput{}, err
	}
	ds, err := r.dataSource(ctx)
	if err != nil {
		return GetTopErrorsOutput{}, err
	}

	since := time.Now().Add(-dur)
	rows, err := r.DB.GetTopErrorsInWindow(ctx, ds, in.Product, in.SKU, in.Provider, since, in.Limit)
	if err != nil {
		return GetTopErrorsOutput{}, err
	}
	out := make([]TopErrorItem, 0, len(rows))
	for _, e := range rows {
		out = append(out, TopErrorItem{Error: e.ErrorCode, Count: e.ErrorCount})
	}
	return GetTopErrorsOutput{
		Errors: out,
		Meta: Meta{DataSource: ds, QueriedAt: time.Now(), Window: winLabel},
	}, nil
}
