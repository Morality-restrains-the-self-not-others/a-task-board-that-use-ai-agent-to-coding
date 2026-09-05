-- 054: 微信分账接收方登记状态（推荐资格 ≠ 商户平台接收方）
-- 资格在 taskReferral.referral_code；商户平台「管理分账接收方」只在
-- POST /v3/profitsharing/receivers/add 成功后出现。本表按推荐人幂等记录登记结果。

CREATE TABLE IF NOT EXISTS billing_profit_sharing_receiver (
    referrer_user_id VARCHAR(64) NOT NULL,
    appid VARCHAR(64) NOT NULL DEFAULT '',
    openid VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL,
    fail_reason VARCHAR(512) NOT NULL DEFAULT '',
    registered_at DATETIME NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (referrer_user_id),
    KEY idx_ps_receiver_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
