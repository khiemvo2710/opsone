package tools

import (
	"strings"
	"testing"
	"time"
)

func TestFormatMaintenanceReply_activeBySKU(t *testing.T) {
	now := time.Now()
	out := GetMaintenanceOutput{
		Active: []MaintenanceItem{
			{SKUCode: "10000", ProviderCode: "ESALE", RemainingMinutes: 45, StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour)},
			{SKUCode: "10000", ProviderCode: "IMEDIA", RemainingMinutes: 45, StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour)},
			{SKUCode: "20000", ProviderCode: "ESALE", RemainingMinutes: 30, StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour)},
		},
	}
	reply := FormatMaintenanceReply("GARENA", "", out)
	if !strings.Contains(reply, "10000") || !strings.Contains(reply, "20000") {
		t.Fatalf("reply missing skus: %q", reply)
	}
	if strings.Contains(reply, "không có bảo trì") {
		t.Fatalf("unexpected empty reply: %q", reply)
	}
}
