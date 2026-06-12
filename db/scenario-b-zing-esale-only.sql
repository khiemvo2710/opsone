-- OpsOne kịch bản B: ZING chỉ provider ESALE (single-provider maintenance)
-- Chạy SAU seed.sql — xem OpsOne.md §10.2

USE opsone;

DELETE pp FROM product_providers pp
JOIN products p ON p.id = pp.product_id
JOIN providers pr ON pr.id = pp.provider_id
WHERE p.product_code = 'ZING' AND pr.provider_code IN ('IMEDIA', 'SHOPPAY');

DELETE FROM routing_config
WHERE product_code = 'ZING' AND provider_code IN ('IMEDIA', 'SHOPPAY');

-- ZING routing per-SKU: 100% ESALE sau thu hẹp
UPDATE routing_config
SET traffic_pct = 100.00, updated_by = 'scenario-b'
WHERE product_code = 'ZING' AND provider_code = 'ESALE';
