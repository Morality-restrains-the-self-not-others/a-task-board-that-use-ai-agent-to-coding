# 后端：租户邀请有效期上限 365 天

- **状态:** accepted
- **日期**: 2026-08-29
- **服务 Owner**: taskTenantService

## 背景与目标

创建/重发邀请的 `expiration_days` 原先只拒绝 `<=0`，无上限，前端最长只给 30 天。目标：允许最长 365 天，缺省 90 天，并拒绝超过 365 的请求，避免无限期 token。

## 范围与边界

- 范围内：`handleInvite`、`handleResendInvitation` 的 `expiration_days` 规范化。
- 范围外：不改 `INVITATION_CREATED` 契约字段名；不改过期扫描 timer。

## 约束与风险

- 缺省 90 天改变「省略字段」的旧行为（原为 7 天）。
- 重发仍拒绝 `<=0`（显式非法），合法范围 1–365。
- 日志只记天数与 token 指纹，不打完整 token。

## 验收标准

1. `expiration_days=365` 创建邀请 201，响应回显 365。
2. `expiration_days=366` 创建/重发 400。
3. 省略 `expiration_days` 创建时按 90 天计算过期。
4. 重发 `expiration_days=0` 仍 400。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 创建邀请（含更长有效期） | INVITATION_CREATED | Kafka invitation-created | handleInvite | 既有投递 | 沿用既有事件，payload 仍含 expiration_days |
| 重发邀请 | INVITATION_CREATED | Kafka invitation-created | handleResendInvitation | 既有投递 | 沿用既有重发发布 |

## 变更记录

- 2026-08-29：默认 7→90；新增上限 365。
