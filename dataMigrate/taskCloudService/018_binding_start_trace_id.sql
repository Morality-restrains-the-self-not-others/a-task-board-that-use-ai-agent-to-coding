-- 018: 评论容器绑定持久化独立启动 TraceId
-- 背景：start-vm 曾把 run trace 设成 task_id，且仅活在 SSE/日志后缀；
--       同任务多评论启容器须各自独立 ID。一等字段 start_trace_id，禁止写入 task_id。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_comment_container_bindings' AND COLUMN_NAME = 'start_trace_id') = 0, 'ALTER TABLE cloud_comment_container_bindings ADD COLUMN start_trace_id VARCHAR(128) NOT NULL DEFAULT \'\'', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
