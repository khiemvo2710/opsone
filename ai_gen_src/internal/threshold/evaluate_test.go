package threshold_test

import "testing"

func TestSuggestActionDecisionTree(t *testing.T) {
	cases := []struct {
		active, healthy int
		want            string
	}{
		{1, 0, "maintenance"},
		{3, 2, "routing"},
		{3, 0, "maintenance"},
	}
	for _, c := range cases {
		got, _ := suggestAction(c.active, c.healthy)
		if got != c.want {
			t.Errorf("active=%d healthy=%d → %q, want %q", c.active, c.healthy, got, c.want)
		}
	}
}

func suggestAction(activeCount, healthyBackup int) (string, string) {
	if activeCount <= 1 {
		return "maintenance", "single provider"
	}
	if healthyBackup >= 1 {
		return "routing", "has backup"
	}
	return "maintenance", "no healthy backup"
}
