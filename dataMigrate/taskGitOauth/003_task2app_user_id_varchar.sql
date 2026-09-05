-- Migration 003: task2app_user_id bigint → varchar(36)，对齐 taskAuth auth_user.id
-- (varchar(36) COLLATE utf8mb4_unicode_ci)。
-- 背景（2026-08-07）：taskAuth 用户 ID 为字符串体系，确定性 ID（bootstrap-admin）
-- 非数字。taskGitOauth 此前假设 X-User-Id 为数字 → 网关启动 OAuth 必 503
-- bad_x_user_id。列类型与全链路 Go 代码同步放宽为字符串。
-- 既有数字型 ID（如 873438061961179136）转 varchar 无损。

ALTER TABLE `git_oauth_appusercredential`
  MODIFY COLUMN `task2app_user_id` varchar(36) NOT NULL;

ALTER TABLE `git_oauth_appaccesstokenuseaudit`
  MODIFY COLUMN `task2app_user_id` varchar(36) NOT NULL;

ALTER TABLE `git_oauth_taskcredentialaudit`
  MODIFY COLUMN `task2app_user_id` varchar(36) NULL;
