package tools

import (
	"fmt"
	"sort"
	"strings"

	"opsone/internal/store"
)

// ProviderMetricsLine is one provider row for chat metrics reply.
type ProviderMetricsLine struct {
	Provider          string
	SuccessRate       float64
	PendingRate       float64
	FailRate          float64
	PendingTxn        uint
	FailTxn           uint
	HasData           bool
	Breached          bool
	BreachReasons     []string
}

// MetricsChatResult aggregates provider metrics for one scope.
type MetricsChatResult struct {
	Product string
	SKU     string
	Window  string
	Lines   []ProviderMetricsLine
}

// FormatMetricsReply builds a Vietnamese chat answer for pending/metrics queries.
func FormatMetricsReply(res MetricsChatResult, pendingFocused bool) string {
	if len(res.Lines) == 0 {
		return "Không có dữ liệu metric trong cửa sổ " + res.Window + "."
	}
	title := fmt.Sprintf("**%s**", res.Product)
	if res.SKU != "" {
		title = fmt.Sprintf("**%s %s**", res.Product, res.SKU)
	}
	var b strings.Builder
	if pendingFocused {
		b.WriteString(title)
		b.WriteString(" — GD pending (cửa sổ ")
		b.WriteString(res.Window)
		b.WriteString("):\n")
	} else {
		b.WriteString(title)
		b.WriteString(" — metric (cửa sổ ")
		b.WriteString(res.Window)
		b.WriteString("):\n")
	}

	hasData := false
	breached := make([]string, 0)
	for _, line := range res.Lines {
		if !line.HasData {
			b.WriteString(fmt.Sprintf("- **%s**: không có dữ liệu\n", line.Provider))
			continue
		}
		hasData = true
		b.WriteString(fmt.Sprintf(
			"- **%s**: **%d** GD pending (%.1f%%), success %.1f%%, fail %.1f%% (fail %d GD)\n",
			line.Provider, line.PendingTxn, line.PendingRate, line.SuccessRate, line.FailRate, line.FailTxn,
		))
		if line.Breached {
			breached = append(breached, line.Provider)
		}
	}
	if !hasData {
		return title + ": không có metric trong cửa sổ " + res.Window + "."
	}

	if len(breached) > 0 {
		sort.Strings(breached)
		b.WriteString("\n⚠️ Vượt ngưỡng: **")
		b.WriteString(strings.Join(breached, "**, **"))
		b.WriteString("**.")
		for _, line := range res.Lines {
			if !line.Breached || len(line.BreachReasons) == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("\n- %s: %s", line.Provider, strings.Join(line.BreachReasons, "; ")))
		}
	} else if pendingFocused {
		b.WriteString("\nHiện không có provider nào vượt ngưỡng pending.")
	}
	return strings.TrimSpace(b.String())
}

// BuildMetricsChatResult assembles provider lines with breach detection.
func BuildMetricsChatResult(product, sku, window string, th store.ProductThreshold, rows []ProviderMetricsLine) MetricsChatResult {
	out := MetricsChatResult{Product: product, SKU: sku, Window: window, Lines: make([]ProviderMetricsLine, 0, len(rows))}
	for _, row := range rows {
		if !row.HasData {
			out.Lines = append(out.Lines, row)
			continue
		}
		reasons := store.SnapshotBreachReasons(row.SuccessRate, row.PendingRate, row.FailRate, row.PendingTxn, row.FailTxn, th)
		row.BreachReasons = reasons
		row.Breached = len(reasons) > 0
		out.Lines = append(out.Lines, row)
	}
	sort.Slice(out.Lines, func(i, j int) bool { return out.Lines[i].Provider < out.Lines[j].Provider })
	return out
}
