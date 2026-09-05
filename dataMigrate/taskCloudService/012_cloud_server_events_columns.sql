-- taskCloudService: cloud_server_events legacy columns + index
-- Extracted from Go migrateCloudServerEventsColumns()
-- Each statement is idempotent via information_schema checks.
-- Applied once via data_migrate_log tracking.

-- Add comment_id column if missing
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_events' AND COLUMN_NAME = 'comment_id') = 0, 'ALTER TABLE cloud_server_events ADD COLUMN comment_id VARCHAR(64) NOT NULL DEFAULT \'\'', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Add company_member_id column if missing
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_events' AND COLUMN_NAME = 'company_member_id') = 0, 'ALTER TABLE cloud_server_events ADD COLUMN company_member_id VARCHAR(64) NOT NULL DEFAULT \'\'', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Create idx_cse_comment index if missing
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_events' AND INDEX_NAME = 'idx_cse_comment') = 0, 'CREATE INDEX idx_cse_comment ON cloud_server_events(comment_id)', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
