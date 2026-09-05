-- OPT-20260809-011: 评论容器绑定启动阶段事件持久化
-- 表：cloud_comment_container_binding_logs（评论容器启动过程权威时间线）
--    冷打开/换设备时前端可还原「排队/等待前序/启动/分配/就绪」完整历史，
--    不再依赖页面打开后才开始派生的本地 SSE 日志。
-- 写入节点：ensure(pending) / ccbStartBinding(starting,csc_allocated,running) /
--    attach_csc / promote_running / waiting_previous / completed / released。
-- 与 cloud_comment_container_bindings 一对多：comment_id 关联。
-- id 使用自增主键保证同一秒内多行仍按写入顺序返回（DATETIME 仅秒级精度，
-- 依赖 id 递增区分同秒先后）。

CREATE TABLE IF NOT EXISTS cloud_comment_container_binding_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  company_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  binding_id VARCHAR(64) NOT NULL DEFAULT '',
  stage VARCHAR(64) NOT NULL DEFAULT '',
  message VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ccbl_task (company_id, task_id),
  KEY idx_ccbl_comment (company_id, task_id, comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
