-- 058: 回填已支付/已退款订单的分账快照（管理员订单展开用）
-- 021 写入路径曾用 referrer_tenant_id = order.tenant_id，且空 openid 直接跳过，导致 0 行。
-- 幂等：已有 billing_profit_sharing.order_id 的订单不插入。
-- 当前无资格的边仍落 failed + qualification_revoked，避免 timer 向微信打款。

INSERT INTO billing_profit_sharing (
  out_profit_sharing_no, order_id, order_number, tenant_id,
  referrer_user_id, referrer_openid,
  total_yuan_cents, commission_yuan_cents, status, fail_reason,
  settle_after, created_at, updated_at
)
SELECT
  CONCAT('PS-BF-', o.id),
  o.id,
  o.order_number,
  o.tenant_id,
  re.referrer_user_id,
  COALESCE(re.referrer_openid, ''),
  o.total_yuan_cents,
  COALESCE((
    SELECT a.commission_points
    FROM billing_referral_commission_accrual a
    WHERE a.referred_user_id = (CAST(o.user_id AS CHAR) COLLATE utf8mb4_unicode_ci)
      AND a.source_txn_id LIKE CONCAT('%', CAST(o.id AS CHAR) COLLATE utf8mb4_unicode_ci, '%')
    ORDER BY a.id DESC
    LIMIT 1
  ), FLOOR((o.total_yuan_cents * 5 + 50) / 100)),
  IF(re.commission_eligible = 1, 'pending', 'failed'),
  IF(re.commission_eligible = 1, '', 'qualification_revoked'),
  DATE_FORMAT(UTC_TIMESTAMP() + INTERVAL 8 DAY, '%Y-%m-%dT%H:%i:%sZ'),
  DATE_FORMAT(UTC_TIMESTAMP(), '%Y-%m-%dT%H:%i:%sZ'),
  DATE_FORMAT(UTC_TIMESTAMP(), '%Y-%m-%dT%H:%i:%sZ')
FROM billing_resource_order o
INNER JOIN billing_referral_edge re
  ON re.referred_user_id = (CAST(o.user_id AS CHAR) COLLATE utf8mb4_unicode_ci)
WHERE o.status IN ('paid', 'refunded')
  AND o.user_id > 0
  AND o.total_yuan_cents > 0
  AND NOT EXISTS (
    SELECT 1 FROM billing_profit_sharing ps WHERE ps.order_id = o.id
  );
