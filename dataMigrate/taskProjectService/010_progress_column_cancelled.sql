-- 005_progress_column_cancelled.sql — 系统默认进度体系新增"已取消"列
-- 将进度列从 3 列扩展为 4 列: 待处理 > 进行中 > 已完成 > 已取消
-- 幂等: INSERT IGNORE + 稳定 ID

INSERT IGNORE INTO project_progress_columns(id, system_id, name, order_num)
VALUES('pc_def_sys_4', 'ps_default_system', '已取消', 3);
