-- 005_fix_charset_mojibake.sql — 修复因 mysql CLI 缺少 --default-character-set=utf8mb4 导致的乱码
-- 根因：apply_datamigrate.sh 的 mysql CLI 未指定字符集，导致 UTF-8 中文被当作 Latin-1 存储
-- 修复：删除受影响的静态种子数据行，以稳定 ID 重新插入
-- 幂等：DELETE + INSERT IGNORE 组合可重复执行

-- ============================================================================
-- 1. 修复全局默认交付物体系层级结构（project_deliverable_columns）
-- ============================================================================

-- 删除可能已损坏的种子数据行（仅删除已知稳定 ID，不动用户自建数据）
DELETE FROM project_deliverable_columns
WHERE system_id = 'ds_default_global'
  AND id IN (
    'dc_def_global_lv1',
    'dc_def_global_lv2',
    'dc_def_global_lv3',
    'dc_def_global_lv4',
    'dc_def_global_lv5'
  );

-- 重新插入正确编码的层级列
INSERT IGNORE INTO project_deliverable_columns(id, system_id, name, order_num)
VALUES
    ('dc_def_global_lv1', 'ds_default_global', '价值流', 0),
    ('dc_def_global_lv2', 'ds_default_global', '业务流程', 1),
    ('dc_def_global_lv3', 'ds_default_global', '活动', 2),
    ('dc_def_global_lv4', 'ds_default_global', '工作项 / 交付物', 3),
    ('dc_def_global_lv5', 'ds_default_global', '子工作项 / 子交付物', 4);

-- ============================================================================
-- 2. 修复系统默认交付物体系名称（project_deliverable_systems）
-- ============================================================================
UPDATE project_deliverable_systems
SET name = '全局默认交付物体系',
    description = '系统内置默认交付物体系'
WHERE id = 'ds_default_global';

-- ============================================================================
-- 3. 修复系统默认进度体系列名（project_progress_columns）
-- ============================================================================
DELETE FROM project_progress_columns
WHERE system_id = 'ps_default_system'
  AND id IN (
    'pc_def_sys_1',
    'pc_def_sys_2',
    'pc_def_sys_3',
    'pc_def_sys_4'
  );

INSERT IGNORE INTO project_progress_columns(id, system_id, name, order_num)
VALUES
    ('pc_def_sys_1', 'ps_default_system', '待处理', 0),
    ('pc_def_sys_2', 'ps_default_system', '进行中', 1),
    ('pc_def_sys_3', 'ps_default_system', '已完成', 2),
    ('pc_def_sys_4', 'ps_default_system', '已取消', 3);
