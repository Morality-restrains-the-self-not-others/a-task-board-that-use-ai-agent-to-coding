-- 040: 更新任务帖默认价格 100→55（¥1.00 → ¥0.55 / 帖 / 12个月）
-- 仅覆盖仍为上一版默认值 100 的行，不覆盖管理员已改价的目录。
-- 续存单位与创建帖同价。

UPDATE billing_unit
SET price = 55, updated_at = NOW()
WHERE unit_type IN ('server_start', 'server_start_renewal')
  AND price = 100;

INSERT INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
SELECT 990000000000000004, 'server_start_renewal', '任务帖续存', 55, '帖/12个月', 1, NOW(), NOW()
FROM DUAL
WHERE NOT EXISTS (
    SELECT 1 FROM billing_unit WHERE unit_type = 'server_start_renewal'
);

-- 验证查询（执行后可运行以确认修改结果）
-- SELECT unit_type, name, price, unit FROM billing_unit WHERE unit_type IN ('server_start', 'server_start_renewal');
