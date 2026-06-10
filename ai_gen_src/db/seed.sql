-- OpsOne seed — §13.10 OpsOne.md
-- Encoding: UTF-8 (Thẻ, đ, …). Import bằng Invoke-OpsOneReset → docker cp (không pipe stdin).
USE traffic_agent;

SET NAMES utf8mb4;

-- Agent settings
INSERT INTO agent_settings (id, notification_recipients, agent_locale) VALUES
  (1, '["ops-team@company.com"]', 'vi-VN');

-- Providers
INSERT INTO providers (provider_code, label) VALUES
  ('ESALE', 'ESALE'),
  ('IMEDIA', 'IMEDIA'),
  ('SHOPPAY', 'SHOPPAY');

-- Products (11)
INSERT INTO products (product_code, label, service_type, routing_mode) VALUES
  ('ZING',         'Thẻ Zing',         'card',       'sku'),
  ('GARENA',       'Thẻ Garena',       'card',       'sku'),
  ('VINAPHONE',    'Thẻ Vinaphone',    'card',       'sku'),
  ('MOBIFONE',     'Thẻ Mobifone',     'card',       'sku'),
  ('VIETTEL',      'Thẻ Viettel',      'card',       'sku'),
  ('DATA_VINA',    'Data Vinaphone',   'topup_data', 'sku'),
  ('DATA_MOBI',    'Data Mobifone',    'topup_data', 'sku'),
  ('DATA_VIETTEL', 'Data Viettel',     'topup_data', 'sku'),
  ('TOPUP_VINA',   'Topup Vinaphone',  'topup',      'provider'),
  ('TOPUP_MOBI',   'Topup Mobifone',   'topup',      'provider'),
  ('TOPUP_VIETTEL','Topup Viettel',    'topup',      'provider');

-- Product × Provider (all enabled by default)
INSERT INTO product_providers (product_id, provider_id, enabled)
SELECT p.id, pr.id, 1
FROM products p
CROSS JOIN providers pr;

-- Card SKUs (5 card products × 4 denominations)
INSERT INTO product_skus (product_id, sku_code, label)
SELECT p.id, s.sku_code, s.label
FROM products p
CROSS JOIN (
  SELECT '10000'  AS sku_code, '10.000đ' AS label UNION ALL
  SELECT '20000',  '20.000đ' UNION ALL
  SELECT '50000',  '50.000đ' UNION ALL
  SELECT '100000', '100.000đ'
) s
WHERE p.service_type = 'card';

-- Topup data SKUs (3 products × 4 packages)
INSERT INTO product_skus (product_id, sku_code, label)
SELECT p.id, s.sku_code, s.label
FROM products p
CROSS JOIN (
  SELECT 'VNP20' AS sku_code, 'VNP20' AS label UNION ALL
  SELECT 'VNP50', 'VNP50' UNION ALL
  SELECT 'V50K',  'V50K' UNION ALL
  SELECT 'V100K', 'V100K'
) s
WHERE p.service_type = 'topup_data';

-- Topup provider-level routing (sku_code = '')
INSERT INTO routing_config (product_code, sku_code, provider_code, baseline_pct, traffic_pct, baseline_updated_by, updated_by) VALUES
  ('TOPUP_VINA',    '', 'ESALE',   70.00, 70.00, 'business', 'business'),
  ('TOPUP_VINA',    '', 'IMEDIA',  20.00, 20.00, 'business', 'business'),
  ('TOPUP_VINA',    '', 'SHOPPAY', 10.00, 10.00, 'business', 'business'),
  ('TOPUP_MOBI',    '', 'ESALE',   60.00, 60.00, 'business', 'business'),
  ('TOPUP_MOBI',    '', 'IMEDIA',  30.00, 30.00, 'business', 'business'),
  ('TOPUP_MOBI',    '', 'SHOPPAY', 10.00, 10.00, 'business', 'business'),
  ('TOPUP_VIETTEL', '', 'ESALE',   50.00, 50.00, 'business', 'business'),
  ('TOPUP_VIETTEL', '', 'IMEDIA',  30.00, 30.00, 'business', 'business'),
  ('TOPUP_VIETTEL', '', 'SHOPPAY', 20.00, 20.00, 'business', 'business');

-- Card routing baseline (same pattern for all card products)
INSERT INTO routing_config (product_code, sku_code, provider_code, baseline_pct, traffic_pct, baseline_updated_by, updated_by)
SELECT p.product_code, r.sku_code, r.provider_code, r.baseline_pct, r.traffic_pct, 'business', 'business'
FROM products p
CROSS JOIN (
  SELECT '10000' AS sku_code, 'ESALE' AS provider_code, 60.00 AS baseline_pct, 60.00 AS traffic_pct UNION ALL
  SELECT '10000', 'IMEDIA', 40.00, 40.00 UNION ALL
  SELECT '20000', 'ESALE', 80.00, 80.00 UNION ALL
  SELECT '20000', 'IMEDIA', 20.00, 20.00 UNION ALL
  SELECT '50000', 'ESALE', 50.00, 50.00 UNION ALL
  SELECT '50000', 'IMEDIA', 30.00, 30.00 UNION ALL
  SELECT '50000', 'SHOPPAY', 20.00, 20.00 UNION ALL
  SELECT '100000', 'ESALE', 40.00, 40.00 UNION ALL
  SELECT '100000', 'IMEDIA', 35.00, 35.00 UNION ALL
  SELECT '100000', 'SHOPPAY', 25.00, 25.00
) r
WHERE p.service_type = 'card';

-- Topup data routing baseline
INSERT INTO routing_config (product_code, sku_code, provider_code, baseline_pct, traffic_pct, baseline_updated_by, updated_by)
SELECT p.product_code, r.sku_code, r.provider_code, r.baseline_pct, r.traffic_pct, 'business', 'business'
FROM products p
CROSS JOIN (
  SELECT 'VNP20' AS sku_code, 'ESALE' AS provider_code, 70.00 AS baseline_pct, 70.00 AS traffic_pct UNION ALL
  SELECT 'VNP20', 'IMEDIA', 30.00, 30.00 UNION ALL
  SELECT 'VNP50', 'ESALE', 70.00, 70.00 UNION ALL
  SELECT 'VNP50', 'IMEDIA', 30.00, 30.00 UNION ALL
  SELECT 'V50K',  'ESALE', 60.00, 60.00 UNION ALL
  SELECT 'V50K',  'IMEDIA', 40.00, 40.00 UNION ALL
  SELECT 'V100K', 'ESALE', 50.00, 50.00 UNION ALL
  SELECT 'V100K', 'IMEDIA', 30.00, 30.00 UNION ALL
  SELECT 'V100K', 'SHOPPAY', 20.00, 20.00
) r
WHERE p.service_type = 'topup_data';

-- Routing scope state (topup products — provider-level scope)
INSERT INTO routing_scope_state (product_code, sku_code, pending_restore)
SELECT product_code, '', 0 FROM products WHERE routing_mode = 'provider';

-- Routing scope state (sku products — one row per SKU)
INSERT INTO routing_scope_state (product_code, sku_code, pending_restore)
SELECT p.product_code, ps.sku_code, 0
FROM products p
JOIN product_skus ps ON ps.product_id = p.id
WHERE p.routing_mode = 'sku';

-- Chat escalation
INSERT INTO provider_chat_escalation (provider_code, chat_app_name, chat_group_name, mention_tags) VALUES
  ('ESALE',   'Microsoft Teams', '[OpsOne] ESALE Support',   '@esale-oncall'),
  ('IMEDIA',  'Microsoft Teams', '[OpsOne] IMEDIA Support',  '@imedia-ops'),
  ('SHOPPAY', 'Zalo',            'OpsOne x SHOPPAY',         '@shopay-support @n2-lead');

-- Alert thresholds (defaults for all products)
INSERT INTO product_alert_thresholds (
  product_code, success_rate_min_pct, pending_rate_max_pct, fail_rate_max_pct,
  fail_txn_count_max, pending_txn_count_max, error_event_count_max, metrics_window_min, consecutive_cycles_required
)
SELECT product_code, 80.00, 15.00, 10.00, 50, 5, 50, 15, 2 FROM products;

-- Product-specific overrides (§1.2) — số GD có thể chỉnh trên UI Cấu hình
UPDATE product_alert_thresholds SET success_rate_min_pct = 80.00, fail_txn_count_max = 120 WHERE product_code = 'TOPUP_VINA';
UPDATE product_alert_thresholds SET success_rate_min_pct = 85.00, pending_rate_max_pct = 12.00, fail_rate_max_pct = 8.00, fail_txn_count_max = 80, pending_txn_count_max = 10 WHERE product_code = 'ZING';
UPDATE product_alert_thresholds SET success_rate_min_pct = 82.00, pending_rate_max_pct = 14.00, fail_txn_count_max = 60, pending_txn_count_max = 8, consecutive_cycles_required = 1 WHERE product_code = 'DATA_VINA';
