-- ═══════════════════════════════════════════════════════════════
-- taskTenantService: 租户邀请投递状态跟踪
-- 对齐 taskAuth auth_email_registration_invite 既有 delivery_status 约定：
--   none      — link 渠道（分享链接，无投递动作）
--   queued    — email/phone 渠道，事件已发布待异步投递（Kafka/SMTP）
--   delivered — SMTP/短信发送成功（taskEvents 消费者回调）
--   failed    — 投递失败（delivery_error 记录原因）
-- 消费方：taskEvents INVITATION_CREATED 消费者（投递后回调
--          /api/internal/tenant/invitations/{id}/delivery-callback/）
-- 幂等: guarded_add_column + data_migrate_log（与 taskAuth/033 同模式）。
-- 若日志被清且列已存在时重跑，裸 ADD COLUMN 会失败；守卫下为 no-op。
-- ═══════════════════════════════════════════════════════════════

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_008;
CREATE PROCEDURE guarded_add_column_008(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND COLUMN_NAME = col
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD COLUMN `', col, '` ', col_def);
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //
DELIMITER ;

CALL guarded_add_column_008('tenant_invitation', 'delivery_status', 'VARCHAR(16) NOT NULL DEFAULT ''none''');
CALL guarded_add_column_008('tenant_invitation', 'delivery_error', 'VARCHAR(512) DEFAULT NULL');
CALL guarded_add_column_008('tenant_invitation', 'email_sent_at', 'DATETIME DEFAULT NULL');

DROP PROCEDURE IF EXISTS guarded_add_column_008;
