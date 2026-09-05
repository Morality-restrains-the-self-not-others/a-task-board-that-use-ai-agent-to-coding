-- ADR-0023 同款：cloud_job_execution_event 按 workspace_id 哈希分表
-- 表名 cloud_job_execution_event_{00..15}，选片 CRC32(workspace_id) % 16（与 Go hash/crc32.ChecksumIEEE 一致）
-- PK Snowflake VARCHAR，禁止 AUTO_INCREMENT
-- 遗留表 cloud_job_execution_event 仅作回填源，应用停止写入

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_00 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_01 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_02 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_03 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_04 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_05 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_06 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_07 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_08 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_09 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_10 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_11 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_12 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_13 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_14 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_job_execution_event_15 (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
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
  PRIMARY KEY (id),
  KEY idx_jee_task_seq (task_id, job_id, seq),
  KEY idx_jee_task_steps (task_id, comment_id, job_id, phase, step_number),
  KEY idx_jee_ws_created (workspace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO cloud_job_execution_event_00 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 0;

INSERT IGNORE INTO cloud_job_execution_event_01 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 1;

INSERT IGNORE INTO cloud_job_execution_event_02 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 2;

INSERT IGNORE INTO cloud_job_execution_event_03 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 3;

INSERT IGNORE INTO cloud_job_execution_event_04 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 4;

INSERT IGNORE INTO cloud_job_execution_event_05 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 5;

INSERT IGNORE INTO cloud_job_execution_event_06 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 6;

INSERT IGNORE INTO cloud_job_execution_event_07 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 7;

INSERT IGNORE INTO cloud_job_execution_event_08 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 8;

INSERT IGNORE INTO cloud_job_execution_event_09 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 9;

INSERT IGNORE INTO cloud_job_execution_event_10 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 10;

INSERT IGNORE INTO cloud_job_execution_event_11 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 11;

INSERT IGNORE INTO cloud_job_execution_event_12 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 12;

INSERT IGNORE INTO cloud_job_execution_event_13 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 13;

INSERT IGNORE INTO cloud_job_execution_event_14 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 14;

INSERT IGNORE INTO cloud_job_execution_event_15 (id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at)
SELECT id, company_id, workspace_id, task_id, comment_id, job_id, seq, phase, message, step_number, delivery_summary, step_state, job_status, layer_id, event_json, created_at FROM cloud_job_execution_event
WHERE CRC32(IFNULL(NULLIF(workspace_id, ''), '')) % 16 = 15;
