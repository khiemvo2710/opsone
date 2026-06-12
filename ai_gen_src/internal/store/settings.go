package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// AgentSettings mirrors agent_settings row id=1 (§13.4).
type AgentSettings struct {
	SchedulerEnabled              bool
	SchedulerIntervalMin          int
	DataSource                    string
	MockEnabled                   bool
	MockIntervalMin               int
	MockScenario                  string
	MockRetentionHours            int
	MaintenanceDefaultDurationMin int
	AgentLocale                   string
	SmtpSender                    string
	NotificationRecipients        []string
}

// NormalizeMaintenanceDefaultDurationMin returns configured minutes or 60.
func NormalizeMaintenanceDefaultDurationMin(v int) int {
	if v <= 0 {
		return 60
	}
	return v
}

// GetAgentSettings loads agent_settings id=1.
func (db *DB) GetAgentSettings(ctx context.Context) (AgentSettings, error) {
	const query = `
		SELECT scheduler_enabled, scheduler_interval_min, data_source,
		       mock_enabled, mock_interval_min, mock_scenario, mock_retention_hours,
		       maintenance_default_duration_min, agent_locale,
		       smtp_sender, notification_recipients
		FROM agent_settings
		WHERE id = 1`
	var s AgentSettings
	var schedEn, mockEn int
	var recipients []byte
	err := db.QueryRowContext(ctx, query).Scan(
		&schedEn, &s.SchedulerIntervalMin, &s.DataSource,
		&mockEn, &s.MockIntervalMin, &s.MockScenario, &s.MockRetentionHours,
		&s.MaintenanceDefaultDurationMin, &s.AgentLocale,
		&s.SmtpSender, &recipients,
	)
	if err != nil {
		return AgentSettings{}, fmt.Errorf("get agent settings: %w", err)
	}
	s.SchedulerEnabled = schedEn == 1
	s.MockEnabled = mockEn == 1

	if len(recipients) > 0 {
		_ = json.Unmarshal(recipients, &s.NotificationRecipients)
	}

	return s, nil
}
