-- 036: 更新任务帖默认价格 55→33（¥0.55 → ¥0.33）
-- 修改 billing_unit、billing_account 中的任务帖定价

-- 更新 billing_unit 定价
UPDATE billing_unit
SET price = 33, updated_at = NOW()
WHERE unit_type = 'server_start' AND price = 55;

-- 更新 billing_account 锁定价（条件执行：仅在列存在时更新）
SET @stmt = IF(
    (SELECT COUNT(*) FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'billing_account'
       AND COLUMN_NAME = 'locked_server_start_points') > 0,
    'UPDATE billing_account SET locked_server_start_points = 33, updated_at = NOW() WHERE locked_server_start_points = 55',
    'SELECT 1 AS skipped_locked_server_start_points_update'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

-- 验证查询（执行后可运行以确认修改结果）
-- SELECT unit_type, name, price FROM billing_unit WHERE unit_type = 'server_start';
-- SELECT COUNT(*) AS accounts_updated FROM billing_account WHERE locked_server_start_points = 33;
