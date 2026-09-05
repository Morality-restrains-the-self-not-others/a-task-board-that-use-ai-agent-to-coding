-- taskProjectService: Core schema
-- Tables: 18 tables — projects, workspaces, project_workspace_accesses, progress systems, etc.

CREATE TABLE IF NOT EXISTS project_entries (
    id VARCHAR(64) PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    company_id VARCHAR(64) NOT NULL,
    installed_image_id VARCHAR(64) DEFAULT '',
    container_image_name VARCHAR(255) DEFAULT '',
    tags TEXT,
    server_run_template TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_projects_company ON project_entries(company_id);

CREATE TABLE IF NOT EXISTS project_repos (
    id VARCHAR(64) PRIMARY KEY,
    project_id VARCHAR(64) NOT NULL,
    repo_url TEXT NOT NULL,
    clone_alias VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_project_repos_project ON project_repos(project_id);

CREATE TABLE IF NOT EXISTS project_workspace_entries (
    id VARCHAR(64) PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    company_id VARCHAR(64) NOT NULL,
    deliverable_system_id VARCHAR(64) DEFAULT '',
    deliverable_system_from VARCHAR(64) DEFAULT 'company',
    is_default TINYINT DEFAULT 0,
    task_archive_tier VARCHAR(64) DEFAULT '7d',
    allow_personal_feature_params TINYINT DEFAULT 0,
    container_image_at_mode_enabled TINYINT NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_workspaces_company ON project_workspace_entries(company_id);

CREATE TABLE IF NOT EXISTS project_workspace_accesses (
    id VARCHAR(64) PRIMARY KEY,
    workspace_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) DEFAULT '',
    group_id VARCHAR(64) DEFAULT '',
    permission VARCHAR(64) NOT NULL DEFAULT 'viewer',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_ws_access_ws ON project_workspace_accesses(workspace_id);
CREATE INDEX idx_ws_access_user ON project_workspace_accesses(user_id);

CREATE TABLE IF NOT EXISTS project_workspaces (
    project_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (project_id, workspace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS project_deliverable_systems (
    id VARCHAR(64) PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    company_id VARCHAR(64) DEFAULT '',
    is_system TINYINT DEFAULT 0,
    is_default TINYINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS project_deliverable_columns (
    id VARCHAR(64) PRIMARY KEY,
    system_id VARCHAR(64) NOT NULL,
    name TEXT NOT NULL,
    order_num INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_dc_system ON project_deliverable_columns(system_id);

CREATE TABLE IF NOT EXISTS project_user_workspace_work_panel_filters (
    user_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    payload_json TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, company_id, workspace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_uw_wpf_ws ON project_user_workspace_work_panel_filters(company_id, workspace_id);

CREATE TABLE IF NOT EXISTS project_tags (
    tag VARCHAR(255) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (tag, project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_project_tags_tag ON project_tags(tag);

CREATE TABLE IF NOT EXISTS project_workspace_task_kind_options (
    workspace_id VARCHAR(64) PRIMARY KEY,
    options_json TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS project_workspace_code_lang_options (
    workspace_id VARCHAR(64) PRIMARY KEY,
    options_json TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS project_workspace_create_task_field_settings (
    workspace_id VARCHAR(64) PRIMARY KEY,
    fields_json TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS project_progress_systems (
    id VARCHAR(64) PRIMARY KEY,
    name TEXT NOT NULL,
    is_default TINYINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS project_progress_columns (
    id VARCHAR(64) PRIMARY KEY,
    system_id VARCHAR(64) NOT NULL,
    name TEXT NOT NULL,
    order_num INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_pc_system ON project_progress_columns(system_id);

CREATE TABLE IF NOT EXISTS project_progress_systems_tenant (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_pst_tid ON project_progress_systems_tenant(tenant_id);

CREATE TABLE IF NOT EXISTS project_progress_columns_tenant (
    id VARCHAR(64) PRIMARY KEY,
    system_id VARCHAR(64) NOT NULL,
    name TEXT NOT NULL,
    order_num INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_pct_sys ON project_progress_columns_tenant(system_id);

CREATE TABLE IF NOT EXISTS project_progress_systems_default_tenant (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL UNIQUE,
    target_type VARCHAR(64) NOT NULL DEFAULT 'tenant',
    target_id VARCHAR(64) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_psdt_tid ON project_progress_systems_default_tenant(tenant_id);

CREATE TABLE IF NOT EXISTS project_progress_systems_workspace (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    target_type VARCHAR(64) NOT NULL DEFAULT 'tenant',
    target_id VARCHAR(64) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_psw_twid ON project_progress_systems_workspace(tenant_id, workspace_id);
