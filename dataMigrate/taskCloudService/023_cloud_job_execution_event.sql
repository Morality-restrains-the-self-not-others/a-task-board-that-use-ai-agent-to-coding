-- 023: cloud_job_execution_event — 容器执行步骤/作业生命周期落库（Kafka 消费者写入）
-- 伸缩：时间累积型；热窗口按月 RANGE 分区；主键含分区键 created_at
-- 幂等：应用层 (task_id, job_id, seq)；分区表 UNIQUE 必须含分区键故不建跨分区 UNIQUE

CREATE TABLE IF NOT EXISTS cloud_job_execution_event (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL DEFAULT '',
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL DEFAULT '',
  job_id VARCHAR(128) NOT NULL,
  seq INT NOT NULL,
  phase VARCHAR(32) NOT NULL,
  message MEDIUMTEXT,
  step_number INT NOT NULL DEFAULT 0,
  delivery_summary TEXT,
  step_state VARCHAR(32) NOT NULL DEFAULT '',
  job_status VARCHAR(32) NOT NULL DEFAULT '',
  layer_id VARCHAR(128) NOT NULL DEFAULT '',
  event_json JSON,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id, created_at),
  KEY idx_job_seq (task_id, job_id, seq, created_at),
  KEY idx_job_steps (task_id, comment_id, job_id, phase, step_number, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (TO_DAYS(created_at)) (
  PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
  PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
  PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
  PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
  PARTITION p202612 VALUES LESS THAN (TO_DAYS('2027-01-01')),
  PARTITION p_future VALUES LESS THAN MAXVALUE
);
