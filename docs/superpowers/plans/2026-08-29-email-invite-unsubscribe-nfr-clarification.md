# 邮件邀请退订 — NFR 澄清

- **日期**: 2026-08-29
- **价值流**: `docs/superpowers/plans/2026-08-29-email-invite-unsubscribe-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 适配性 | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| GET/POST `/api/public/email-unsubscribe/` | 无 tenant；键为 email | email 是自然键，非租户分片 | L0：退订集合全局、增长慢 | 升级触发：单表 >100 万行再按 email hash 分片 |
| GET `/api/internal/email-unsubscription/?email=` | email | 与表 UNIQUE(email) 对齐 | L0 | 同上 |
| POST 超管 email-invitations | 无 tenant | 平台级邀请 | L0 | 不补 tenant 键 |
| POST `/api/tenant/{id}/accounts/members/invite/` | tenantId | 合适 | L2 沿用既有租户边界 | 无 |
| Kafka `email-unsubscribed` | email（规范化） | 合适 | L1 可按 email 分区 | 键用 email 非 user_id |
| SPA `/auth/unsubscribe/` | 无 | n/a | L0 | 静态确认 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 业务边界 | 幂等键 | 重放语义 |
|------|--------|----------|----------|--------|----------|
| 公开退订 | 写退订行 + 事件 | 邮件客户端多次 GET/POST | 同一规范化 email 一条 | `email` UNIQUE | 二次请求成功空操作；事件消费用 `email` |
| 内部 GET | 无 | — | — | L0 | — |
| 超管/租户邀请（已退订） | 写邀请行，跳过 SMTP | 双击 | 每次邀请一条记录 | 前端 `Idempotency-Key` + 既有 pending 去重 | 已有 pending 仍 400 can_resend |
| EMAIL_UNSUBSCRIBED 消费 | 仅日志 | Kafka 重投 | 同一 email 退订事实 | `email`（禁止 company_id） | skip + warn |
| 验证码发送 | 有，但不在本增量拦截 | — | — | 不改 | — |

资金路径不适用。前端退订页为真实 `<a>` / 落地确认：`Anti-Replay-OK: 公开确认页无写按钮`。邀请提交沿用 `createClickGuard`。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 安全 | L3 | 公开写靠 HMAC；日志不打完整 token |
| 数据一致性 | L2 | UNIQUE(email)；事件 after commit |
| 容错 | L2 | 内部查询失败 fail-open 发信并 warn |
| 可伸缩性 | L0 | 见上表 |
| 性能 | L1 | 邀请路径一次内部 HTTP |
| 可观测性 | L2 | `email_unsubscribed` / `invite_email_skipped` 结构化日志 |

## 领域模型影响

聚合 `EmailUnsubscription` 以规范化 email 为身份；无租户维度。邀请聚合不持有退订状态，发送前查询端口。
