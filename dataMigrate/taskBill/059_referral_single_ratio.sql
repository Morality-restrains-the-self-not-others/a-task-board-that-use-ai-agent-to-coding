-- 059: 管理员只设一个分成比例；5%~30% 为范围展示。
-- 将存量双列对齐为推荐比例（政策佣金），避免继续按旧「分账上限 30%」打款。
UPDATE billing_referral_config
SET profit_sharing_ratio_percent = referral_rate_percent
WHERE profit_sharing_ratio_percent <> referral_rate_percent;
