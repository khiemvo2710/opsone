-- OpsOne: clear runtime/transactional data, keep catalog + routing_config + agent_settings.
-- Usage: Invoke-OpsOneClearRuntime (Windows) or make clear-runtime

USE opsone;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

TRUNCATE TABLE chat_feedback;
TRUNCATE TABLE chat_interaction_log;
TRUNCATE TABLE chat_command_patterns;
TRUNCATE TABLE chat_few_shot_examples;
TRUNCATE TABLE chat_voice_corrections;
TRUNCATE TABLE chat_user_prefs;
TRUNCATE TABLE chat_messages;
TRUNCATE TABLE chat_sessions;
TRUNCATE TABLE chat_intent_stats;
TRUNCATE TABLE agent_change_log;
TRUNCATE TABLE maintenance_windows;
TRUNCATE TABLE recommendations;
TRUNCATE TABLE routing_plans;
TRUNCATE TABLE incidents;
TRUNCATE TABLE agent_state_history;
TRUNCATE TABLE health_status_product;
TRUNCATE TABLE agent_analysis_history;
TRUNCATE TABLE agent_analysis_cycles;
TRUNCATE TABLE mock_error_stats;
TRUNCATE TABLE mock_metrics;
TRUNCATE TABLE mock_generator_run;
TRUNCATE TABLE metrics_snapshot;
TRUNCATE TABLE notification_log;
TRUNCATE TABLE config_audit_log;
TRUNCATE TABLE routing_scope_state;

SET FOREIGN_KEY_CHECKS = 1;

SELECT 'runtime cleared' AS status;
SELECT COUNT(*) AS products FROM products;
SELECT COUNT(*) AS routing_rows FROM routing_config;
SELECT COUNT(*) AS pending_plans FROM routing_plans;
SELECT COUNT(*) AS cycles FROM agent_analysis_cycles;
