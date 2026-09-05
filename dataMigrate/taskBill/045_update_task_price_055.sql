-- 045: 更新任务帖默认价格 33→55（¥0.33 → ¥0.55 / 帖 / 12个月）
-- 仅覆盖仍为上一版默认值 33 的创建帖行，不覆盖管理员已改价的目录。
-- 续存单位保持 0（041），本迁移不改 server_start_renewal。

UPDATE billing_unit
SET price = 55, updated_at = NOW()
WHERE unit_type = 'server_start' AND price = 33;

-- 验证查询（执行后可运行以确认修改结果）
-- SELECT unit_type, name, price, unit
-- FROM billing_unit WHERE unit_type = 'server_start';
