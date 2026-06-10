package agent

import "testing"

func TestOverlayRecoveryState(t *testing.T) {
	t.Parallel()
	cases := []struct {
		base     string
		breached bool
		cycles   int
		want     string
		clear    bool
	}{
		{"INCIDENT", true, 0, "INCIDENT", false},
		{"INCIDENT", true, 1, "RECOVERING", false},
		{"INCIDENT", false, 1, "RECOVERING", false},
		{"INCIDENT", false, 2, "NORMAL", true},
		{"INCIDENT", true, 2, "INCIDENT", false},
		{"NORMAL", false, 3, "NORMAL", true},
	}
	for _, tc := range cases {
		got, clear := overlayRecoveryState(tc.base, tc.breached, tc.cycles)
		if got != tc.want || clear != tc.clear {
			t.Fatalf("cycles=%d breached=%v: got (%s,%v) want (%s,%v)", tc.cycles, tc.breached, got, clear, tc.want, tc.clear)
		}
	}
}
