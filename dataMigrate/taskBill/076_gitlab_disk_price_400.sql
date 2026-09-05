-- 076: GitLab 磁盘默认定价 800→400（¥8.00 → ¥4.00 /GB/月）
-- 仅覆盖仍为上一版默认值 800 的行，不覆盖管理员已改价的目录。

UPDATE billing_unit
SET price = 400, updated_at = NOW()
WHERE unit_type = 'gitlab_disk' AND price = 800;
