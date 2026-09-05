-- taskBudget (via taskCloudService): Budget ledger schema
-- Database: task_budget
-- Tables: 5 tables — budget defaults, task budgets, usage tracking, idempotency, permissions

CREATE TABLE IF NOT EXISTS cloud_workspace_model_budget_default (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    base_url VARCHAR(255) NOT NULL,
    model_name VARCHAR(128) NOT NULL,
    input_price_per_1m VARCHAR(64) NOT NULL DEFAULT '0',
    output_price_per_1m VARCHAR(64) NOT NULL DEFAULT '0',
    budget_limit VARCHAR(64) NOT NULL DEFAULT '0',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_workspace_model(workspace_id, provider, model_name, base_url)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cloud_task_model_budget (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    todo_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    base_url VARCHAR(255) NOT NULL,
    model_name VARCHAR(128) NOT NULL,
    budget_limit VARCHAR(64) NULL,
    budget_limit_source VARCHAR(32) NOT NULL DEFAULT 'inherited',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_todo_model(todo_id, provider, model_name, base_url)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cloud_task_model_budget_usage (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    todo_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    base_url VARCHAR(255) NOT NULL,
    model_name VARCHAR(128) NOT NULL,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    spent_amount VARCHAR(64) NOT NULL DEFAULT '0',
    last_reported_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_usage_model(todo_id, provider, model_name, base_url)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cloud_task_model_budget_usage_idempotency (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    idempotency_key VARCHAR(191) NOT NULL UNIQUE,
    usage_id VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cloud_tenant_budget_permission (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    subject_type VARCHAR(32) NOT NULL,
    subject_id VARCHAR(64) NOT NULL,
    can_raise_task_budget TINYINT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_tenant_perm(company_id, subject_type, subject_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
