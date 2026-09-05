# 平台注册邀请码（每日限量）

- **状态**: in_progress
- **日期**: 2026-07-22
- **服务 Owner**: taskAuth（`task-auth.db`）
- **网关**: taskGateway → taskAuth

## 业务意图

引入与「公司成员邀请」「用户引荐 accessCode」解耦的**平台注册邀请码**：

1. 系统管理员可开关机制，并配置**每日放量**（自然日，时区 `Asia/Shanghai`）。
2. 已登录用户可在个人资料页**申请**邀请码，并查看自己发出的码的使用情况。
3. 机制开启时，邮箱/手机注册必须提交有效未使用邀请码；校验与核销在 taskAuth 注册路径内原子完成。
4. 登录页（含 `add_account=1`）在机制开启时展示邀请码输入，写入 `sessionStorage` 并带到注册页。
5. 管理员可查看注册邀请关系（发放人 → 码 → 使用人/状态）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 管理员更新邀请码策略 | REGISTRATION_INVITE_POLICY_UPDATED | Kafka `registration-invite-policy-updated` | taskAuth `updateRegistrationInvitePolicy` | taskEvents `registration_invite/1_observability` 结构化日志 | — |
| 用户申请发放邀请码 | REGISTRATION_INVITE_CODE_ISSUED | Kafka `registration-invite-code-issued` | taskAuth `applyRegistrationInviteCode` | taskEvents `registration_invite/1_observability` 结构化日志 | — |
| 注册成功核销邀请码 | REGISTRATION_INVITE_CODE_REDEEMED | Kafka `registration-invite-code-redeemed` | taskAuth email/phone register redeem | taskEvents `registration_invite/1_observability` 结构化日志 | — |
| 公开查询策略/列表查询 | — | — | — | — | 纯查询 |

## 非目标

- 不替代公司 `Invitation` / People 邀请流
- 不替代推荐页 `accessCode` 引荐推荐收益
- 不做邀请码邮件群发（可后续迭代）

## 验收要点

- 关闭：注册不要求邀请码；申请接口返回业务关闭错误
- 开启且配额耗尽：申请失败；注册用旧码仍可（若未使用）
- 一码一用；并发双核销仅一成功
- Admin / Profile / Register / Login UI 与 API 一致
