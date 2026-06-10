package store

import (
	"context"
	"fmt"
)

// AgentSettings mirrors agent_settings row id=1 (§13.4).
type AgentSettings struct {
	SchedulerEnabled      bool
	SchedulerIntervalMin  int
	DataSource            string
	MockEnabled           bool
	MockIntervalMin       int
	MockScenario          string
	MockRetentionHours    int
	AgentLocale           string
}

// GetAgentSettings loads agent_settings id=1.
func (db *DB) GetAgentSettings(ctx context.Context) (AgentSettings, error) {
	const query = `
		SELECT scheduler_enabled, scheduler_interval_min, data_source,
		       mock_enabled, mock_interval_min, mock_scenario, mock_retention_hours,
		       agent_locale
		FROM agent_settings
		WHERE id = 1`
	var s AgentSettings
	var schedEn, mockEn int
	err := db.QueryRowContext(ctx, query).Scan(
		&schedEn, &s.SchedulerIntervalMin, &s.DataSource,
		&mockEn, &s.MockIntervalMin, &s.MockScenario, &s.MockRetentionHours,
		&s.AgentLocale,
	)
	if err != nil {
		return AgentSettings{}, fmt.Errorf("get agent settings: %w", err)
	}
	s.SchedulerEnabled = schedEn == 1
	s.MockEnabled = mockEn == 1
	return s, nil
}
