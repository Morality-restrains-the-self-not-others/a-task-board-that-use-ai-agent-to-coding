-- taskTaskService: Core schema
-- Tables: 13 tables — tasks, task_assignees, comments, task_git_identities, etc.

CREATE TABLE IF NOT EXISTS task_tasks (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    description TEXT,
    completed TINYINT DEFAULT 0,
    priority VARCHAR(64) DEFAULT '0',
    order_num INTEGER DEFAULT 0,
    workspace_id VARCHAR(64) NOT NULL,
    owner_id VARCHAR(64) DEFAULT '',
    operator_id VARCHAR(64) DEFAULT '',
    deliverable_obj_id VARCHAR(64) DEFAULT '',
    progress_column_id VARCHAR(64) DEFAULT '',
    parent_task_id VARCHAR(64) DEFAULT '',
    fork_from_id VARCHAR(64) DEFAULT '',
    installed_image_id VARCHAR(64) DEFAULT '',
    auto_run TINYINT DEFAULT 0,
    auto_commit_after_agent_complete TINYINT DEFAULT 0,
    feature_params_source VARCHAR(64) DEFAULT 'company',
    personal_feature_params_config_id VARCHAR(64) DEFAULT '',
    task_kind VARCHAR(64) DEFAULT '',
    code_lang VARCHAR(64) DEFAULT '',
    due_date VARCHAR(64) DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- Idempotent index creation: only create if the index does not already exist.
-- MySQL 8.0 does not support CREATE INDEX IF NOT EXISTS.
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND INDEX_NAME = 'idx_tasks_ws') = 0, 'CREATE INDEX idx_tasks_ws ON task_tasks(workspace_id)', 'SELECT 1 \'skip: idx_tasks_ws exists\'');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND INDEX_NAME = 'idx_tasks_tenant') = 0, 'CREATE INDEX idx_tasks_tenant ON task_tasks(tenant_id)', 'SELECT 1 \'skip: idx_tasks_tenant exists\'');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS task_assignees (
    task_id VARCHAR(64) NOT NULL,
    company_member_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (task_id, company_member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_projects (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    base_branch VARCHAR(255) DEFAULT '',
    target_branch VARCHAR(255) DEFAULT '',
    repo_branches TEXT,
    repo_address VARCHAR(512) DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_projects' AND INDEX_NAME = 'idx_task_projects_task') = 0, 'CREATE INDEX idx_task_projects_task ON task_projects(task_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS task_branch_strategies (
    task_id VARCHAR(64) PRIMARY KEY,
    work_branch_name VARCHAR(255) DEFAULT '',
    merge_target_branch_name VARCHAR(255) DEFAULT '',
    target_branch_name VARCHAR(255) DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_repo_identities (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    repo_url TEXT NOT NULL,
    git_identity_id VARCHAR(64) DEFAULT '',
    git_name VARCHAR(255) DEFAULT '',
    git_email VARCHAR(255) DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(task_id, repo_url(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_comments (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    created_by_id VARCHAR(64) NOT NULL,
    content TEXT NOT NULL,
    mentions_json TEXT,
    execution_mode VARCHAR(64) NOT NULL DEFAULT 'wait_previous',
    depends_on_comment_ids TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments' AND INDEX_NAME = 'idx_comments_task') = 0, 'CREATE INDEX idx_comments_task ON task_comments(task_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS task_git_identities (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) DEFAULT '',
    company_id VARCHAR(64) DEFAULT '',
    label VARCHAR(255) DEFAULT '',
    git_user_name VARCHAR(255) DEFAULT '',
    git_user_email VARCHAR(255) DEFAULT '',
    git_remote_username VARCHAR(255) DEFAULT '',
    is_default TINYINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_feature_params (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL UNIQUE,
    params TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_feature_params_snapshots (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    params TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_feature_params_snapshots' AND INDEX_NAME = 'idx_tfp_snap_task') = 0, 'CREATE INDEX idx_tfp_snap_task ON task_feature_params_snapshots(task_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS task_top_deliverable_schedule_rhythms (
    task_id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    enabled TINYINT NOT NULL DEFAULT 0,
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    updated_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_queued_auto_run_memberships (
    task_id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    top_task_id VARCHAR(64) NOT NULL,
    depth INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(64) NOT NULL DEFAULT 'queued',
    enqueued_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_queued_auto_run_memberships' AND INDEX_NAME = 'idx_qarm_top') = 0, 'CREATE INDEX idx_qarm_top ON task_queued_auto_run_memberships(top_task_id, depth, enqueued_at)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS task_queued_machine_slots (
    task_id VARCHAR(64) PRIMARY KEY,
    top_task_id VARCHAR(64) NOT NULL,
    acquired_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_queued_machine_slots' AND INDEX_NAME = 'idx_qms_top') = 0, 'CREATE INDEX idx_qms_top ON task_queued_machine_slots(top_task_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS task_top_deliverable_schedule_rhythm_windows (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    daily_start VARCHAR(16) NOT NULL DEFAULT '',
    daily_end VARCHAR(16) NOT NULL DEFAULT '',
    max_queued_machines INTEGER NOT NULL DEFAULT 0,
    auto_close TINYINT NOT NULL DEFAULT 0,
    auto_close_warn_minutes INTEGER NOT NULL DEFAULT 5,
    auto_close_warn_key VARCHAR(255) NOT NULL DEFAULT '',
    auto_close_release_key VARCHAR(255) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_top_deliverable_schedule_rhythm_windows' AND INDEX_NAME = 'idx_srw_task') = 0, 'CREATE INDEX idx_srw_task ON task_top_deliverable_schedule_rhythm_windows(task_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
