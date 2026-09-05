-- taskAIComment: Legacy column additions for databases created before dataMigrate
-- Each ALTER is idempotent via information_schema check.

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'ai_comment_container_agent_comments' AND COLUMN_NAME = 'context_pack_json') = 0, 'ALTER TABLE ai_comment_container_agent_comments ADD COLUMN context_pack_json TEXT', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'ai_comment_task_comments' AND COLUMN_NAME = 'execution_mode') = 0, 'ALTER TABLE ai_comment_task_comments ADD COLUMN execution_mode VARCHAR(64) NOT NULL DEFAULT ''wait_previous''', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
