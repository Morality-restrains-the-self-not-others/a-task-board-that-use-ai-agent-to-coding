-- OPT-20260818-015 owner DB 幂等兜底（gap: cloud_server_stopped）
-- CLOUD_SERVER_STOPPED 消费幂等仅靠进程内 MemoryStore + 载荷 stop_request_id（从未落库）；
-- 消费者进程重启后重放同一 stop_request_id 会再次 DeleteInstance / ClearAfterStop。
-- 此处把 stop_request_id 落库为唯一键，作为 owner 侧最终去重（约束 54 §4）。

CREATE TABLE IF NOT EXISTS cloud_stop_request (
  id BIGINT NOT NULL AUTO_INCREMENT,
  stop_request_id VARCHAR(128) NOT NULL,
  tenant_id VARCHAR(64) NOT NULL DEFAULT '',
  workspace_id VARCHAR(64) NOT NULL DEFAULT '',
  task_id VARCHAR(64) NOT NULL,
  instance_id VARCHAR(128) NOT NULL DEFAULT '',
  processed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_csr_stop_request (stop_request_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
