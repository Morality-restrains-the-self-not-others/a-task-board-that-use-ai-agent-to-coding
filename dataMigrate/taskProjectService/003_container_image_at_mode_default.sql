-- Migration: 容器镜像 @ 模式默认开启
-- Change container_image_at_mode_enabled default from 0 to 1
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='workspaces'),'ALTER TABLE `workspaces` MODIFY COLUMN container_image_at_mode_enabled TINYINT NOT NULL DEFAULT 1','ALTER TABLE `project_workspace_entries` MODIFY COLUMN container_image_at_mode_enabled TINYINT NOT NULL DEFAULT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
