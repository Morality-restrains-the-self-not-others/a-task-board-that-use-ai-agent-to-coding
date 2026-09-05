-- 镜像审核历史表（OPT-20260824）：此前 InsertReviewHistory 一直写入
-- ai_provider_containerimagereviewhistory，但该表从未在任何迁移中创建，
-- 导致 approve/reject 的历史记录全部静默丢失（错误被忽略）。
-- 创建后由 runAll /api/dev/init-databases 幂等应用。
CREATE TABLE IF NOT EXISTS `ai_provider_containerimagereviewhistory` (
  `id` BIGINT NOT NULL PRIMARY KEY,
  `action` VARCHAR(32) NOT NULL DEFAULT '',
  `note` TEXT NOT NULL,
  `reviewed_at` DATETIME NULL,
  `created_at` DATETIME NULL,
  `container_image_id` BIGINT NOT NULL,
  `reviewer_id` BIGINT NULL,
  KEY `idx_review_history_container` (`container_image_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
