-- OpsOne — MySQL schema
-- Source: OpsOne.md §13 | Engine: InnoDB | Charset: utf8mb4
-- Convention: topup uses sku_code = '' (empty string, not NULL)

CREATE DATABASE IF NOT EXISTS opsone
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE opsone;

-- Clear existing tables for a clean reset (§13)
SET FOREIGN_KEY_CHECKS = 0;
DROP TABLE IF EXISTS chat_messages, chat_sessions, notification_log, provider_chat_escalation, config_audit_log, auto_routing_time_windows, agent_change_log, maintenance_windows, recommendations, routing_plans, incidents, health_status_product, agent_analysis_history, agent_analysis_cycles, metrics_snapshot, mock_error_stats, mock_metrics, mock_generator_run, product_alert_thresholds, routing_config, product_skus, product_providers, providers, products, agent_settings;
SET FOREIGN_KEY_CHECKS = 1;

-- 13.2 Catalog
CREATE TABLE products (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_code  VARCHAR(32)  NOT NULL COMMENT 'ZING, TOPUP_VINA, ...',
  label         VARCHAR(128) NOT NULL,
  service_type  ENUM('card','topup_data','topup') NOT NULL,
  routing_mode  ENUM('sku','provider') NOT NULL,
  enabled       TINYINT(1)   NOT NULL DEFAULT 1,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_products_code (product_code),
  KEY idx_products_enabled (enabled, service_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE providers (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  provider_code VARCHAR(32) NOT NULL COMMENT 'ESALE, IMEDIA, SHOPPAY',
  label         VARCHAR(64) NOT NULL,
  enabled       TINYINT(1)  NOT NULL DEFAULT 1,
  UNIQUE KEY uk_providers_code (provider_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product_providers (
  product_id    INT UNSIGNED NOT NULL,
  provider_id   INT UNSIGNED NOT NULL,
  PRIMARY KEY (product_id, provider_id),
  CONSTRAINT fk_pp_product  FOREIGN KEY (product_id)  REFERENCES products(id),
  CONSTRAINT fk_pp_provider FOREIGN KEY (provider_id) REFERENCES providers(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product_skus (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_id    INT UNSIGNED NOT NULL,
  sku_code      VARCHAR(32)  NOT NULL COMMENT '20000, VNP50, ...',
  label         VARCHAR(128) NOT NULL,
  enabled       TINYINT(1)   NOT NULL DEFAULT 1,
  UNIQUE KEY uk_product_sku (product_id, sku_code),
  KEY idx_skus_product (product_id),
  CONSTRAINT fk_skus_product FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product_alert_thresholds (
  id                          INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_code                VARCHAR(32)  NOT NULL,
  enabled                     TINYINT(1)   NOT NULL DEFAULT 1,
  success_rate_min_pct        DECIMAL(5,2) NULL,
  pending_rate_max_pct        DECIMAL(5,2) NULL,
  fail_rate_max_pct           DECIMAL(5,2) NULL,
  fail_txn_count_max          INT UNSIGNED NULL,
  error_event_count_max       INT UNSIGNED NULL,
  pending_txn_count_max       INT UNSIGNED NULL,
  metrics_window_min          TINYINT UNSIGNED NOT NULL DEFAULT 15,
  consecutive_cycles_required TINYINT UNSIGNED NOT NULL DEFAULT 2,
  alert_email_enabled         TINYINT(1)   NOT NULL DEFAULT 1,
  updated_at                  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by                  VARCHAR(64)  NULL,
  UNIQUE KEY uk_alert_product (product_code),
  CONSTRAINT fk_alert_product FOREIGN KEY (product_code) REFERENCES products(product_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 13.3 Routing
CREATE TABLE routing_config (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_code  VARCHAR(32)  NOT NULL,
  sku_code      VARCHAR(32)  NOT NULL DEFAULT '' COMMENT 'empty = topup provider-level',
  provider_code VARCHAR(32)  NOT NULL,
  traffic_pct   DECIMAL(5,2) NOT NULL COMMENT '0-100',
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by    VARCHAR(64)  NULL,
  UNIQUE KEY uk_routing (product_code, sku_code, provider_code),
  KEY idx_routing_product (product_code, sku_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 13.4 Agent settings
CREATE TABLE agent_settings (
  id                      TINYINT UNSIGNED NOT NULL PRIMARY KEY DEFAULT 1,
  scheduler_enabled       TINYINT(1)       NOT NULL DEFAULT 1,
  scheduler_interval_min  TINYINT UNSIGNED NOT NULL DEFAULT 5,
  data_source             ENUM('mock','production') NOT NULL DEFAULT 'mock',
  auto_routing_mode       ENUM('recommend_only','time_window','always') NOT NULL DEFAULT 'recommend_only',
  mock_enabled            TINYINT(1)       NOT NULL DEFAULT 1,
  mock_interval_min       TINYINT UNSIGNED NOT NULL DEFAULT 1,
  mock_scenario           ENUM('normal','esale_degrading','sku_local_fault','random_spike','imedia_garena_pending') NOT NULL DEFAULT 'normal',
  mock_retention_hours    SMALLINT UNSIGNED NOT NULL DEFAULT 24,
  maintenance_default_duration_min TINYINT UNSIGNED NOT NULL DEFAULT 60,
  maintenance_auto_enabled TINYINT(1)      NOT NULL DEFAULT 0,
  notification_enabled  TINYINT(1)       NOT NULL DEFAULT 1,
  notification_recipients JSON             NOT NULL,
  notification_on_red_only TINYINT(1)     NOT NULL DEFAULT 1,
  notification_pending_threshold DECIMAL(5,2) NOT NULL DEFAULT 15.00,
  notification_fail_threshold DECIMAL(5,2) NOT NULL DEFAULT 10.00,
  notification_cooldown_min TINYINT UNSIGNED NOT NULL DEFAULT 60,
  default_success_rate_min_pct DECIMAL(5,2) NOT NULL DEFAULT 80.00,
  default_pending_rate_max_pct DECIMAL(5,2) NOT NULL DEFAULT 15.00,
  default_fail_rate_max_pct DECIMAL(5,2) NOT NULL DEFAULT 10.00,
  default_fail_txn_count_max INT UNSIGNED NOT NULL DEFAULT 100,
  default_error_event_count_max INT UNSIGNED NOT NULL DEFAULT 50,
  default_pending_txn_count_max INT UNSIGNED NULL,
  default_metrics_window_min TINYINT UNSIGNED NOT NULL DEFAULT 15,
  default_consecutive_cycles_required TINYINT UNSIGNED NOT NULL DEFAULT 2,
  agent_locale            VARCHAR(8)       NOT NULL DEFAULT 'vi-VN',
  updated_at              DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  updated_by              VARCHAR(64)      NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE auto_routing_time_windows (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  days_mask     VARCHAR(16)  NOT NULL COMMENT 'mon,tue,wed,...',
  start_time    TIME         NOT NULL,
  end_time      TIME         NOT NULL,
  enabled       TINYINT(1)   NOT NULL DEFAULT 1,
  sort_order    SMALLINT     NOT NULL DEFAULT 0,
  KEY idx_time_windows_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE config_audit_log (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  changed_by    VARCHAR(64)  NOT NULL,
  change_type   VARCHAR(32)  NOT NULL COMMENT 'agent_settings, time_window',
  before_json   JSON         NULL,
  after_json    JSON         NOT NULL,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_config_audit_time (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE provider_chat_escalation (
  id              INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  provider_code   VARCHAR(32)  NOT NULL,
  chat_app_name   VARCHAR(64)  NOT NULL,
  chat_group_name VARCHAR(128) NOT NULL,
  mention_tags    VARCHAR(256) NOT NULL,
  enabled         TINYINT(1)   NOT NULL DEFAULT 1,
  updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_escalation_provider (provider_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE notification_log (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  dedupe_key          VARCHAR(128) NOT NULL,
  trigger_event       ENUM('routing_applied','maintenance_active','maintenance_scheduled') NOT NULL,
  health_status       ENUM('green','yellow','red') NOT NULL,
  product_code        VARCHAR(32)  NOT NULL,
  provider_code       VARCHAR(32)  NOT NULL,
  sku_code            VARCHAR(32)  NOT NULL DEFAULT '',
  cycle_id            BIGINT UNSIGNED NULL,
  incident_id         VARCHAR(32)  NULL,
  agent_change_id     BIGINT UNSIGNED NULL,
  maintenance_id      VARCHAR(32)  NULL,
  metrics_snapshot    JSON         NOT NULL COMMENT 'rates, counts, breach_reasons',
  action_summary      TEXT         NOT NULL,
  chat_escalation_json JSON        NOT NULL,
  recipients          JSON         NOT NULL,
  subject             VARCHAR(256) NOT NULL,
  status              ENUM('pending','sent','failed') NOT NULL DEFAULT 'pending',
  error_message       VARCHAR(512) NULL,
  sent_at             DATETIME     NULL,
  created_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_notification_dedupe (dedupe_key),
  KEY idx_notification_time (created_at),
  KEY idx_notification_product (product_code, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 13.5 Mock data
CREATE TABLE mock_generator_run (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  started_at    DATETIME     NOT NULL,
  finished_at   DATETIME     NULL,
  scenario      VARCHAR(32)  NOT NULL,
  rows_metrics  INT UNSIGNED NOT NULL DEFAULT 0,
  status        ENUM('running','success','failed') NOT NULL DEFAULT 'running',
  error_message VARCHAR(512) NULL,
  KEY idx_mock_run_started (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE mock_metrics (
  id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  recorded_at        DATETIME       NOT NULL,
  product_code       VARCHAR(32)    NOT NULL,
  sku_code           VARCHAR(32)    NOT NULL DEFAULT '',
  provider_code      VARCHAR(32)    NOT NULL,
  success_rate       DECIMAL(5,2)   NOT NULL,
  pending_rate       DECIMAL(5,2)   NOT NULL,
  fail_rate          DECIMAL(5,2)   NOT NULL,
  total_transactions INT UNSIGNED   NOT NULL DEFAULT 0,
  revenue_last_hour  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  generator_run_id   BIGINT UNSIGNED NULL,
  KEY idx_mock_metrics_lookup (product_code, provider_code, sku_code, recorded_at),
  KEY idx_mock_metrics_time (recorded_at),
  CONSTRAINT fk_mock_metrics_run FOREIGN KEY (generator_run_id) REFERENCES mock_generator_run(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE mock_error_stats (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  recorded_at     DATETIME     NOT NULL,
  product_code    VARCHAR(32)  NOT NULL,
  provider_code   VARCHAR(32)  NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '',
  error_code      VARCHAR(16)  NOT NULL,
  error_count     INT UNSIGNED NOT NULL,
  generator_run_id BIGINT UNSIGNED NULL,
  KEY idx_mock_errors_lookup (product_code, provider_code, recorded_at),
  KEY idx_mock_errors_time (recorded_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 13.6 Production metrics
CREATE TABLE metrics_snapshot (
  id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  recorded_at        DATETIME       NOT NULL,
  product_code       VARCHAR(32)    NOT NULL,
  sku_code           VARCHAR(32)    NOT NULL DEFAULT '',
  provider_code      VARCHAR(32)    NOT NULL,
  success_rate       DECIMAL(5,2)   NOT NULL,
  pending_rate       DECIMAL(5,2)   NOT NULL,
  fail_rate          DECIMAL(5,2)   NOT NULL,
  total_transactions INT UNSIGNED   NOT NULL DEFAULT 0,
  revenue_last_hour  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  KEY idx_metrics_lookup (product_code, provider_code, sku_code, recorded_at),
  KEY idx_metrics_time (recorded_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 13.7 Agent analysis history
CREATE TABLE agent_analysis_cycles (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_started   DATETIME     NOT NULL,
  cycle_finished  DATETIME     NULL,
  data_source     ENUM('mock','production') NOT NULL,
  health_status   ENUM('green','yellow','red') NOT NULL DEFAULT 'green',
  health_summary  VARCHAR(512) NULL,
  decision        VARCHAR(32)  NULL COMMENT 'monitor, incident, shift_traffic, maintenance',
  status          ENUM('running','success','failed') NOT NULL DEFAULT 'running',
  KEY idx_cycles_started (cycle_started),
  KEY idx_cycles_health (health_status, cycle_started)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE agent_analysis_history (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id        BIGINT UNSIGNED NOT NULL,
  recorded_at     DATETIME     NOT NULL,
  product_code    VARCHAR(32)  NOT NULL,
  service_type    ENUM('card','topup_data','topup') NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '',
  provider_code   VARCHAR(32)  NOT NULL,
  success_rate    DECIMAL(5,2) NOT NULL,
  pending_rate    DECIMAL(5,2) NOT NULL,
  fail_rate       DECIMAL(5,2) NOT NULL,
  KEY idx_history_cycle (cycle_id),
  KEY idx_history_lookup (product_code, provider_code, sku_code, recorded_at),
  CONSTRAINT fk_history_cycle FOREIGN KEY (cycle_id) REFERENCES agent_analysis_cycles(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE health_status_product (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id        BIGINT UNSIGNED NOT NULL,
  product_code    VARCHAR(32)  NOT NULL,
  health_status   ENUM('green','yellow','red') NOT NULL,
  health_summary  VARCHAR(256) NULL,
  UNIQUE KEY uk_health_cycle_product (cycle_id, product_code),
  CONSTRAINT fk_health_cycle FOREIGN KEY (cycle_id) REFERENCES agent_analysis_cycles(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 13.8 Output
CREATE TABLE incidents (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  incident_id     VARCHAR(32)  NOT NULL COMMENT '20260604-001',
  cycle_id        BIGINT UNSIGNED NULL,
  severity        ENUM('low','medium','high') NOT NULL,
  product_code    VARCHAR(32)  NOT NULL,
  provider_code   VARCHAR(32)  NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '',
  sku_label       VARCHAR(64)  NULL,
  success_before  DECIMAL(5,2) NULL,
  success_after   DECIMAL(5,2) NULL,
  fail_before     DECIMAL(5,2) NULL,
  fail_after      DECIMAL(5,2) NULL,
  summary         TEXT         NULL,
  status          ENUM('open','acknowledged','resolved') NOT NULL DEFAULT 'open',
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_incident_id (incident_id),
  KEY idx_incidents_product (product_code, created_at),
  KEY idx_incidents_status (status, severity)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE routing_plans (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id        BIGINT UNSIGNED NULL,
  product_code    VARCHAR(32)  NOT NULL,
  scope           ENUM('provider','sku','sku_batch') NOT NULL,
  sku_code        VARCHAR(32)  NOT NULL DEFAULT '',
  plan_json       JSON         NOT NULL COMMENT 'current, performance, suggested, expected',
  status          ENUM('draft','pending_approve','approved','rejected','executed') NOT NULL DEFAULT 'draft',
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  approved_by     VARCHAR(64)  NULL,
  approved_at     DATETIME     NULL,
  KEY idx_routing_plans_product (product_code, created_at),
  KEY idx_routing_plans_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE recommendations (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cycle_id        BIGINT UNSIGNED NULL,
  incident_id     VARCHAR(32)  NULL,
  product_code    VARCHAR(32)  NOT NULL,
  action_type     VARCHAR(32)  NOT NULL COMMENT 'shift_traffic, monitor, escalate, maintenance',
  action_detail   TEXT         NOT NULL,
  monitor_until   DATETIME     NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_recommendations_product (product_code, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE maintenance_windows (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  maintenance_id    VARCHAR(32)  NOT NULL COMMENT '20260604-M001',
  product_code      VARCHAR(32)  NOT NULL,
  provider_code     VARCHAR(32)  NOT NULL,
  sku_code          VARCHAR(32)  NOT NULL DEFAULT '',
  starts_at         DATETIME     NOT NULL,
  ends_at           DATETIME     NOT NULL,
  status            ENUM('pending_approve','scheduled','active','completed','cancelled') NOT NULL DEFAULT 'pending_approve',
  trigger_type      ENUM('opsone_recommend','opsone_auto','admin_manual') NOT NULL,
  severity          ENUM('low','medium','high') NULL,
  reason            TEXT         NULL,
  cycle_id          BIGINT UNSIGNED NULL,
  incident_id       VARCHAR(32)  NULL,
  approved_by       VARCHAR(64)  NULL,
  approved_at       DATETIME     NULL,
  cancelled_by      VARCHAR(64)  NULL,
  cancelled_at      DATETIME     NULL,
  created_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_maintenance_id (maintenance_id),
  KEY idx_maintenance_product (product_code, status, starts_at),
  KEY idx_maintenance_active (status, ends_at),
  KEY idx_maintenance_window (product_code, provider_code, sku_code, starts_at, ends_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE agent_change_log (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  change_type       ENUM('routing') NOT NULL DEFAULT 'routing',
  product_code      VARCHAR(32)  NOT NULL,
  scope             ENUM('provider','sku','sku_batch') NOT NULL,
  sku_code          VARCHAR(32)  NOT NULL DEFAULT '',
  routing_before    JSON         NOT NULL,
  routing_after     JSON         NOT NULL,
  trigger_type      ENUM('auto','admin_approve','manual','rollback') NOT NULL,
  change_status     ENUM('applied','rolled_back','superseded') NOT NULL DEFAULT 'applied',
  cycle_id          BIGINT UNSIGNED NULL,
  routing_plan_id   BIGINT UNSIGNED NULL,
  incident_id       VARCHAR(32)  NULL,
  rollback_of_id    BIGINT UNSIGNED NULL,
  reason            TEXT         NULL,
  executed_by       VARCHAR(64)  NULL,
  rolled_back_at    DATETIME     NULL,
  rolled_back_by    VARCHAR(64)  NULL,
  created_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_agent_changes_product (product_code, created_at),
  KEY idx_agent_changes_status (change_status, product_code, scope, sku_code, created_at),
  KEY idx_agent_changes_cycle (cycle_id),
  CONSTRAINT fk_agent_change_rollback FOREIGN KEY (rollback_of_id) REFERENCES agent_change_log(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 13.9 Chat
CREATE TABLE chat_sessions (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id         VARCHAR(64)  NOT NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_chat_sessions_user (user_id, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_messages (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  session_id      BIGINT UNSIGNED NOT NULL,
  role            ENUM('user','assistant') NOT NULL,
  content         TEXT         NOT NULL,
  input_source    ENUM('text','voice') NOT NULL DEFAULT 'text',
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_chat_messages_session (session_id, created_at),
  CONSTRAINT fk_chat_messages_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
