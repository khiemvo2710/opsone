package config

import "testing"

func TestNormalizeLLMModel(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"minimax/MiniMax-M2.5", "minimax/minimax-m2.5"},
		{"minimax/minimax-m2.5", "minimax/minimax-m2.5"},
		{"  openai/gpt-4o-mini  ", "openai/gpt-4o-mini"},
		{"", DefaultLLMModel},
	}
	for _, tc := range tests {
		if got := normalizeLLMModel(tc.in); got != tc.want {
			t.Fatalf("normalizeLLMModel(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
