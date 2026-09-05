# 功能意图：管理员模拟用户登录（后端）

- **日期**: 2026-08-23
- **状态**: 实施中
- **服务**: taskAuth
- **ADR**: ADR-0037 / ADR-0038

## 背景与目标

持有 `user:impersonate` 的平台角色可对指定用户签发独立模拟会话，网关将该请求鉴权为目标用户，同时保留操作者身份供退出与审计。

## 范围与边界

- 范围内：开始/结束/状态三个 API；`auth_impersonation_session`（含 reason）；forward-auth 识别模拟 token；Kafka 事件；收信箱通知。
- 范围外：租户内成员互模拟、GitLab impersonation、改用户密码。

## 约束与风险

- 独立 token，不碰目标 `auth_customtoken`。
- 防提权：模拟超管须操作者有 `platform:manage`。
- 日志禁止输出 token / 邮箱 / 手机号 / 完整理由正文。
- 开始模拟必须 JSON `reason`（8～500 字），并给被模拟用户写信。

## 验收标准

1. 无 `user:impersonate` 调用开始接口 → 403。
2. 有权限且目标可登录 → 200，token 解析为 target id。
3. 自模拟 → 409。嵌套模拟他人 → 409。**同一目标已有开放会话** → 200 重放该会话并返回目标前台 `redirect_url`（不是系统管理页）。停用用户 → 422。
4. stop 后凭据恢复为 actor。
5. 成功路径投递对应领域事件。
6. 无 reason 或过短 → 400。
7. 成功后收信箱有一封含理由的信；同一 session 不重复写信。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 消费者 |
|----------|--------|----|--------|--------|
| 开始模拟 | UserImpersonationStarted | user-impersonation-started | ImpersonationAppService | 人工审计（无自动消费者） |
| 结束模拟 | UserImpersonationStopped | user-impersonation-stopped | ImpersonationAppService | 同上 |
| 收信箱新信 | UserInboxMessageCreated | user-inbox-message-created | ImpersonationAppService | 本期无自动消费者 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-23 | 初稿 |
| 2026-08-23 | ADR-0038：理由 + 收信箱 + 日志标识 |
| 2026-08-23 | 同一目标开放会话重放 200，落点禁止系统管理页 |
