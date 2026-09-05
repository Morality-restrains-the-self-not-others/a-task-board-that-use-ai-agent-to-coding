-- 005: cloud_server_events / cloud_server_config_histories / cloud_feature_params_access_audit — MySQL 原生 RANGE 分区
-- 策略 A：按月 RANGE 分区，主键均包含分区键 created_at
-- 前置条件：表为空或数据可清理（无真实业务数据）
--
-- 设计要点：
--   1. 所有 Snowflake VARCHAR(64) 主键均改为 (id, created_at) 复合主键
--   2. 按月分区，提前创建 4 个月 + p_future 兜底
--   3. 冷数据清理：ALTER TABLE ... TRUNCATE PARTITION（秒级）
--   4. 分区运维：db/scripts/ensure_partitions.sh 每月自动拆分 p_future

-- ============================================================
-- cloud_server_events — 云服务事件日志
-- ============================================================
ALTER TABLE cloud_server_events DROP PRIMARY KEY, ADD PRIMARY KEY (id, created_at);

ALTER TABLE cloud_server_events
PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- ============================================================
-- cloud_server_config_histories — 云服务器配置历史
-- ============================================================
ALTER TABLE cloud_server_config_histories DROP PRIMARY KEY, ADD PRIMARY KEY (id, created_at);

ALTER TABLE cloud_server_config_histories
PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- ============================================================
-- cloud_feature_params_access_audit — 访问审计日志
-- ============================================================
ALTER TABLE cloud_feature_params_access_audit DROP PRIMARY KEY, ADD PRIMARY KEY (id, created_at);

ALTER TABLE cloud_feature_params_access_audit
PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
