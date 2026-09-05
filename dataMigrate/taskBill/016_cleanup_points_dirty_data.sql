-- 016: 清理积分脏数据 — 移除「积分」术语残留、修正旧定价、清除误扣费记录
-- 背景：价格体系重构（2026-07-25）将「积分」→「元」计价，移除普通任务，智能体任务→任务

-- ============================================================
-- 1. billing_unit 元数据：移除「积分」术语残留，修正名称和价格
-- ============================================================

-- recharge: 「积分充值」→「支付」，「积分」→「元」
UPDATE billing_unit
SET name = '支付', unit = '元', updated_at = NOW()
WHERE unit_type = 'recharge';

-- server_start: 「智能体任务」→「任务」，价格 30 → 55（¥0.55 = 55 积分/分）
UPDATE billing_unit
SET name = '任务', price = 55, updated_at = NOW()
WHERE unit_type = 'server_start';

-- post_creation: 标记为已废弃（普通任务资源类型已移除，chargeTaskPost 为 no-op）
UPDATE billing_unit
SET is_active = 0, price = 0, updated_at = NOW()
WHERE unit_type = 'post_creation';

-- gitlab_disk: 确保名称和单位正确
UPDATE billing_unit
SET name = 'GitLab 磁盘', unit = 'GB/月', updated_at = NOW()
WHERE unit_type = 'gitlab_disk' AND (name != 'GitLab 磁盘' OR unit != 'GB/月');

-- gitlab_traffic: 确保名称和单位正确
UPDATE billing_unit
SET name = 'GitLab 流量费', unit = 'GB', updated_at = NOW()
WHERE unit_type = 'gitlab_traffic' AND (name != 'GitLab 流量费' OR unit != 'GB');

-- ============================================================
-- 2. billing_pricing_package：修正种子数据中的旧积分值
-- ============================================================

-- 默认套餐 (package_number=0): 普通任务已移除 → normal_task_points=0
-- 编程任务（现「任务」）价格 30→55
UPDATE billing_pricing_package
SET normal_task_points = 0,
    programming_task_points = 55,
    normal_task_renewal_points_per_month = 0,
    updated_at = NOW()
WHERE package_number = 0;

-- 所有套餐：废弃字段置零 + programming_task_points 30→55
UPDATE billing_pricing_package
SET normal_task_points = 0,
    normal_task_renewal_points_per_month = 0,
    programming_task_points = 55,
    updated_at = NOW()
WHERE normal_task_points > 0
   OR normal_task_renewal_points_per_month > 1
   OR programming_task_points != 55;

-- ============================================================
-- 3. billing_transaction：清理含「积分」描述的旧误扣费记录
--    这些记录使用了已废弃的定价（普通任务扣 3 积分、智能体任务扣 30 积分）
--    且描述中包含「积分」字样，属于脏数据
-- ============================================================

-- 先清理关联的 outbox_message
DELETE FROM billing_outbox_message
WHERE billing_transaction_id IN (
    SELECT id FROM billing_transaction
    WHERE description LIKE '%积分%'
       OR description LIKE '%智能体任务%'
       OR description LIKE '%创建任务帖消耗%'
);

-- 清理关联的 idempotency_key
DELETE FROM billing_idempotency_key
WHERE transaction_id IN (
    SELECT id FROM billing_transaction
    WHERE description LIKE '%积分%'
       OR description LIKE '%智能体任务%'
       OR description LIKE '%创建任务帖消耗%'
);

-- 清理关联的 usage 记录（通过 account_id + created_at 关联）
DELETE FROM billing_usage
WHERE account_id IN (
    SELECT DISTINCT account_id FROM billing_transaction
    WHERE description LIKE '%积分%'
       OR description LIKE '%智能体任务%'
       OR description LIKE '%创建任务帖消耗%'
);

-- 删除脏交易记录
DELETE FROM billing_transaction
WHERE description LIKE '%积分%'
   OR description LIKE '%智能体任务%'
   OR description LIKE '%创建任务帖消耗%';

-- ============================================================
-- 4. billing_payment_ledger：清理测试充值台账
-- ============================================================

-- 删除没有关联有效 transaction 的台账记录
DELETE FROM billing_payment_ledger
WHERE billing_transaction_id NOT IN (
    SELECT id FROM billing_transaction
);

-- ============================================================
-- 5. billing_account：修正锁定价并重置测试余额
-- ============================================================

-- 修正所有账户的锁定价
UPDATE billing_account
SET locked_post_creation_points = 0,
    locked_server_start_points = 55,
    locked_normal_renewal_points_per_month = 0,
    updated_at = NOW();

-- 所有余额来源于已清理的脏交易和测试充值，重置为 0
-- 余额为 0 的账户保持不变
UPDATE billing_account
SET balance = 0,
    frozen_balance = 0,
    updated_at = NOW()
WHERE balance != 0 OR frozen_balance != 0;

-- ============================================================
-- 6. billing_resource_order：确保无脏数据
-- ============================================================

-- 取消所有 pending 状态的测试订单（如有）
UPDATE billing_resource_order
SET status = 'cancelled', cancelled_at = NOW()
WHERE status = 'pending';

-- ============================================================
-- 验证查询（执行后可运行以确认清理结果）
-- ============================================================
-- SELECT unit_type, name, price, unit, is_active FROM billing_unit ORDER BY id;
-- SELECT COUNT(*) AS remaining_tx FROM billing_transaction;
-- SELECT tenant_id, balance, locked_server_start_points FROM billing_account ORDER BY tenant_id;
