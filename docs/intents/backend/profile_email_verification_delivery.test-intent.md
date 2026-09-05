# 测试意图：个人资料邮箱绑定验证码须真实投递

## 覆盖的功能意图

`docs/intents/backend/profile_email_verification_delivery.intent.md`

## 测例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `LoadSettings` 叠 conf-local `host_password` | 空骨架被 overlay 覆盖 |
| T2 | fallback `verification_code` 正文 | 含 OTP |
| T3 | 非 mock + SMTP 不可达 | `sendEmailVerificationCode` 返回「邮件发送失败」；HTTP 502 |
| T4 | `pre_delivered=true` | EMAIL_SENT 消费者 Success 且不拨 SMTP |
| T5 | 前端发送 502 | 展示「邮件发送失败」且不含「验证码已发送，请查收邮箱」；`data-traceId` 非空 |

## 对应测试文件

- `taskEvents/notifications/cfg/settings_overlay_test.go`
- `taskAuth/src/events_email_fallback_test.go`
- `taskAuth/src/verification_code_test.go` `TestSendEmailVerificationCodeReportsDeliveryFailureWhenNotMock`
- `taskEvents/notifications/delivery_skip_test.go` `TestEmailSentSkipsWhenPreDelivered`
- `taskFE/app/src/components/UserProfileEmailBindingPanel.test.js`
