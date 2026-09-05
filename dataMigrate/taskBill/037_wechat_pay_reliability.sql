-- ═══════════════════════════════════════════════════════════════
-- taskBill 微信支付回调可靠性 (v64 wechat-login-openid-unionid §4)
-- 1) billing_payment_pending 持久化（替代内存 map，进程重启不丢回调）
-- 2) billing_resource_order 增加 user_id（下单会话）与支付凭据记录列
--    （pay_method/pay_app_key/pay_openid/pay_unionid 仅审计记录，登录/支付解耦）
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS billing_payment_pending (
    out_trade_no VARCHAR(64) PRIMARY KEY,
    user_id      BIGINT       NOT NULL DEFAULT 0,
    tenant_id    BIGINT       NOT NULL DEFAULT 0,
    order_id     BIGINT       NOT NULL DEFAULT 0,
    amount_fen   BIGINT       NOT NULL DEFAULT 0,
    status       VARCHAR(16)  NOT NULL DEFAULT 'pending',  -- pending/paid
    created_at   DATETIME     NOT NULL,
    paid_at      DATETIME     DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE billing_resource_order
    ADD COLUMN user_id      BIGINT       NOT NULL DEFAULT 0,
    ADD COLUMN pay_method   VARCHAR(32)  NOT NULL DEFAULT '',
    ADD COLUMN pay_app_key  VARCHAR(64)  NOT NULL DEFAULT '',
    ADD COLUMN pay_openid   VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN pay_unionid  VARCHAR(128) NOT NULL DEFAULT '';
