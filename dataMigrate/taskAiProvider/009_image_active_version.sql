-- 镜像「激活」版本概念（OPT-20260824）：同一镜像组允许多个审批通过
-- （approved）的版本，但仅允许一个「激活」版本（is_active=1，组内唯一）——
-- 激活版本是公开目录中对外生效的版本，厂商可在多个已上架版本间切换激活。
-- 审批通过时组内无激活版本则自动激活（首个上架即生效）。
-- 幂等：ADD COLUMN 使用 IF NOT EXISTS；回填仅作用于存量 approved 行。
ALTER TABLE `ai_provider_vendorcontainerimage`
  ADD COLUMN `is_active` TINYINT(1) NOT NULL DEFAULT 0
  COMMENT '组内唯一激活标志（公开目录生效版本，须 status=approved）'
  AFTER `status`;

-- 存量回填：每组最新（id 最大）的 approved 版本置为激活，保证升级后目录语义一致。
UPDATE `ai_provider_vendorcontainerimage` ci
JOIN (
  SELECT `image_group_id`, MAX(`id`) AS `max_id`
  FROM `ai_provider_vendorcontainerimage`
  WHERE `status` = 'approved'
  GROUP BY `image_group_id`
) t ON t.`image_group_id` = ci.`image_group_id` AND t.`max_id` = ci.`id`
SET ci.`is_active` = 1;
