-- 024: VIP 会员等级体系
-- tier: normal（普通会员）| vip1（VIP1 会员）
-- cumulative_consumption_cents: 累计核销消费金额（分），用于自动升级判定
-- 注册用户默认 normal，累计消费满 10000 分（100 元）自动升级 vip1

CREATE TABLE IF NOT EXISTS billing_membership (
  id bigint NOT NULL PRIMARY KEY,
  tenant_id bigint NOT NULL UNIQUE,
  tier varchar(20) NOT NULL DEFAULT 'normal' CHECK (tier IN ('normal', 'vip1')),
  cumulative_consumption_cents bigint NOT NULL DEFAULT 0,
  upgraded_at datetime NULL,
  created_at datetime NOT NULL,
  updated_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX billing_membership_tenant_id ON billing_membership (tenant_id);

-- 为已有计费账户的租户创建默认 normal 会员记录
INSERT IGNORE INTO billing_membership (id, tenant_id, tier, cumulative_consumption_cents, created_at, updated_at)
SELECT
  CAST((ba.tenant_id * 1000000 + 1) AS SIGNED) AS id,
  ba.tenant_id,
  'normal',
  COALESCE((
    SELECT SUM(bt.amount)
    FROM billing_transaction bt
    WHERE bt.account_id = ba.id AND bt.transaction_type = 'consumption'
  ), 0),
  NOW(),
  NOW()
FROM billing_account ba
WHERE ba.tenant_id NOT IN (SELECT tenant_id FROM billing_membership);

-- 对存量累计消费 >= 10000 分的租户，自动升级到 vip1
UPDATE billing_membership
SET tier = 'vip1', upgraded_at = NOW(), updated_at = NOW()
WHERE tier = 'normal'
  AND cumulative_consumption_cents >= 10000;
