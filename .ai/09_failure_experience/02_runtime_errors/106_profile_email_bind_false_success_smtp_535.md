# [运行时] Profile 邮箱绑定「验证码已发送」但收件箱为空

## 基本信息

- 案例编号：FE-20260901-EMAIL-BIND-FALSE-SUCCESS
- 录入日期：2026-09-01
- 关联服务：`task-auth`、`task-events-email-sent-1-send-email`、`taskFE`

## 失败现象

- 页面：`/profile/?sso_error=email_required#rg=profile.email_binding`
- 可见文案：`验证码已发送，请查收邮箱`（`p.text-sm.text-text-light`）
- 用户未收到邮件
- HTTP `POST /api/accounts/users/send_verification_code/` **200**（约 1s）
- traceId：`959d757f-8034-428f-b31f-091b9ee1a2e9`、`7fdf1b90-aee4-4780-97b8-5488582f9e96`

## 根因

1. **假成功**：taskAuth Kafka 入队 `EMAIL_SENT` 即 200；前端把 200 当成已投递。
2. **消费者未叠 conf-local**：`LoadSettings` 直读 tracked `email.yaml`（`host_password: ''`），QQ SMTP `535 Login fail`。
3. **OTP fallback 正文**：`verification_code` 无专用模板，SMTP 回退只有标题。

## 解决方案

1. `ReadAppFragment` 叠 `conf-local/events/domain-events/email.yaml`。
2. `verification_code` 同步 SMTP，失败 502；成功后再 Kafka 且 `pre_delivered=true`。
3. fallback 正文含 OTP。

## 预防

- 含机密的 YAML 禁止 `os.ReadFile` 绕过 conf-local。
- OTP 等短 TTL 通知禁止「入队即成功」。
