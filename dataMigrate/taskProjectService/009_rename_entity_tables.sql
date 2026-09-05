-- Migration 003: Rename entity tables to complete prefix compliance.
-- Rule: taskProjectService → project_ prefix.
-- Uses non-colliding names to avoid conflict with junction table `project_workspaces`.

SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='projects') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='project_entries'),'ALTER TABLE `projects` RENAME TO `project_entries`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='workspaces') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='project_workspace_entries'),'ALTER TABLE `workspaces` RENAME TO `project_workspace_entries`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
