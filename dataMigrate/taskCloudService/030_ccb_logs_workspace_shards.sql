-- ADR-0023: 评论容器启动日志按 workspace_id 哈希分表
-- 表名 cloud_comment_container_binding_logs_{00..15}
-- 选片 CRC32(workspace_id) % 16（与 Go hash/crc32.ChecksumIEEE 一致）
-- PK Snowflake VARCHAR，禁止 AUTO_INCREMENT
-- 遗留表 cloud_comment_container_binding_logs 仅作回填源，应用停止写入

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_00 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_01 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_02 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_03 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_04 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_05 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_06 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_07 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_08 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_09 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_10 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_11 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_12 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_13 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_14 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs_15 (
  id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_ws_task (workspace_id, company_id, task_id, created_at),
  KEY idx_ccbl_ws_comment (workspace_id, company_id, task_id, comment_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO cloud_comment_container_binding_logs_00 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 0;
INSERT IGNORE INTO cloud_comment_container_binding_logs_01 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 1;
INSERT IGNORE INTO cloud_comment_container_binding_logs_02 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 2;
INSERT IGNORE INTO cloud_comment_container_binding_logs_03 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 3;
INSERT IGNORE INTO cloud_comment_container_binding_logs_04 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 4;
INSERT IGNORE INTO cloud_comment_container_binding_logs_05 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 5;
INSERT IGNORE INTO cloud_comment_container_binding_logs_06 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 6;
INSERT IGNORE INTO cloud_comment_container_binding_logs_07 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 7;
INSERT IGNORE INTO cloud_comment_container_binding_logs_08 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 8;
INSERT IGNORE INTO cloud_comment_container_binding_logs_09 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 9;
INSERT IGNORE INTO cloud_comment_container_binding_logs_10 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 10;
INSERT IGNORE INTO cloud_comment_container_binding_logs_11 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 11;
INSERT IGNORE INTO cloud_comment_container_binding_logs_12 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 12;
INSERT IGNORE INTO cloud_comment_container_binding_logs_13 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 13;
INSERT IGNORE INTO cloud_comment_container_binding_logs_14 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 14;
INSERT IGNORE INTO cloud_comment_container_binding_logs_15 (
  id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
)
SELECT CAST(l.id AS CHAR), IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), ''),
  l.company_id, l.task_id, l.comment_id, l.binding_id, l.stage, l.message, l.created_at
FROM cloud_comment_container_binding_logs l
WHERE IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '') <> ''
  AND (CRC32(IFNULL((SELECT c.workspace_id FROM cloud_server_configs c
    WHERE c.task_id = l.task_id AND c.workspace_id <> ''
    ORDER BY c.updated_at DESC LIMIT 1), '')) % 16) = 15;
