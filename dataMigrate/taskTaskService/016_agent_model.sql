-- taskTaskService: 派生副本所选智能体模型持久化到任务行（OPT-20260825-014）
-- agent_model — 创建/派生时 auto_run agent_models[0].model 的快照；
--               供任务详情层图「选择模型」默认展示（不依赖任务级 feature-params）。
-- 幂等：information_schema 列存在性守卫（与既有 015 风格一致）。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND COLUMN_NAME = 'agent_model') = 0,
    'ALTER TABLE task_tasks ADD COLUMN agent_model VARCHAR(128) NULL AFTER container_image_snapshot',
    'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
