-- 默认定价套餐（永久有效，package_number=0）
INSERT IGNORE INTO billing_pricing_package (
    id, package_number, valid_from, valid_to,
    normal_task_points, programming_task_points,
    normal_task_renewal_points_per_month, programming_task_renewal_points_per_month,
    created_at, updated_at
) VALUES (
    1000000000000000001, 0, '2025-01-01', NULL,
    3, 30, 1, 8,
    NOW(), NOW()
);

-- 默认计费单元：任务帖创建
INSERT IGNORE INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
VALUES (1000000000000000002, 'post_creation', '任务帖创建', 3, '次', 1, NOW(), NOW());

-- 默认计费单元：服务器启动
INSERT IGNORE INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
VALUES (1000000000000000003, 'server_start', '智能体任务', 30, '次', 1, NOW(), NOW());
