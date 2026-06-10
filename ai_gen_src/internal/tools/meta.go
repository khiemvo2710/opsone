package tools

import (
	"fmt"
	"time"
)

// Meta is attached to every tool response (§14.4).
type Meta struct {
	DataSource string    `json:"data_source"`
	QueriedAt  time.Time `json:"queried_at"`
	Window     string    `json:"window,omitempty"`
}

// ToolError is the error envelope (§14.1).
type ToolError struct {
	Code     string `json:"code"`
	MessageVI string `json:"message_vi"`
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.MessageVI)
}

func newErr(code, msg string) error {
	return &ToolError{Code: code, MessageVI: msg}
}

// ParseWindow converts "15m", "1h" to duration; default 15m.
func ParseWindow(w string) (time.Duration, string, error) {
	if w == "" {
		return 15 * time.Minute, "15m", nil
	}
	if len(w) < 2 {
		return 0, "", newErr("invalid_window", "Cửa sổ thời gian không hợp lệ")
	}
	unit := w[len(w)-1]
	numStr := w[:len(w)-1]
	var n int
	if _, err := fmt.Sscanf(numStr, "%d", &n); err != nil || n <= 0 {
		return 0, "", newErr("invalid_window", "Cửa sổ thời gian không hợp lệ")
	}
	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, w, nil
	case 'h':
		return time.Duration(n) * time.Hour, w, nil
	default:
		return 0, "", newErr("invalid_window", "Cửa sổ thời gian không hợp lệ")
	}
}
