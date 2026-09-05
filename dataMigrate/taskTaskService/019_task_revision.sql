-- task_revision: append-only snapshots of task title/description (ADR-0051).
-- Time-accumulating: monthly RANGE partitions. PK includes created_at.
-- version_num is allocated in-app (MAX+1); UNIQUE(task_id, version_num) cannot
-- be declared because partition key must be in every unique key.

CREATE TABLE IF NOT EXISTS task_revision (
    id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    task_id VARCHAR(64) NOT NULL,
    version_num INT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    actor_user_id VARCHAR(64) NOT NULL DEFAULT '',
    changed_fields VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id, created_at),
    INDEX idx_tr_task_created (task_id, created_at),
    INDEX idx_tr_tenant_created (tenant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
    PARTITION p202612 VALUES LESS THAN (TO_DAYS('2027-01-01')),
    PARTITION p202701 VALUES LESS THAN (TO_DAYS('2027-02-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
