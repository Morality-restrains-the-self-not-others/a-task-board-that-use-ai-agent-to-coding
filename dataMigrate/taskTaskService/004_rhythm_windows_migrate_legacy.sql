-- taskTaskService: Migrate legacy rhythm window data from flat columns to windows table
-- Extracted from Go migrateLegacyRhythmWindows() data migration step
-- Guard: daily_start column still exists in top_deliverable_schedule_rhythms
-- Idempotent: INSERT only for tasks not already present in windows table
-- Applied once via data_migrate_log tracking.

SET @has_legacy = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'top_deliverable_schedule_rhythms' AND COLUMN_NAME = 'daily_start');

SET @stmt = IF(@has_legacy > 0, 'INSERT INTO top_deliverable_schedule_rhythm_windows(id, task_id, tenant_id, workspace_id, daily_start, daily_end, max_queued_machines, auto_close, auto_close_warn_minutes, auto_close_warn_key, auto_close_release_key, sort_order, created_at, updated_at) SELECT CONCAT(task_id, \'_win_migrated\'), task_id, tenant_id, workspace_id, COALESCE(daily_start, \'\'), COALESCE(daily_end, \'\'), COALESCE(max_queued_machines, 0), COALESCE(auto_close, 0), COALESCE(auto_close_warn_minutes, 5), COALESCE(auto_close_warn_key, \'\'), COALESCE(auto_close_release_key, \'\'), 0, NOW(), NOW() FROM top_deliverable_schedule_rhythms WHERE (daily_start != \'\' OR daily_end != \'\') AND task_id NOT IN (SELECT task_id FROM top_deliverable_schedule_rhythm_windows)', 'SELECT 1 \'skip: legacy rhythm windows not present\'');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
