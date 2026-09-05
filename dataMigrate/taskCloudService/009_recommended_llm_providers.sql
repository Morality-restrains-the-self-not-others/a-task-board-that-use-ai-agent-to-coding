-- OPT-20260730-001: recommended-llm-providers table
-- Incremental migration — added after 001_schema.sql for existing deployments
-- where data_migrate_log already has 001_schema.sql as applied.

CREATE TABLE IF NOT EXISTS cloud_recommended_llm_providers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    docs_url TEXT NOT NULL,
    description TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_rlp_sort ON cloud_recommended_llm_providers(sort_order, name);
