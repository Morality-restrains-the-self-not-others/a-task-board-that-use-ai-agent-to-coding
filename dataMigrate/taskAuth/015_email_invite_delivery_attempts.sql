-- 邮件投递尝试历史记录表：记录每次投递尝试的详细信息
-- 替代之前覆盖式更新 delivery_status/delivery_error，实现完整审计追溯
-- 与 auth_email_registration_invite 的 delivery_status/delivery_error 保持同步：
--   delivery_status 始终反映最新一次投递的状态（方便快速筛选）
--   delivery_attempt 表提供完整历史（方便审计和问题排查）
CREATE TABLE auth_email_invite_delivery_attempt (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    invitation_id BIGINT NOT NULL COMMENT '关联的邀请记录 ID',
    attempt_number INT NOT NULL DEFAULT 1 COMMENT '第几次投递尝试',
    delivery_method VARCHAR(20) NOT NULL COMMENT '投递方式: kafka/smtp',
    delivery_status VARCHAR(20) NOT NULL COMMENT '本次投递状态: queued/delivered/failed',
    delivery_error TEXT DEFAULT NULL COMMENT '本次投递失败原因（如有）',
    attempted_at DATETIME NOT NULL COMMENT '投递尝试时间',
    INDEX idx_invitation_id (invitation_id),
    INDEX idx_attempted_at (attempted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮件投递尝试历史记录';
