-- 充值类计费单元（交易流水「计费单元」列展示用；非按次扣费价目）
INSERT IGNORE INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
VALUES (1000000000000000004, 'recharge', '积分充值', 1, '积分', 1, NOW(), NOW());

-- 回填历史充值流水（写入时曾未带 billing_unit_id）
UPDATE billing_transaction
SET billing_unit_id = 1000000000000000004
WHERE transaction_type = 'recharge'
  AND (billing_unit_id IS NULL OR billing_unit_id = 0);
