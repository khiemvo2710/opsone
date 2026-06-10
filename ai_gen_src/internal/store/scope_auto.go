package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ScopeAutoConfig per product+sku routing scope (§9.5).
type ScopeAutoConfig struct {
	ProductCode  string
	SKUCode      string
	AutoAction   string
	WindowStart  string // datetime-local or ISO
	WindowEnd    string
}

func scopeAutoKey(product, sku string) string {
	return product + "\x00" + sku
}

// NormalizeScopeAutoAction maps legacy values to recommend_only | auto | time_window.
func NormalizeScopeAutoAction(v string) string {
	switch v {
	case "auto":
		return "auto"
	case "time_window":
		return "time_window"
	default:
		return "recommend_only"
	}
}

func parseScopeDateTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"15:04:05",
		"15:04",
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err != nil {
			continue
		}
		if layout == "15:04:05" || layout == "15:04" {
			now := time.Now()
			t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		}
		return t, true
	}
	return time.Time{}, false
}

// InAutoTimeWindow reports whether now falls in [start, end).
func InAutoTimeWindow(now time.Time, start, end string) bool {
	st, ok1 := parseScopeDateTime(start)
	en, ok2 := parseScopeDateTime(end)
	if !ok1 || !ok2 {
		return false
	}
	return !now.Before(st) && now.Before(en)
}

// ShouldAutoApplyScope returns true when agent may apply routing without admin approve.
func ShouldAutoApplyScope(cfg ScopeAutoConfig, now time.Time) bool {
	switch NormalizeScopeAutoAction(cfg.AutoAction) {
	case "auto":
		return true
	case "time_window":
		return InAutoTimeWindow(now, cfg.WindowStart, cfg.WindowEnd)
	default:
		return false
	}
}

func formatDateTimePtr(t sql.NullString) string {
	if !t.Valid || t.String == "" {
		return ""
	}
	parsed, ok := parseScopeDateTime(t.String)
	if !ok {
		return ""
	}
	return parsed.Format("2006-01-02T15:04")
}

func nullDateTime(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	t, ok := parseScopeDateTime(s)
	if !ok {
		return nil
	}
	return t.Format("2006-01-02 15:04:05")
}

// ListScopeAutoConfig loads auto settings for all scopes.
func (db *DB) ListScopeAutoConfig(ctx context.Context) (map[string]ScopeAutoConfig, error) {
	const query = `
		SELECT product_code, sku_code, auto_action, window_start, window_end
		FROM routing_scope_state`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]ScopeAutoConfig)
	for rows.Next() {
		var c ScopeAutoConfig
		var ws, we sql.NullString
		if err := rows.Scan(&c.ProductCode, &c.SKUCode, &c.AutoAction, &ws, &we); err != nil {
			return nil, err
		}
		c.AutoAction = NormalizeScopeAutoAction(c.AutoAction)
		c.WindowStart = formatDateTimePtr(ws)
		c.WindowEnd = formatDateTimePtr(we)
		out[scopeAutoKey(c.ProductCode, c.SKUCode)] = c
	}
	return out, rows.Err()
}

// GetScopeAutoConfig loads one scope; missing row returns defaults.
func (db *DB) GetScopeAutoConfig(ctx context.Context, product, sku string) (ScopeAutoConfig, error) {
	const query = `
		SELECT auto_action, window_start, window_end
		FROM routing_scope_state
		WHERE product_code = ? AND sku_code = ?`
	var c ScopeAutoConfig
	c.ProductCode = product
	c.SKUCode = sku
	var ws, we sql.NullString
	err := db.QueryRowContext(ctx, query, product, sku).Scan(&c.AutoAction, &ws, &we)
	if err == sql.ErrNoRows {
		return ScopeAutoConfig{
			ProductCode: product,
			SKUCode:     sku,
			AutoAction:  "recommend_only",
		}, nil
	}
	if err != nil {
		return ScopeAutoConfig{}, err
	}
	c.AutoAction = NormalizeScopeAutoAction(c.AutoAction)
	c.WindowStart = formatDateTimePtr(ws)
	c.WindowEnd = formatDateTimePtr(we)
	return c, nil
}

// UpsertScopeAutoConfig saves per-scope auto routing mode.
func (db *DB) UpsertScopeAutoConfig(ctx context.Context, c ScopeAutoConfig) error {
	c.AutoAction = NormalizeScopeAutoAction(c.AutoAction)
	if c.AutoAction == "time_window" {
		st, ok1 := parseScopeDateTime(c.WindowStart)
		en, ok2 := parseScopeDateTime(c.WindowEnd)
		if !ok1 || !ok2 || !en.After(st) {
			return fmt.Errorf("time_window cần khung thời gian từ–đến hợp lệ")
		}
	} else {
		c.WindowStart = ""
		c.WindowEnd = ""
	}
	const query = `
		INSERT INTO routing_scope_state (product_code, sku_code, auto_action, window_start, window_end, pending_restore)
		VALUES (?, ?, ?, ?, ?, 0)
		ON DUPLICATE KEY UPDATE
			auto_action = VALUES(auto_action),
			window_start = VALUES(window_start),
			window_end = VALUES(window_end)`
	_, err := db.ExecContext(ctx, query,
		c.ProductCode, c.SKUCode, c.AutoAction,
		nullDateTime(c.WindowStart), nullDateTime(c.WindowEnd),
	)
	return err
}
