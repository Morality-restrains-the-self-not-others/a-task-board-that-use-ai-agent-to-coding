-- OPT-20260730-001: sub-token-providers table
-- Incremental migration — added after 001_schema.sql for existing deployments
-- where data_migrate_log already has 001_schema.sql as applied.

CREATE TABLE IF NOT EXISTS cloud_sub_token_providers (
    id VARCHAR(64) PRIMARY KEY,
    provider_name VARCHAR(255) NOT NULL,
    base_url VARCHAR(1024) NOT NULL,
    derive_endpoint VARCHAR(512) NOT NULL DEFAULT '/api/token/derive',
    derive_method VARCHAR(16) NOT NULL DEFAULT 'POST',
    derive_params TEXT NOT NULL,
    response_token_field VARCHAR(255) NOT NULL DEFAULT 'token',
    expires_in_field VARCHAR(255) NOT NULL DEFAULT 'expires_in',
    description TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_sub_token_provider_name ON cloud_sub_token_providers(provider_name);
