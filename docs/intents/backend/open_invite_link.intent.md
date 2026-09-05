# 开放式邀请链接（租户）

- **状态**: accepted
- **日期**: 2026-08-29
- **服务 Owner**: taskTenantService（`tenant_invitation` / `tenant_invitation_redemption`）
- **设计:** `docs/superpowers/specs/2026-08-29-open-invite-link-design.md`

## 业务意图

租户管理员可创建**开放式**成员邀请链接：同一 token 允许多名未入司用户加入，直到过期或 `max_uses` 耗尽。默认链接仍为单次。邮箱/电话邀请保持单次。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|------------|--------|--------------|---------|
| 创建邀请链接（含开放） | INVITATION_CREATED | Kafka invitation-created | handleInvite | 既有投递；link 无 SMTP | — |
| 用户接受邀请加入 | MEMBER_JOINED | Kafka member-joined | handleJoin | 既有 git identity 等 | — |
| validate / pending 列表 | — | — | — | — | 纯查询 |

## 验收要点

- 缺省 invite 行为与改造前一致（单次）
- 开放链两名不同用户 join 均 201；第三人在 max_uses=2 时 400
- 并发超额只成功 N 人
- 同用户二次 join 400「您已在公司中」
- 非 link 渠道 open → 400
