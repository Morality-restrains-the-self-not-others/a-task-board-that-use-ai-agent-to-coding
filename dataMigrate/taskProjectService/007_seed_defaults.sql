-- 001_seed_defaults.sql — 系统默认交付物体系 + 进度体系
-- 幂等：所有写入使用 INSERT OR IGNORE + 稳定 ID

-- ============================================================================
-- 默认交付物体系（全局默认）
-- ============================================================================
INSERT IGNORE INTO project_deliverable_systems(id, name, description, company_id, is_system, is_default)
VALUES('ds_default_global', '全局默认交付物体系', '系统内置默认交付物体系', '', 1, 1);

-- 全局默认交付物体系层级结构: 价值流 > 业务流程 > 活动 > 工作项/交付物 > 子工作项/子交付物
INSERT IGNORE INTO project_deliverable_columns(id, system_id, name, order_num) VALUES('dc_def_global_lv1', 'ds_default_global', '价值流', 0);
INSERT IGNORE INTO project_deliverable_columns(id, system_id, name, order_num) VALUES('dc_def_global_lv2', 'ds_default_global', '业务流程', 1);
INSERT IGNORE INTO project_deliverable_columns(id, system_id, name, order_num) VALUES('dc_def_global_lv3', 'ds_default_global', '活动', 2);
INSERT IGNORE INTO project_deliverable_columns(id, system_id, name, order_num) VALUES('dc_def_global_lv4', 'ds_default_global', '工作项 / 交付物', 3);
INSERT IGNORE INTO project_deliverable_columns(id, system_id, name, order_num) VALUES('dc_def_global_lv5', 'ds_default_global', '子工作项 / 子交付物', 4);

-- ============================================================================
-- 默认进度体系（系统默认）
-- ============================================================================
INSERT IGNORE INTO project_progress_systems(id, name, is_default) VALUES('ps_default_system', '系统默认进度体系', 1);

-- 系统默认进度列: 待处理 > 进行中 > 已完成 > 已取消
INSERT IGNORE INTO project_progress_columns(id, system_id, name, order_num) VALUES('pc_def_sys_1', 'ps_default_system', '待处理', 0);
INSERT IGNORE INTO project_progress_columns(id, system_id, name, order_num) VALUES('pc_def_sys_2', 'ps_default_system', '进行中', 1);
INSERT IGNORE INTO project_progress_columns(id, system_id, name, order_num) VALUES('pc_def_sys_3', 'ps_default_system', '已完成', 2);
INSERT IGNORE INTO project_progress_columns(id, system_id, name, order_num) VALUES('pc_def_sys_4', 'ps_default_system', '已取消', 3);

-- 去重：确保最多只有一个默认进度体系
UPDATE project_progress_systems SET is_default = 0 WHERE is_default = 1 AND id != 'ps_default_system';

-- 去重：确保最多只有一个系统默认交付物体系
UPDATE project_deliverable_systems SET is_default = 0 WHERE is_system = 1 AND is_default = 1 AND id != 'ds_default_global';
