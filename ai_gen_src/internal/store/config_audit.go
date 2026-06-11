package store

import (
	"context"
	"encoding/json"
)

// AgentSettingsFull for API config GET.
type AgentSettingsFull struct {
	SchedulerEnabled              bool   `json:"scheduler_enabled"`
	SchedulerIntervalMin          int    `json:"scheduler_interval_min"`
	DataSource                    string `json:"data_source"`
	MockEnabled                   bool   `json:"mock_enabled"`
	MockIntervalMin               int    `json:"mock_interval_min"`
	MockScenario                  string `json:"mock_scenario"`
	MaintenanceDefaultDurationMin int    `json:"maintenance_default_duration_min"`
	AgentLocale                   string `json:"agent_locale"`
}

// GetAgentSettingsFull loads settings for API.
func (db *DB) GetAgentSettingsFull(ctx context.Context) (AgentSettingsFull, error) {
	const query = `
		SELECT scheduler_enabled, scheduler_interval_min, data_source,
		       mock_enabled, mock_interval_min, mock_scenario,
		       maintenance_default_duration_min, agent_locale
		FROM agent_settings WHERE id = 1`
	var s AgentSettingsFull
	var schedEn, mockEn int
	err := db.QueryRowContext(ctx, query).Scan(
		&schedEn, &s.SchedulerIntervalMin, &s.DataSource,
		&mockEn, &s.MockIntervalMin, &s.MockScenario,
		&s.MaintenanceDefaultDurationMin, &s.AgentLocale,
	)
	if err != nil {
		return AgentSettingsFull{}, err
	}
	s.SchedulerEnabled = schedEn == 1
	s.MockEnabled = mockEn == 1
	return s, nil
}

// ConfigUpdatePatch allowed PUT fields.
type ConfigUpdatePatch struct {
	SchedulerEnabled              *bool   `json:"scheduler_enabled"`
	SchedulerIntervalMin          *int    `json:"scheduler_interval_min"`
	MockEnabled                   *bool   `json:"mock_enabled"`
	MockScenario                  *string `json:"mock_scenario"`
	MaintenanceDefaultDurationMin *int    `json:"maintenance_default_duration_min"`
}

// ApplyConfigUpdate updates agent_settings id=1.
func (db *DB) ApplyConfigUpdate(ctx context.Context, p ConfigUpdatePatch) error {
	before, _ := db.GetAgentSettingsFull(ctx)
	if p.SchedulerEnabled != nil {
		v := 0
		if *p.SchedulerEnabled {
			v = 1
		}
		if _, err := db.ExecContext(ctx, `UPDATE agent_settings SET scheduler_enabled = ? WHERE id = 1`, v); err != nil {
			return err
		}
	}
	if p.SchedulerIntervalMin != nil {
		if _, err := db.ExecContext(ctx, `UPDATE agent_settings SET scheduler_interval_min = ? WHERE id = 1`, *p.SchedulerIntervalMin); err != nil {
			return err
		}
	}
	if p.MockEnabled != nil {
		v := 0
		if *p.MockEnabled {
			v = 1
		}
		if _, err := db.ExecContext(ctx, `UPDATE agent_settings SET mock_enabled = ? WHERE id = 1`, v); err != nil {
			return err
		}
	}
	if p.MockScenario != nil {
		if _, err := db.ExecContext(ctx, `UPDATE agent_settings SET mock_scenario = ? WHERE id = 1`, *p.MockScenario); err != nil {
			return err
		}
	}
	if p.MaintenanceDefaultDurationMin != nil {
		v := NormalizeMaintenanceDefaultDurationMin(*p.MaintenanceDefaultDurationMin)
		if _, err := db.ExecContext(ctx, `UPDATE agent_settings SET maintenance_default_duration_min = ? WHERE id = 1`, v); err != nil {
			return err
		}
	}
	after, _ := db.GetAgentSettingsFull(ctx)
	return db.InsertConfigAudit(ctx, "admin", "config_update", before, after)
}

// InsertConfigAudit writes config_audit_log.
func (db *DB) InsertConfigAudit(ctx context.Context, by, changeType string, before, after any) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	var b *[]byte
	if len(beforeJSON) > 2 {
		b = &beforeJSON
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO config_audit_log (changed_by, change_type, before_json, after_json)
		VALUES (?, ?, ?, ?)`, by, changeType, b, afterJSON)
	return err
}
