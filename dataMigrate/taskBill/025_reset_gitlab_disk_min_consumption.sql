-- 025: 移除 GitLab 磁盘最低累计消费门槛（已由 VIP 会员等级体系接管）
-- 将 billing_unit 中 gitlab_disk 的 min_consumption_cents 重置为 0

UPDATE billing_unit SET min_consumption_cents = 0, updated_at = NOW()
WHERE unit_type = 'gitlab_disk' AND min_consumption_cents > 0;
