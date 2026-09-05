-- taskTaskService: 任务镜像技能绑定（D2=B 设计「镜像/技能名↔ID 映射存储」）
-- image_skill_id        — 任务绑定技能的稳定 ID（sk_<sha1(name+镜像seed)>[:12]，taskCloudService 派生）
-- container_image_snapshot — 保存时定格的「名↔ID 映射快照」JSON：
--                           {image_id, image_name, skill_id?, skill_name?}；
--                           镜像目录后续变化不影响任务显示与编辑回填。
-- 幂等：information_schema 列存在性守卫（与既有 038 风格一致）。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND COLUMN_NAME = 'image_skill_id') = 0,
    'ALTER TABLE task_tasks ADD COLUMN image_skill_id VARCHAR(64) NULL AFTER installed_image_id',
    'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND COLUMN_NAME = 'container_image_snapshot') = 0,
    'ALTER TABLE task_tasks ADD COLUMN container_image_snapshot JSON NULL AFTER image_skill_id',
    'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
