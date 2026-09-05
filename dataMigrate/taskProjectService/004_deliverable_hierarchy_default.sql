-- 004_deliverable_hierarchy_default.sql — 全局默认交付物体系层级结构调整
-- 将看板列（待处理/进行中/已完成）替换为价值流层级：
--   价值流 > 业务流程 > 活动 > 工作项/交付物 > 子工作项/子交付物
-- 幂等：稳定 ID + INSERT IGNORE + 先删后插

-- 1. 移除旧的看板列
DELETE FROM project_deliverable_columns
WHERE system_id = 'ds_default_global'
  AND id IN ('dc_def_global_1', 'dc_def_global_2', 'dc_def_global_3');

-- 2. 插入新的价值流层级列
INSERT IGNORE INTO project_deliverable_columns(id, system_id, name, order_num)
VALUES
    ('dc_def_global_lv1', 'ds_default_global', '价值流', 0),
    ('dc_def_global_lv2', 'ds_default_global', '业务流程', 1),
    ('dc_def_global_lv3', 'ds_default_global', '活动', 2),
    ('dc_def_global_lv4', 'ds_default_global', '工作项 / 交付物', 3),
    ('dc_def_global_lv5', 'ds_default_global', '子工作项 / 子交付物', 4);
