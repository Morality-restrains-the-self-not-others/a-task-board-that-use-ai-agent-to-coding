# 补丁：邮箱 OTP 邮件投递走 Kafka

**日期**: 2026-07-15  
**关联**: `2026-07-15-otp-go-full-native-sms-design.md`、清理迭代 `otp-email-and-cleanup-bridges`

## 结论

**是：生产邮件投递应走 Kafka。**

`conf/events/domain-events/config.yaml` 的 SSOT 为 `transport: kafka`；`EMAIL_SENT` 对应 topic `email-sent`（与 Django `KAFKA_TOPICS` / taskEvents `EventTopic` 一致）。消费者为 `taskEvents/email_sent/1_send_email`。

## taskAuth 行为

1. **唯一路径**：taskAuth 直发 Kafka（`publishDomainEventKafka` → topic `email-sent`）
2. **失败语义**：未配置 bootstrap 或发布失败时返回错误；**不**再调用 Django `dispatch-email/`（该端点已删除）
3. **测试**：`SMS_PROVIDER=mock` 时允许跳过投递失败

不再以 Redis Stream 作为主路径（仅当全局 transport 切回 redis 时另议）。
