-- 计费单元展示名：server_start 由「任务服务器启动服务费」改为「智能体任务」
UPDATE billing_unit
SET name = '智能体任务', updated_at = NOW()
WHERE unit_type = 'server_start';
