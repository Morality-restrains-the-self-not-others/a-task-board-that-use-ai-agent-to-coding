-- 041: 续存不另扣费 — 存量 server_start_renewal 单价置 0
-- 续存只消耗 task_post_quota，本单位若仍存在则不得带创建帖标价。

UPDATE billing_unit
SET price = 0, name = '任务帖续存（仅扣配额）', updated_at = NOW()
WHERE unit_type = 'server_start_renewal' AND price <> 0;
