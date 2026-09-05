-- 引荐一级分成：关系边同步 + 应计分录（pending/settled/voided）
CREATE TABLE IF NOT EXISTS billing_referral_edge (
    referred_user_id VARCHAR(64) PRIMARY KEY,
    referrer_user_id VARCHAR(255) NOT NULL,
    referrer_tenant_id BIGINT NOT NULL,
    bound_at VARCHAR(255) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX idx_referral_edge_referrer
    ON billing_referral_edge (referrer_user_id);

CREATE TABLE IF NOT EXISTS billing_referral_commission_accrual (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    referrer_user_id VARCHAR(255) NOT NULL,
    referred_user_id VARCHAR(255) NOT NULL,
    referrer_tenant_id BIGINT NOT NULL,
    source_txn_id VARCHAR(255) NOT NULL UNIQUE,
    source_txn_db_id BIGINT,
    consumption_points BIGINT NOT NULL,
    commission_points BIGINT NOT NULL,
    status VARCHAR(64) NOT NULL,
    consumed_at DATETIME NOT NULL,
    settle_after DATETIME NOT NULL,
    settled_at DATETIME,
    settle_txn_id VARCHAR(255),
    voided_at DATETIME,
    void_reason VARCHAR(512),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE INDEX idx_referral_accrual_referrer_status
    ON billing_referral_commission_accrual (referrer_user_id, status);

CREATE INDEX idx_referral_accrual_settle
    ON billing_referral_commission_accrual (status, settle_after);
