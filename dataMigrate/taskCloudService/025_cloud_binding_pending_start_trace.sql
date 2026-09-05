-- 025: 评论容器绑定 pending 启机 TraceId 旁路表（跨重启/多副本暂存）
-- 背景：start-vm persist 早于 binding INSERT 时，暂存曾用进程内 sync.Map，
--       进程重启或多副本会丢失，冷打开第二条评论仍缺启动 TraceId。
-- 该表为旁路暂存：drain 回写 cloud_comment_container_bindings.start_trace_id 后删除行。

CREATE TABLE IF NOT EXISTS cloud_binding_pending_start_trace (
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  start_trace_id VARCHAR(128) NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (task_id, comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
