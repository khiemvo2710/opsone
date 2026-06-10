-- OpsOne — seed data
-- Source: OpsOne.md §1.1, §13.10

USE traffic_agent;

INSERT INTO agent_settings (id, notification_recipients, agent_locale) VALUES (1, '["ops-team@company.com"]', 'vi-VN');

INSERT INTO providers (provider_code, label) VALUES
  ('ESALE', 'ESALE'),
  ('IMEDIA', 'IMEDIA'),
  ('SHOPPAY', 'SHOPPAY');

INSERT INTO products (product_code, label, service_type, routing_mode) VALUES
  ('ZING', 'Thẻ Zing', 'card', 'sku'),
  ('GARENA', 'Thẻ Garena', 'card', 'sku'),
  ('VINAPHONE', 'Thẻ Vinaphone', 'card', 'sku'),
  ('MOBIFONE', 'Thẻ Mobifone', 'card', 'sku'),
  ('VIETTEL', 'Thẻ Viettel', 'card', 'sku'),
  ('DATA_VINA', 'Data Vinaphone', 'topup_data', 'sku'),
  ('DATA_MOBI', 'Data Mobifone', 'topup_data', 'sku'),
  ('DATA_VIETTEL', 'Data Viettel', 'topup_data', 'sku'),
  ('TOPUP_VINA', 'Topup Vinaphone', 'topup', 'provider'),
  ('TOPUP_MOBI', 'Topup Mobifone', 'topup', 'provider'),
  ('TOPUP_VIETTEL', 'Topup Viettel', 'topup', 'provider');

-- Card SKUs (mệnh giá VND)
INSERT INTO product_skus (product_id, sku_code, label)
SELECT p.id, s.sku_code, s.label
FROM products p
JOIN (
  SELECT '10000' AS sku_code, '10.000đ' AS label UNION ALL
  SELECT '20000', '20.000đ' UNION ALL
  SELECT '50000', '50.000đ' UNION ALL
  SELECT '100000', '100.000đ' UNION ALL
  SELECT '200000', '200.000đ' UNION ALL
  SELECT '500000', '500.000đ'
) s
WHERE p.service_type = 'card';

-- Topup data SKUs
INSERT INTO product_skus (product_id, sku_code, label)
SELECT p.id, s.sku_code, s.label
FROM products p
JOIN (
  SELECT 'VNP20' AS sku_code, 'VNP20' AS label UNION ALL
  SELECT 'VNP50', 'VNP50' UNION ALL
  SELECT 'V50K', 'V50K' UNION ALL
  SELECT 'V100K', 'V100K'
) s
WHERE p.service_type = 'topup_data';

-- Every product × every provider
INSERT INTO product_providers (product_id, provider_id)
SELECT p.id, pr.id
FROM products p
CROSS JOIN providers pr;

-- Topup routing: provider-level (sku_code = '')
INSERT INTO routing_config (product_code, sku_code, provider_code, traffic_pct, updated_by)
SELECT p.product_code, '', pr.provider_code,
  CASE pr.provider_code
    WHEN 'ESALE' THEN 70.00
    WHEN 'IMEDIA' THEN 20.00
    ELSE 10.00
  END,
  'seed'
FROM products p
CROSS JOIN providers pr
WHERE p.service_type = 'topup';

-- Card / topup_data: per-SKU routing (default 70/20/10)
INSERT INTO routing_config (product_code, sku_code, provider_code, traffic_pct, updated_by)
SELECT p.product_code, ps.sku_code, pr.provider_code,
  CASE pr.provider_code
    WHEN 'ESALE' THEN 70.00
    WHEN 'IMEDIA' THEN 20.00
    ELSE 10.00
  END,
  'seed'
FROM products p
JOIN product_skus ps ON ps.product_id = p.id
CROSS JOIN providers pr
WHERE p.service_type IN ('card', 'topup_data');

-- Sample auto-routing time window (weekdays 08:00–18:00)
INSERT INTO auto_routing_time_windows (days_mask, start_time, end_time, enabled, sort_order)
VALUES ('mon,tue,wed,thu,fri', '08:00:00', '18:00:00', 1, 0);

INSERT INTO provider_chat_escalation (provider_code, chat_app_name, chat_group_name, mention_tags) VALUES
  ('ESALE', 'Microsoft Teams', '[OpsOne] ESALE Support', '@esale-oncall'),
  ('IMEDIA', 'Microsoft Teams', '[OpsOne] IMEDIA Support', '@imedia-ops'),
  ('SHOPPAY', 'Zalo', 'OpsOne x SHOPPAY', '@shopay-support @n2-lead');

-- Ngưỡng cảnh báo per product (§1.2) — sau khi có products
INSERT INTO product_alert_thresholds (
  product_code, success_rate_min_pct, pending_rate_max_pct, fail_rate_max_pct,
  fail_txn_count_max, error_event_count_max, metrics_window_min, consecutive_cycles_required
)
SELECT product_code, 80.00, 15.00, 10.00, 100, 50, 15, 2 FROM products;

UPDATE product_alert_thresholds
SET success_rate_min_pct = 85.00, pending_rate_max_pct = 12.00, fail_rate_max_pct = 8.00, fail_txn_count_max = 80
WHERE product_code IN ('ZING','GARENA','VINAPHONE','MOBIFONE','VIETTEL');

UPDATE product_alert_thresholds SET fail_txn_count_max = 120
WHERE product_code IN ('TOPUP_VINA','TOPUP_MOBI','TOPUP_VIETTEL');
