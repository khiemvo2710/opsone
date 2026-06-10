package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// MaintenanceWindow row from DB.
type MaintenanceWindow struct {
	MaintenanceID string
	ProductCode   string
	ProviderCode  string
	SKUCode       string
	StartsAt      time.Time
	EndsAt        time.Time
	Status        string
	Reason        sql.NullString
}

// MetricScopeKey identifies one mock/agent metric scope (product × sku × provider).
func MetricScopeKey(product, sku, provider string) string {
	return product + "\x00" + sku + "\x00" + provider
}

// ListInWindowMaintenanceMetricScopes returns scopes with active maintenance at now (for mock suppress).
func (db *DB) ListInWindowMaintenanceMetricScopes(ctx context.Context, now time.Time) (map[string]struct{}, error) {
	const query = `
		SELECT product_code, provider_code, sku_code
		FROM maintenance_windows
		WHERE status IN ('scheduled', 'active')
		  AND starts_at <= ? AND ends_at > ?`
	rows, err := db.QueryContext(ctx, query, now, now)
	if err != nil {
		return nil, fmt.Errorf("list maintenance metric suppress: %w", err)
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var product, provider, sku string
		if err := rows.Scan(&product, &provider, &sku); err != nil {
			return nil, err
		}
		out[MetricScopeKey(product, sku, provider)] = struct{}{}
	}
	return out, rows.Err()
}

// ListActiveMaintenance returns scheduled/active windows for scope (§6.8).
func (db *DB) ListActiveMaintenance(ctx context.Context, product, provider, sku string) ([]MaintenanceWindow, error) {
	const query = `
		SELECT maintenance_id, product_code, provider_code, sku_code, starts_at, ends_at, status, reason
		FROM maintenance_windows
		WHERE product_code = ? AND provider_code = ? AND sku_code = ?
		  AND status IN ('scheduled', 'active')
		  AND ends_at > NOW()
		ORDER BY starts_at ASC`
	rows, err := db.QueryContext(ctx, query, product, provider, sku)
	if err != nil {
		return nil, fmt.Errorf("list maintenance: %w", err)
	}
	defer rows.Close()
	var out []MaintenanceWindow
	for rows.Next() {
		var w MaintenanceWindow
		if err := rows.Scan(&w.MaintenanceID, &w.ProductCode, &w.ProviderCode, &w.SKUCode,
			&w.StartsAt, &w.EndsAt, &w.Status, &w.Reason); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// HasOverlappingMaintenance checks scheduled/active overlap (§6.9).
func (db *DB) HasOverlappingMaintenance(ctx context.Context, product, provider, sku string, starts, ends time.Time) (bool, error) {
	const query = `
		SELECT COUNT(*) FROM maintenance_windows
		WHERE product_code = ? AND provider_code = ? AND sku_code = ?
		  AND status IN ('scheduled', 'active')
		  AND starts_at < ? AND ends_at > ?`
	var n int
	err := db.QueryRowContext(ctx, query, product, provider, sku, ends, starts).Scan(&n)
	return n > 0, err
}

// CancelOverlappingMaintenance cancels scheduled/active windows overlapping the interval.
func (db *DB) CancelOverlappingMaintenance(ctx context.Context, product, provider, sku string, starts, ends time.Time, cancelledBy string) (int64, error) {
	const query = `
		UPDATE maintenance_windows
		SET status = 'cancelled', cancelled_by = ?, cancelled_at = NOW()
		WHERE product_code = ? AND provider_code = ? AND sku_code = ?
		  AND status IN ('scheduled', 'active')
		  AND starts_at < ? AND ends_at > ?`
	res, err := db.ExecContext(ctx, query, cancelledBy, product, provider, sku, ends, starts)
	if err != nil {
		return 0, fmt.Errorf("cancel overlapping maintenance: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// CancelActiveMaintenanceForSKU cancels in-window scheduled/active maintenance for a scope.
func (db *DB) CancelActiveMaintenanceForSKU(ctx context.Context, product, sku, cancelledBy string) (int64, error) {
	const query = `
		UPDATE maintenance_windows
		SET status = 'cancelled', cancelled_by = ?, cancelled_at = NOW()
		WHERE product_code = ? AND sku_code = ?
		  AND status IN ('scheduled', 'active')
		  AND starts_at <= NOW() AND ends_at > NOW()`
	res, err := db.ExecContext(ctx, query, cancelledBy, product, sku)
	if err != nil {
		return 0, fmt.Errorf("cancel maintenance scope: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// CancelMaintenanceByIDs cancels specific windows (dashboard maintenance_ids).
func (db *DB) CancelMaintenanceByIDs(ctx context.Context, ids []string, cancelledBy string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, cancelledBy)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		UPDATE maintenance_windows
		SET status = 'cancelled', cancelled_by = ?, cancelled_at = NOW()
		WHERE maintenance_id IN (%s)
		  AND status IN ('scheduled', 'active')`, strings.Join(placeholders, ","))
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("cancel maintenance by id: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// CountActiveMaintenanceForSKU counts scheduled/active windows still in effect for a scope.
func (db *DB) CountActiveMaintenanceForSKU(ctx context.Context, product, sku string) (int64, error) {
	const query = `
		SELECT COUNT(*) FROM maintenance_windows
		WHERE product_code = ? AND sku_code = ?
		  AND status IN ('scheduled', 'active')
		  AND ends_at > NOW()`
	var n int64
	err := db.QueryRowContext(ctx, query, product, sku).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count active maintenance: %w", err)
	}
	return n, nil
}

// UpdateActiveMaintenanceTimesForSKU updates starts/ends for all active windows on a scope.
func (db *DB) UpdateActiveMaintenanceTimesForSKU(ctx context.Context, product, sku string, startsAt, endsAt time.Time) (int64, error) {
	const query = `
		UPDATE maintenance_windows
		SET starts_at = ?, ends_at = ?
		WHERE product_code = ? AND sku_code = ?
		  AND status IN ('scheduled', 'active')
		  AND ends_at > NOW()`
	res, err := db.ExecContext(ctx, query, startsAt, endsAt, product, sku)
	if err != nil {
		return 0, fmt.Errorf("update maintenance scope: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// CancelMaintenanceByID marks one window cancelled (rollback partial admin approve).
func (db *DB) CancelMaintenanceByID(ctx context.Context, maintenanceID, cancelledBy string) error {
	const query = `
		UPDATE maintenance_windows
		SET status = 'cancelled', cancelled_by = ?, cancelled_at = NOW()
		WHERE maintenance_id = ? AND status IN ('scheduled', 'active')`
	_, err := db.ExecContext(ctx, query, cancelledBy, maintenanceID)
	return err
}

// InsertMaintenance creates a maintenance window.
func (db *DB) InsertMaintenance(ctx context.Context, maintenanceID, product, provider, sku string,
	starts, ends time.Time, status, triggerType, reason string, cycleID *uint64) error {
	const query = `
		INSERT INTO maintenance_windows (
			maintenance_id, product_code, provider_code, sku_code,
			starts_at, ends_at, status, trigger_type, reason, cycle_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, query,
		maintenanceID, product, provider, sku,
		starts, ends, status, triggerType, reason, cycleID,
	)
	return err
}

// ChatEscalation provider chat config (§8.9).
type ChatEscalation struct {
	ProviderCode  string
	ChatAppName   string
	ChatGroupName string
	MentionTags   string
}

// GetChatEscalation loads escalation for provider.
func (db *DB) GetChatEscalation(ctx context.Context, providerCode string) (ChatEscalation, bool, error) {
	const query = `
		SELECT provider_code, chat_app_name, chat_group_name, mention_tags
		FROM provider_chat_escalation
		WHERE provider_code = ? AND enabled = 1`
	var c ChatEscalation
	err := db.QueryRowContext(ctx, query, providerCode).Scan(
		&c.ProviderCode, &c.ChatAppName, &c.ChatGroupName, &c.MentionTags,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ChatEscalation{}, false, nil
	}
	if err != nil {
		return ChatEscalation{}, false, err
	}
	return c, true, nil
}

// InsertNotificationLog writes notification_log row.
func (db *DB) InsertNotificationLog(ctx context.Context, dedupeKey, triggerEvent, healthStatus,
	product, provider, sku, subject, actionSummary, status string,
	metricsJSON, chatJSON, recipientsJSON []byte) error {
	const query = `
		INSERT INTO notification_log (
			dedupe_key, trigger_event, health_status, product_code, provider_code, sku_code,
			metrics_snapshot, action_summary, chat_escalation_json, recipients,
			subject, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, query,
		dedupeKey, triggerEvent, healthStatus, product, provider, sku,
		metricsJSON, actionSummary, chatJSON, recipientsJSON,
		subject, status,
	)
	return err
}

// NotificationExists checks dedupe_key.
func (db *DB) NotificationExists(ctx context.Context, dedupeKey string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_log WHERE dedupe_key = ?`, dedupeKey).Scan(&n)
	return n > 0, err
}
