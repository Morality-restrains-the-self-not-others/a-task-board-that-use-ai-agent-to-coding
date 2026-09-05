-- 006: cloud_server_config_defaults — 云服务器默认配置（从 Django CloudServerConfigDefault 迁移）
CREATE TABLE IF NOT EXISTS cloud_server_config_defaults (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    authorization_id VARCHAR(64) NOT NULL,
    platform_type VARCHAR(64) NOT NULL DEFAULT 'aliyun',
    region VARCHAR(64) DEFAULT '',
    vpc_id VARCHAR(255) DEFAULT '',
    vswitch_id VARCHAR(255) DEFAULT '',
    security_group_id VARCHAR(255) DEFAULT '',
    payment_type VARCHAR(64) DEFAULT '',
    bandwidth_charging_mode VARCHAR(64) DEFAULT '',
    bandwidth INT DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_cscd_company_auth (company_id, authorization_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
