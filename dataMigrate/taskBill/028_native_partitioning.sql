-- 028: billing_transaction / billing_usage / billing_outbox_message — MySQL 原生 RANGE 分区
-- 策略 A：按月 RANGE 分区，主键/唯一键均包含分区键 created_at
-- 前置条件：表为空或数据可清理（无真实业务数据）
--
-- 设计要点：
--   1. PRIMARY KEY (id, created_at) — 分区键必须包含在所有唯一键中
--   2. billing_transaction.transaction_id UNIQUE 也需加入 created_at
--   3. 按月分区，提前创建 4 个月 + p_future 兜底
--   4. 冷数据清理：ALTER TABLE ... TRUNCATE PARTITION（秒级，零风险）
--   5. 分区运维：db/scripts/ensure_partitions.sh 每月自动拆分 p_future

-- ============================================================
-- billing_transaction — 交易流水
-- ============================================================

-- Step 1: 移除原有唯一键约束（transaction_id 不含 created_at 无法在分区表中保留）
ALTER TABLE billing_transaction DROP INDEX transaction_id;

-- Step 2: 修改主键为复合主键（id + 分区键）
ALTER TABLE billing_transaction DROP PRIMARY KEY, ADD PRIMARY KEY (id, created_at);

-- Step 3: 重建唯一索引（含分区键）
ALTER TABLE billing_transaction ADD UNIQUE INDEX uq_transaction (transaction_id, created_at);

-- Step 4: 添加分区
ALTER TABLE billing_transaction
PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- ============================================================
-- billing_usage — 资源用量
-- ============================================================

-- PK 改为复合主键（usage_time 是此表的时间列）
ALTER TABLE billing_usage DROP PRIMARY KEY, ADD PRIMARY KEY (id, usage_time);

ALTER TABLE billing_usage
PARTITION BY RANGE (TO_DAYS(usage_time)) (
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- ============================================================
-- billing_outbox_message — 消息队列（分区 + TTL 结合）
-- ============================================================

ALTER TABLE billing_outbox_message DROP PRIMARY KEY, ADD PRIMARY KEY (id, created_at);

ALTER TABLE billing_outbox_message
PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p202608 VALUES LESS THAN (TO_DAYS('2026-09-01')),
    PARTITION p202609 VALUES LESS THAN (TO_DAYS('2026-10-01')),
    PARTITION p202610 VALUES LESS THAN (TO_DAYS('2026-11-01')),
    PARTITION p202611 VALUES LESS THAN (TO_DAYS('2026-12-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
