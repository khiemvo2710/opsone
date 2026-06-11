package llm

import (
	"encoding/json"
	"testing"
)

func TestToolDefJSON(t *testing.T) {
	def := ToolDef{
		Type: "function",
		Function: FunctionSpec{
			Name:        "get_metrics",
			Description: "test",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"product": map[string]any{"type": "string"},
				},
			},
		},
	}
	raw, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 10 {
		t.Fatalf("unexpected json: %s", raw)
	}
}

func TestClientEnabled(t *testing.T) {
	c := NewClient(Config{BaseURL: "https://example.com/v1", APIKey: "k", Model: "m"})
	if !c.Enabled() {
		t.Fatal("expected enabled")
	}
	c2 := NewClient(Config{})
	if c2.Enabled() {
		t.Fatal("expected disabled")
	}
}
