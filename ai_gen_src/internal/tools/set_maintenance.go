package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
	"opsone/internal/notify"
)

func normalizeMaintenanceTrigger(trigger string) string {
	switch trigger {
	case "opsone_recommend", "opsone_auto", "admin_manual":
		return trigger
	case "admin_approve":
		return "admin_manual"
	default:
		if trigger == "" {
			return "opsone_auto"
		}
		return "opsone_auto"
	}
}

// SetMaintenanceInput §6.9.
type SetMaintenanceInput struct {
	Product     string
	Provider    string
	SKU         string
	StartsAt    time.Time
	EndsAt      time.Time
	TriggerType string
	Reason      string
	Status      string
	CycleID     *uint64
	Seq         int    // disambiguates multi-provider SKU-wide approve in the same second
	SkipNotify  bool   // If true, do not send notification (caller will batch)
	Actor       string // Who performed the action
}

// SetMaintenanceOutput §6.9.
type SetMaintenanceOutput struct {
	MaintenanceID string `json:"maintenance_id"`
	Status        string `json:"status"`
	Product       string `json:"product"`
	Provider      string `json:"provider"`
	StartsAt      time.Time `json:"starts_at"`
	EndsAt        time.Time `json:"ends_at"`
	Meta          Meta   `json:"meta"`
}

// SetMaintenance schedules or activates a maintenance window (§6.9).
func (r *Registry) SetMaintenance(ctx context.Context, in SetMaintenanceInput) (SetMaintenanceOutput, error) {
	if in.Product == "" || in.Provider == "" {
		return SetMaintenanceOutput{}, newErr("invalid_input", "Thiếu product hoặc provider")
	}
	if !in.EndsAt.After(in.StartsAt) {
		return SetMaintenanceOutput{}, newErr("invalid_window", "ends_at phải sau starts_at")
	}

	trigger := normalizeMaintenanceTrigger(in.TriggerType)
	if trigger == "admin_manual" {
		_, _ = r.DB.CancelOverlappingMaintenance(ctx, in.Product, in.Provider, in.SKU, in.StartsAt, in.EndsAt, "admin_replace")
	} else {
		overlap, err := r.DB.HasOverlappingMaintenance(ctx, in.Product, in.Provider, in.SKU, in.StartsAt, in.EndsAt)
		if err != nil {
			return SetMaintenanceOutput{}, err
		}
		if overlap {
			return SetMaintenanceOutput{}, newErr("overlap", "Cửa sổ bảo trì chồng lấn với cửa sổ hiện có")
		}
	}

	status := in.Status
	if status == "" {
		status = "scheduled"
		if !in.StartsAt.After(time.Now()) {
			status = "active"
		}
	}
	now := time.Now()
	mid := buildMaintenanceID(in.Provider, in.Seq, now)
	if err := r.DB.InsertMaintenance(ctx, mid, in.Product, in.Provider, in.SKU, in.StartsAt, in.EndsAt, status, trigger, in.Reason, in.CycleID); err != nil {
		return SetMaintenanceOutput{}, err
	}

	// Send notification
	if !in.SkipNotify {
		go func() {
			actor := in.Actor
			if actor == "" {
				actor = "OpsOne"
			}
			_ = r.Notify.SendIfNeeded(context.Background(), notify.EmailParams{
				Product:       in.Product,
				Provider:      in.Provider,
				SKU:           in.SKU,
				HealthStatus:  "yellow", // Maintenance is usually a warning
				TriggerEvent:  "maintenance_active",
				ActionSummary: fmt.Sprintf("Bắt đầu bảo trì (%s)", mid),
				Reason:        in.Reason,
				MaintenanceID: &mid,
				CycleID:       in.CycleID,
				Actor:         actor,
			})
		}()
	}

	ds, _ := r.dataSource(ctx)
	return SetMaintenanceOutput{
		MaintenanceID: mid,
		Status:        status,
		Product:       in.Product,
		Provider:      in.Provider,
		StartsAt:      in.StartsAt,
		EndsAt:        in.EndsAt,
		Meta:          Meta{DataSource: ds, QueriedAt: now},
	}, nil
}

// buildMaintenanceID returns a unique maintenance_id (<= 32 chars) per provider + seq.
func buildMaintenanceID(provider string, seq int, now time.Time) string {
	prov := strings.ToUpper(strings.TrimSpace(provider))
	if len(prov) > 6 {
		prov = prov[:6]
	}
	if seq < 0 {
		seq = 0
	}
	if seq > 99 {
		seq = seq % 100
	}
	ms := now.Nanosecond() / 1_000_000
	// MT + yyMMddHHmmss (12) + provider (<=6) + seq (2) + ms (3) => max 25 chars
	id := fmt.Sprintf("MT%s%s%02d%03d", now.Format("060102150405"), prov, seq, ms)
	if len(id) > 32 {
		id = id[:32]
	}
	return id
}
