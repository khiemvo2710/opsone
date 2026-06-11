package config

import (
	"bufio"
	"fmt"
	"log"
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
	LLMModel          string
	LLMTimeoutSec     int
	DevAuthBypass     bool
}

// DefaultGreenNodeLLMURL is GreenNode AIP MaaS OpenAI-compatible endpoint.
const DefaultGreenNodeLLMURL = "https://maas-llm-aiplatform-hcm.api.vngcloud.vn/v1"

// DefaultLLMModel is the GreenNode AIP model path for OpsOne chat (OpenAI-compatible).
const DefaultLLMModel = "minimax/minimax-m2.5"

// DefaultTimezone is Vietnam local time for OpsOne (vi-VN, UTC+7).
const DefaultTimezone = "Asia/Ho_Chi_Minh"

// Load reads configuration from environment variables with defaults from §2.1.
func Load() Config {
	loadDotEnv(".env")
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
		LLMAPIURL:      envOr("LLM_API_URL", envOr("LLM_BASE_URL", "")),
		LLMAPIKey:      envOr("LLM_API_KEY", envOr("AIP_API_KEY", "")),
		LLMModel:       normalizeLLMModel(envOr("LLM_MODEL", DefaultLLMModel)),
		LLMTimeoutSec:  envIntOr("LLM_TIMEOUT_SECONDS", 30),
		DevAuthBypass:  envOr("DEV_AUTH_BYPASS", "true") == "true",
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}

// LLMEnabled reports whether chat/agent LLM calls are configured.
func (c Config) LLMEnabled() bool {
	return strings.TrimSpace(c.LLMAPIKey) != ""
}

// LLMBaseURL returns the OpenAI-compatible MaaS base URL (with /v1).
func (c Config) LLMBaseURL() string {
	url := strings.TrimSpace(c.LLMAPIURL)
	if url == "" {
		return DefaultGreenNodeLLMURL
	}
	return strings.TrimRight(url, "/")
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 1 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

// normalizeLLMModel maps GreenNode AIP ids to lowercase (GET /v1/models is case-sensitive).
func normalizeLLMModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return DefaultLLMModel
	}
	return strings.ToLower(model)
}

func (c Config) LogLLMStartup() {
	if !c.LLMEnabled() {
		log.Println("LLM chat: disabled (set LLM_API_KEY in .env)")
		return
	}
	log.Printf("LLM chat: enabled model=%s url=%s", c.LLMModel, c.LLMBaseURL())
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
