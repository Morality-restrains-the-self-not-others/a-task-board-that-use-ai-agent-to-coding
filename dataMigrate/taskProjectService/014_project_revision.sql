-- project_revision: append-only snapshots of project name/description (ADR-0051).
-- Time-accumulating: monthly RANGE partitions. PK includes created_at.
-- tenant_id mirrors project_entries.company_id.
-- version_num is allocated in-app (MAX+1).

CREATE TABLE IF NOT EXISTS project_revision (
    id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    version_num INT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    actor_user_id VARCHAR(64) NOT NULL DEFAULT '',
    changed_fields VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id, created_at),
    INDEX idx_pr_project_created (project_id, created_at),
    INDEX idx_pr_tenant_created (tenant_id, created_at)
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
