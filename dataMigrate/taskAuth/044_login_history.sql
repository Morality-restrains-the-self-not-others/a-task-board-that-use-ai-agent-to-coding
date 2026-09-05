-- 登录历史（时间累积）：每次成功认证一行。
-- 年增量按日活可达百万+：首日按月 RANGE 分区。
-- PK 必须含分区键 logged_in_at。查询热路径：user_id + logged_in_at DESC。

CREATE TABLE IF NOT EXISTS auth_login_history (
  id BIGINT NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  logged_in_at DATETIME(6) NOT NULL,
  client_ip VARCHAR(64) NOT NULL DEFAULT '',
  user_agent VARCHAR(512) NOT NULL DEFAULT '',
  entry VARCHAR(32) NOT NULL,
  method_type VARCHAR(32) NOT NULL DEFAULT '',
  PRIMARY KEY (id, logged_in_at),
  KEY idx_auth_login_history_user_time (user_id, logged_in_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (TO_DAYS(logged_in_at)) (
  PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
  PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
  PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
  PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
  PARTITION p202612 VALUES LESS THAN (TO_DAYS('2027-01-01')),
  PARTITION p202701 VALUES LESS THAN (TO_DAYS('2027-02-01')),
  PARTITION p_future VALUES LESS THAN MAXVALUE
);
