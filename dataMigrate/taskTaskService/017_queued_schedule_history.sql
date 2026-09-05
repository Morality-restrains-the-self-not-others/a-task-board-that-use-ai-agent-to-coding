-- taskTaskService: 工作空间排队调度历史（时间累积型）
-- 主键含分区键 created_at；按月 RANGE 分区。热窗查询走 (workspace_id, created_at)。
-- 幂等：CREATE TABLE IF NOT EXISTS + 列存在性守卫。

CREATE TABLE IF NOT EXISTS task_queued_schedule_history (
    id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    task_id VARCHAR(64) NOT NULL DEFAULT '',
    message VARCHAR(512) NOT NULL DEFAULT '',
    actor_user_id VARCHAR(64) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id, created_at),
    INDEX idx_tqsh_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
    PARTITION p202612 VALUES LESS THAN (TO_DAYS('2027-01-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'workspace_schedule_rhythms' AND COLUMN_NAME = 'last_in_window') = 0,
  'ALTER TABLE workspace_schedule_rhythms ADD COLUMN last_in_window TINYINT NULL',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
