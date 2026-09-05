-- taskCloudService: Core schema (main database)
-- Tables from runMigrations() + ensureFeatureParamsSchema()
-- Total: 16 tables

CREATE TABLE IF NOT EXISTS cloud_platform_authorizations (
    id VARCHAR(64) PRIMARY KEY, platform_type VARCHAR(64) NOT NULL DEFAULT 'aliyun',
    authorization_type VARCHAR(64) DEFAULT 'access_key',
    secret_id VARCHAR(255) NOT NULL, secret_key VARCHAR(255) NOT NULL,
    remark VARCHAR(255) DEFAULT '', company_id VARCHAR(64) NOT NULL,
    active TINYINT DEFAULT 1, oauth_token_id VARCHAR(64) DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_cpa_company ON cloud_platform_authorizations(company_id);

CREATE TABLE IF NOT EXISTS cloud_platform_authorization_active_methods (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    platform_type VARCHAR(64) NOT NULL DEFAULT 'aliyun',
    auth_method VARCHAR(64) NOT NULL DEFAULT 'access_key',
    auth_instance_id VARCHAR(64) NOT NULL,
    is_active TINYINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_cpam_company ON cloud_platform_authorization_active_methods(company_id);
CREATE UNIQUE INDEX idx_cpam_unique ON cloud_platform_authorization_active_methods(company_id,platform_type,auth_method,auth_instance_id);

CREATE TABLE IF NOT EXISTS cloud_oauth_tokens (
    id VARCHAR(64) PRIMARY KEY, authorization_id VARCHAR(64) NOT NULL,
    access_token TEXT NOT NULL, refresh_token TEXT,
    company_id VARCHAR(64) DEFAULT '', platform_type VARCHAR(64) DEFAULT 'aliyun',
    authorization_type VARCHAR(64) DEFAULT 'oauth', scope TEXT,
    expires_at DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_oat_auth ON cloud_oauth_tokens(authorization_id);

CREATE TABLE IF NOT EXISTS cloud_server_configs (
    id VARCHAR(64) PRIMARY KEY, company_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL, task_id VARCHAR(64) NOT NULL,
    comment_id VARCHAR(64) NOT NULL DEFAULT '',
    platform VARCHAR(64) NOT NULL DEFAULT 'aliyun', instance_id VARCHAR(255) DEFAULT '',
    security_group_id VARCHAR(255) DEFAULT '', vswitch_id VARCHAR(255) DEFAULT '',
    region VARCHAR(64) DEFAULT '', zone_id VARCHAR(64) DEFAULT '',
    authorization_id VARCHAR(64) DEFAULT '', public_ip VARCHAR(64) DEFAULT '',
    server_url VARCHAR(512) DEFAULT '', business_api_endpoint VARCHAR(512) DEFAULT '',
    error_reason TEXT,
    client_token VARCHAR(255) DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    container_vscode_url VARCHAR(512) DEFAULT '',
    idle_since DATETIME DEFAULT NULL,
    launch_request_id VARCHAR(255) DEFAULT '',
    last_runtime_status VARCHAR(64) DEFAULT '',
    terminal_released TINYINT NOT NULL DEFAULT 0,
    started_via VARCHAR(64) DEFAULT '',
    image_invoker_user_id VARCHAR(64) DEFAULT '',
    last_heartbeat_at DATETIME,
    verification_secret VARCHAR(255) DEFAULT '',
    userdata_run_verified DATETIME DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_csc_workspace ON cloud_server_configs(workspace_id);
CREATE INDEX idx_csc_task ON cloud_server_configs(task_id);

CREATE TABLE IF NOT EXISTS cloud_server_config_histories (
    id VARCHAR(64) PRIMARY KEY, company_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL, task_id VARCHAR(64) NOT NULL,
    platform VARCHAR(64) NOT NULL DEFAULT 'aliyun', platform_id INTEGER DEFAULT 1,
    instance_id VARCHAR(255) DEFAULT '', instance_type_id VARCHAR(64) DEFAULT '',
    security_group_id VARCHAR(255) DEFAULT '', vswitch_id VARCHAR(255) DEFAULT '',
    region VARCHAR(64) DEFAULT '', zone_id VARCHAR(64) DEFAULT '',
    authorization_id VARCHAR(64) DEFAULT '', public_ip VARCHAR(64) DEFAULT '',
    server_url VARCHAR(512) DEFAULT '', business_api_endpoint VARCHAR(512) DEFAULT '',
    error_reason TEXT, stop_reason VARCHAR(255) DEFAULT '',
    runtime_source VARCHAR(64) DEFAULT '', launch_request_id VARCHAR(255) DEFAULT '',
    cpu_cores INTEGER DEFAULT 1, memory_gb INTEGER DEFAULT 1, storage_gb INTEGER DEFAULT 40,
    start_reason VARCHAR(255) DEFAULT '',
    started_at DATETIME, stopped_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_cs_hist_task ON cloud_server_config_histories(task_id);

CREATE TABLE IF NOT EXISTS cloud_tenant_installed_images (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    external_image_id VARCHAR(255) NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    version VARCHAR(64) DEFAULT '',
    image_url TEXT NOT NULL,
    target_architectures TEXT,
    size INTEGER,
    is_dev_mode TINYINT DEFAULT 0,
    vendor_id VARCHAR(64) DEFAULT '',
    vendor_name VARCHAR(255) DEFAULT '',
    installed_by_id VARCHAR(64) DEFAULT '',
    installed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NULL DEFAULT NULL COMMENT '镜像更新时间（目录 updated_at 安装时快照；无则回填安装时间）',
    userdata_template_id VARCHAR(64) DEFAULT '',
    auto_run_steps_md TEXT,
    auto_run_steps_extract_status VARCHAR(64) DEFAULT '',
    auto_run_steps_digest VARCHAR(255) DEFAULT '',
    UNIQUE(tenant_id, external_image_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_tii_tenant ON cloud_tenant_installed_images(tenant_id);

CREATE TABLE IF NOT EXISTS cloud_vendor_platform_credentials (
    id VARCHAR(64) PRIMARY KEY,
    vendor_id VARCHAR(64) NOT NULL,
    platform_type VARCHAR(64) NOT NULL DEFAULT 'aliyun',
    secret_id VARCHAR(255) NOT NULL,
    secret_key VARCHAR(255) NOT NULL,
    remark VARCHAR(255) DEFAULT '',
    last_verified_at DATETIME,
    last_verify_error TEXT,
    is_active TINYINT DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(vendor_id, platform_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_vcc_vendor ON cloud_vendor_platform_credentials(vendor_id);

CREATE TABLE IF NOT EXISTS cloud_server_events (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    task_id VARCHAR(64) NOT NULL,
    comment_id VARCHAR(64) NOT NULL DEFAULT '',
    company_member_id VARCHAR(64) NOT NULL DEFAULT '',
    event_type VARCHAR(64) NOT NULL,
    event_data TEXT NOT NULL,
    status VARCHAR(64) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_cse_task ON cloud_server_events(task_id);
CREATE INDEX idx_cse_company ON cloud_server_events(company_id);
CREATE INDEX idx_cse_company_task_type ON cloud_server_events(company_id, task_id, event_type);

CREATE TABLE IF NOT EXISTS cloud_access_key_iam_associations (
    id VARCHAR(64) PRIMARY KEY,
    cloud_platform_auth_id VARCHAR(64) NOT NULL,
    access_key VARCHAR(255) NOT NULL,
    iam_id VARCHAR(255) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_akia_auth ON cloud_access_key_iam_associations(cloud_platform_auth_id);

CREATE TABLE IF NOT EXISTS cloud_comment_container_bindings (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL DEFAULT '',
    task_id VARCHAR(64) NOT NULL,
    comment_id VARCHAR(64) NOT NULL,
    execution_mode VARCHAR(64) NOT NULL DEFAULT 'wait_previous',
    depends_on_comment_id VARCHAR(64) DEFAULT '',
    status VARCHAR(64) NOT NULL DEFAULT 'pending',
    mock_container_name VARCHAR(255) DEFAULT '',
    csc_id VARCHAR(64) DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, task_id, comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_ccb_task ON cloud_comment_container_bindings(company_id, task_id);

CREATE TABLE IF NOT EXISTS cloud_workspace_machine_policies (
    company_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    idle_recycle_minutes INTEGER NOT NULL DEFAULT 5,
    prefer_idle_reuse TINYINT NOT NULL DEFAULT 0,
    enabled_authorization_ids TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (company_id, workspace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Feature params tables (from ensureFeatureParamsSchema)

CREATE TABLE IF NOT EXISTS cloud_tenant_feature_params (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL UNIQUE,
    providers TEXT NOT NULL,
    agent_model VARCHAR(255) NOT NULL DEFAULT '',
    agent_model_provider VARCHAR(255) NOT NULL DEFAULT '',
    agent_max_steps INTEGER NOT NULL DEFAULT 200,
    summary_model VARCHAR(255) NOT NULL DEFAULT '',
    summary_model_provider VARCHAR(255) NOT NULL DEFAULT '',
    llm_budget_enabled TINYINT NOT NULL DEFAULT 0,
    extra_env_vars TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cloud_workspace_feature_params (
    id VARCHAR(64) PRIMARY KEY,
    workspace_id VARCHAR(64) NOT NULL UNIQUE,
    company_id VARCHAR(64) NOT NULL DEFAULT '',
    use_company_default TINYINT NOT NULL DEFAULT 1,
    providers TEXT NOT NULL,
    agent_model VARCHAR(255) NOT NULL DEFAULT '',
    agent_model_provider VARCHAR(255) NOT NULL DEFAULT '',
    agent_max_steps INTEGER NOT NULL DEFAULT 200,
    summary_model VARCHAR(255) NOT NULL DEFAULT '',
    summary_model_provider VARCHAR(255) NOT NULL DEFAULT '',
    llm_budget_enabled TINYINT NOT NULL DEFAULT 0,
    extra_env_vars TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cloud_personal_feature_params_config (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL DEFAULT '',
    name VARCHAR(255) NOT NULL DEFAULT '',
    providers TEXT NOT NULL,
    agent_model VARCHAR(255) NOT NULL DEFAULT '',
    agent_model_provider VARCHAR(255) NOT NULL DEFAULT '',
    agent_max_steps INTEGER NOT NULL DEFAULT 200,
    summary_model VARCHAR(255) NOT NULL DEFAULT '',
    summary_model_provider VARCHAR(255) NOT NULL DEFAULT '',
    llm_budget_enabled TINYINT NOT NULL DEFAULT 0,
    extra_env_vars TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, company_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cloud_feature_params_access_audit (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL DEFAULT '',
    company_id VARCHAR(64) NOT NULL DEFAULT '',
    workspace_id VARCHAR(64) NOT NULL DEFAULT '',
    resource VARCHAR(64) NOT NULL DEFAULT 'company',
    access_context VARCHAR(255) NOT NULL DEFAULT '',
    view_mode VARCHAR(64) NOT NULL DEFAULT 'denied',
    auth_method VARCHAR(64) NOT NULL DEFAULT 'unknown',
    http_method VARCHAR(16) NOT NULL DEFAULT '',
    path VARCHAR(512) NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    client_ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent VARCHAR(512) NOT NULL DEFAULT '',
    referer VARCHAR(512) NOT NULL DEFAULT '',
    trace_id VARCHAR(64) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_fp_audit_user ON cloud_feature_params_access_audit(user_id);
CREATE INDEX idx_fp_audit_company ON cloud_feature_params_access_audit(company_id);
CREATE INDEX idx_fp_audit_created ON cloud_feature_params_access_audit(created_at);

CREATE TABLE IF NOT EXISTS cloud_task_feature_params_snapshot (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    source VARCHAR(64) NOT NULL DEFAULT 'company',
    source_config_id VARCHAR(64) NOT NULL DEFAULT '',
    source_display_name VARCHAR(255) NOT NULL DEFAULT '',
    resolved_env TEXT NOT NULL,
    providers_summary TEXT NOT NULL,
    agent_model VARCHAR(255) NOT NULL DEFAULT '',
    agent_model_provider VARCHAR(255) NOT NULL DEFAULT '',
    agent_max_steps INTEGER NOT NULL DEFAULT 200,
    summary_model VARCHAR(255) NOT NULL DEFAULT '',
    summary_model_provider VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_fp_snapshot_task ON cloud_task_feature_params_snapshot(task_id, created_at);

CREATE TABLE IF NOT EXISTS cloud_task_api_key_usage (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    provider_name VARCHAR(255) NOT NULL DEFAULT '',
    api_key_hash VARCHAR(255) NOT NULL DEFAULT '',
    key_type VARCHAR(64) NOT NULL DEFAULT 'master',
    used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_fp_api_key_task ON cloud_task_api_key_usage(task_id, provider_name);
