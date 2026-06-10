package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Config holds environment-backed settings for Phase 0+.
type Config struct {
	MySQLDSN          string
	DataSource        string
	APIAddr           string
	CORSOrigin        string
	AgentLocale       string
	AppTimezone       string
	SMTPHost          string
	SMTPPort          string
	SMTPUser          string
	SMTPPass          string
	SMTPFrom          string
	NotificationMock  bool
	LLMAPIURL         string
	LLMAPIKey         string
	DevAuthBypass     bool
}

// DefaultTimezone is Vietnam local time for OpsOne (vi-VN, UTC+7).
const DefaultTimezone = "Asia/Ho_Chi_Minh"

// Load reads configuration from environment variables with defaults from §2.1.
func Load() Config {
	tz := envOr("APP_TIMEZONE", DefaultTimezone)
	return Config{
		MySQLDSN:         ensureDSNTimezone(envOr("MYSQL_DSN", defaultMySQLDSN(tz)), tz),
		DataSource:       envOr("DATA_SOURCE", "mock"),
		APIAddr:          envOr("API_ADDR", ":8080"),
		CORSOrigin:       envOr("CORS_ORIGIN", "http://localhost:5173"),
		AgentLocale:      envOr("AGENT_LOCALE", "vi-VN"),
		AppTimezone:      tz,
		SMTPHost:         envOr("SMTP_HOST", "localhost"),
		SMTPPort:         envOr("SMTP_PORT", "1025"),
		SMTPUser:         envOr("SMTP_USER", ""),
		SMTPPass:         envOr("SMTP_PASS", ""),
		SMTPFrom:         envOr("SMTP_FROM", "opsone@company.local"),
		NotificationMock: envOr("NOTIFICATION_MOCK", "false") == "true",
		LLMAPIURL:      envOr("LLM_API_URL", ""),
		LLMAPIKey:      envOr("LLM_API_KEY", ""),
		DevAuthBypass:  envOr("DEV_AUTH_BYPASS", "true") == "true",
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultMySQLDSN(tz string) string {
	return "app:secret@tcp(localhost:3306)/traffic_agent?parseTime=true&charset=utf8mb4&loc=" + url.QueryEscape(tz)
}

// ensureDSNTimezone appends loc= if MYSQL_DSN omits it (prevents UTC storage vs local clock).
func ensureDSNTimezone(dsn, tz string) string {
	if strings.Contains(dsn, "loc=") {
		return dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "loc=" + url.QueryEscape(tz)
}

// Validate returns an error if required fields are missing.
func (c Config) Validate() error {
	if c.MySQLDSN == "" {
		return fmt.Errorf("MYSQL_DSN is required")
	}
	return nil
}
