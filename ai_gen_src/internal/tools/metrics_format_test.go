package tools

import (
	"strings"
	"testing"

	"opsone/internal/store"
)

func TestFormatMetricsReply_pendingBreach(t *testing.T) {
	th := store.ProductThreshold{PendingTxnCountMax: 8, PendingRateMaxPct: 5}
	res := BuildMetricsChatResult("MOBIFONE", "50000", "15m", th, []ProviderMetricsLine{
		{Provider: "ESALE", HasData: true, SuccessRate: 95.3, PendingRate: 3.6, FailRate: 1.1, PendingTxn: 29, FailTxn: 9, Breached: true,
			BreachReasons: []string{"Số GD pending 29 vượt ngưỡng 8"}},
		{Provider: "IMEDIA", HasData: true, SuccessRate: 97, PendingRate: 0.5, FailRate: 2, PendingTxn: 2, FailTxn: 8},
	})
	out := FormatMetricsReply(res, true)
	if !strings.Contains(out, "29") || !strings.Contains(out, "ESALE") {
		t.Fatalf("missing pending data: %s", out)
	}
	if !strings.Contains(out, "Vượt ngưỡng") {
		t.Fatalf("missing breach warning: %s", out)
	}
	if strings.Contains(out, "bảo trì") {
		t.Fatal("metrics reply must not mention maintenance")
	}
}
