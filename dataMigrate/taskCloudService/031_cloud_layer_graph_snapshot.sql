-- 031: cloud_layer_graph_snapshot — 评论级 ztree 层图 last-write-wins 快照
-- 伸缩：非时间累积（每评论一行）；访问键 (workspace_id, task_id, comment_id)；HASH 预留、一期不分片
-- 主键 Snowflake 字符串；禁止 AUTO_INCREMENT

CREATE TABLE IF NOT EXISTS cloud_layer_graph_snapshot (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
  workspace_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  graph_json JSON NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_ws_task_comment (workspace_id, task_id, comment_id),
  KEY idx_task_comment (task_id, comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
