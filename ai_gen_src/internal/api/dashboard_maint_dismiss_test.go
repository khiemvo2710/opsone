package api

import "testing"

func TestMaintenanceDismissedThisCycle(t *testing.T) {
	if !maintenanceDismissedThisCycle(15, 15, true) {
		t.Fatal("expected suppress when dismissed in same completed cycle")
	}
	if maintenanceDismissedThisCycle(15, 16, true) {
		t.Fatal("expected show after next cycle completes")
	}
	if maintenanceDismissedThisCycle(0, 15, true) {
		t.Fatal("legacy dismiss without cycle_id should not suppress")
	}
	if maintenanceDismissedThisCycle(15, 15, false) {
		t.Fatal("no dismiss row should not suppress")
	}
}
