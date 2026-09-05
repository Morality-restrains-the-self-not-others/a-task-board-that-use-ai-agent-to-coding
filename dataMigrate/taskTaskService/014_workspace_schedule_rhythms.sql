-- taskTaskService: 工作空间级排队调度节奏（OPT：工作空间自动调度）
-- 与任务级 task_top_deliverable_schedule_rhythms 并存：工作空间未配置节奏时，
-- 调度分发回退任务级（legacy）逻辑，生产既有配置不中断。
-- 幂等：CREATE TABLE IF NOT EXISTS + 索引存在性守卫。

CREATE TABLE IF NOT EXISTS workspace_schedule_rhythms (
    workspace_id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    enabled TINYINT NOT NULL DEFAULT 0,
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    updated_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS workspace_schedule_rhythm_windows (
    id VARCHAR(128) PRIMARY KEY,
    workspace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
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
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'workspace_schedule_rhythm_windows' AND INDEX_NAME = 'idx_wsrw_ws') = 0, 'CREATE INDEX idx_wsrw_ws ON workspace_schedule_rhythm_windows(workspace_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
