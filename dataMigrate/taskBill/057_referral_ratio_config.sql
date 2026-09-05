-- 推荐码管理：分账比例 / 推荐比例（平台政策；微信打款另受商户 max_ratio 上限约束）
ALTER TABLE billing_referral_config
  ADD COLUMN profit_sharing_ratio_percent INTEGER NOT NULL DEFAULT 30,
  ADD COLUMN referral_rate_percent INTEGER NOT NULL DEFAULT 5;
