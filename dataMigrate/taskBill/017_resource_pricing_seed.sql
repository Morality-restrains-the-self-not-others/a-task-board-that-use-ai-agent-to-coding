-- 017: 资源定价种子数据 — 确保 billing_unit 表有完整的最新定价
-- 不再依赖价格套餐，billing_unit 是定价的唯一来源

-- 更新已有 billing_unit 行到合理默认值（如果仍是旧的种子值）
UPDATE billing_unit SET name = '任务', price = 33, unit = '次', updated_at = NOW()
  WHERE unit_type = 'server_start' AND price <= 30;

UPDATE billing_unit SET name = 'GitLab 磁盘', price = 800, unit = 'GB/月', updated_at = NOW()
  WHERE unit_type = 'gitlab_disk' AND price <= 1;

-- 确保 gitlab_traffic 存在（如果不存在则插入）
INSERT IGNORE INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
VALUES (1000000000000000008, 'gitlab_traffic', 'GitLab 流量费', 100, 'GB', 1, NOW(), NOW());

-- 确保 server_start 存在（如果 getCurrentResourcePricing 要用）
INSERT IGNORE INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
VALUES (1000000000000000009, 'server_start', '任务', 33, '次', 1, NOW(), NOW());

-- 确保 gitlab_disk 存在
INSERT IGNORE INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
VALUES (1000000000000000010, 'gitlab_disk', 'GitLab 磁盘', 800, 'GB/月', 1, NOW(), NOW());
