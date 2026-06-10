package config_test

import (
	"strings"
	"testing"

	"opsone/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	cfg := config.Load()
	if cfg.MySQLDSN == "" {
		t.Error("MySQLDSN should have default")
	}
	if cfg.DataSource != "mock" {
		t.Errorf("DataSource = %q, want mock", cfg.DataSource)
	}
	if cfg.AgentLocale != "vi-VN" {
		t.Errorf("AgentLocale = %q, want vi-VN", cfg.AgentLocale)
	}
	if cfg.AppTimezone != config.DefaultTimezone {
		t.Errorf("AppTimezone = %q, want %s", cfg.AppTimezone, config.DefaultTimezone)
	}
	if !strings.Contains(cfg.MySQLDSN, "loc=Asia") {
		t.Errorf("MySQLDSN should include loc=Asia/Ho_Chi_Minh, got %q", cfg.MySQLDSN)
	}
}

func TestValidate(t *testing.T) {
	if err := (config.Config{}).Validate(); err == nil {
		t.Error("empty config should fail validation")
	}
	if err := (config.Config{MySQLDSN: "x"}).Validate(); err != nil {
		t.Errorf("valid config should pass: %v", err)
	}
}
