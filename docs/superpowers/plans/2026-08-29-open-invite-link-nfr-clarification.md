# 开放式邀请链接 — NFR 澄清

- **日期**: 2026-08-29
- **增量**: Inc1–Inc4（见价值流）
- **默认级别**: L2；认证/入职路径一致性 L3

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | 动作 | 可伸缩性 |
|------|---------|--------|------|----------|
| POST `/api/tenant/{tenantId}/accounts/members/invite/` | tenantId | 合适（租户隔离） | 无 | L1 沿用租户边界 |
| GET `.../validate-invite/?token=` | tenantId | 合适 | token 非分片键 | L1 |
| POST `.../join/` | tenantId | 合适 | 无 | L1 |
| GET pending-invitations | tenantId | 合适 | 无 | L1 |
| FE `/tenant/:tenant/people/invite/` | tenant | 合适 | 无 | L1 |
| FE `/tenant/:tenant/people/join/?token=` | tenant | 合适 | 无 | L1 |
| Kafka `invitation-created` / `member-joined` | company_id in payload | 合适 | 沿用 | L1 |

无无键路径。升级触发：单租户开放链 QPS 入职 > 100/s 再评估 invitation_id 热点。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| POST invite | 新 invitation 行 | 双击生成 | 每次点击=新链 | 前端 `Idempotency-Key`（既有 clickGuard） | 同键应只建一条；服务端若无幂等表则靠前端门闩 + 可接受两条链（L2） |
| POST join | 成员 + redemption + use_count | 双击/重试 | (invitation_id, user_id) 与 (user_id, company_id) | 前端 Idempotency-Key + UNIQUE 成员/核销 | 重放：已成员 400；超额 400 |
| INVITATION_CREATED 消费 | 邮件（仅 email） | Kafka 重投 | invitation_id | 既有消费者幂等 | 开放 link 渠道无 SMTP |
| MEMBER_JOINED 消费 | git identity | 重复投递 | member_id | 既有 | 不变 |

资金/配额：否。入职路径一致性 **L3**（UNIQUE + 行锁）。

## 类别支撑程度

| 类别 | 级别 | 说明 |
|------|------|------|
| 安全 | L3 | token 熵；不日志 token；跨租户隔离 |
| 数据一致性 | L3 | 事务 + UNIQUE + 条件 UPDATE |
| 可伸缩性 | L1 | 租户键足够 |
| 容错 | L2 | 失败 400；revoke 即时 |
| 可观测性 | L2 | 结构化日志无 token 全文 |
| 易用性 | L2 | 默认单次不破坏习惯 |

## 质量场景

- **刺激**: 2 用户同时 join max_uses=1 开放链 → **响应**: 仅 1 个 201。
- **刺激**: 默认不选开放 → **响应**: 第二次 join 400（与现网一致）。

## 领域模型影响

Invitation 聚合必须包含 `use_count`/`max_uses` 作为一致性边界；Redemption 是聚合内实体（或子实体），UNIQUE(invitation_id, user_id)。Join 必须在同一事务内完成 member + redemption + 计数。
