-- OpsOne kịch bản C: IMEDIA treo Garena 10000 (nhiều GD pending)
-- Chạy SAU seed.sql — xem OpsOne.md §10.2

USE opsone;

-- Đảm bảo scenario là imedia_garena_pending
UPDATE agent_settings
SET mock_scenario = 'imedia_garena_pending', mock_enabled = 1
WHERE id = 1;

-- Kịch bản này chủ yếu mô phỏng real-time qua Mock Generator.
-- Không cần cấu chỉnh routing_config vì IMEDIA vẫn đang bật và có traffic.
