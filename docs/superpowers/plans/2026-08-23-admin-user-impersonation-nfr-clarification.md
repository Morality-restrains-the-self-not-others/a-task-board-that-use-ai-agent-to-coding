# NFR 澄清：管理员模拟用户登录

- **Date:** 2026-08-23
- **Value stream:** `docs/superpowers/plans/2026-08-23-admin-user-impersonation-value-stream.md`
- **Default level:** L3 for auth/security（auto-flow: auth 域抬到 L3）

## 路径分片键强制审视

| 路径 | 方法 | 是否携带分片 ID | 该 ID 是否合适分片键 | 可伸缩性 | 动作 |
|------|------|-----------------|----------------------|----------|------|
| `/api/system-admin/users/{id}/impersonate/` | POST | 有 `user id`（目标） | 否：模拟是平台操作，伸缩要素是 actor 操作频率而非租户 | L1 | 不按 tenant 分库；热表按时间归档。路径保留 user id 以防 IDOR 模糊 |
| `/api/auth/impersonation/stop/` | POST | 无 tenant；会话由 token 定位 | token 非分片键 | L0 | 全局会话表，年增量 <10 万。升级触发：年会话 >100 万再按 actor 哈希 |
| `/api/auth/impersonation/status/` | GET | 无 | — | L0 | 只读，按 token 点查 |
| `/system-admin/users/` 编辑页 | UI | 无 | — | L0 | 平台控制台 |
| Kafka `user-impersonation-started` | 消息 | payload `actor_user_id`+`target_user_id` | 审计事件，不按租户消费 | L0 | key 用 `session_id` |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 键粒度判定 |
|------|--------|------------|--------------|--------|----------|------------|
| POST impersonate | 签发会话 + cookie + 事件 | 双击、重试、网关重放 | 同一 actor 对同一 target 的一次未结束模拟 | `Idempotency-Key`；服务端另约束「actor 已有 open session」 | 同键返回同一 token；不同键但已在模拟中 → 409 | 与边界同粒度（一次 open session） |
| POST stop | 结束会话 + 恢复 cookie + 事件 | 双击退出 | 该 `session_id` 结束一次 | session token | 已结束再 stop → 200 空操作（ended_at 不变） | 会话级 |
| GET status | 无 | — | — | — | L0 只读 | — |
| Kafka 消费 | 本期无自动消费者 | — | — | 若未来消费须用 `session_id`+event type | — | 禁止用 actor/tenant 当键 |

资金/配额路径：本增量不触达资金。鉴权路径按 L3：DB 唯一约束 `(token_key)` + 幂等键列。

## 类别支撑程度

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全性 | L3 | 独立 token、防提权、审计事件、不打 PII |
| 数据一致性 | L2 | 会话行与 cookie 同请求内完成；事件异步 |
| 容错 | L2 | Kafka 失败只记日志不阻断签发（与 USER_LOGGED_IN 一致） |
| 可伸缩性 | L1 | 见路径表 |
| 可观测性 | L2 | 结构化日志 event=impersonation_* + trace_id |

## 质量场景

1. **刺激：** 无权限员工 POST impersonate。**响应：** 403，表无新行，无事件。
2. **刺激：** 300ms 内双击按钮。**响应：** 仅一次 HTTP，第二次被前端门闩拦住；若两请求同 Idempotency-Key 则同一 token。
3. **刺激：** 模拟中访问工作台 API。**响应：** X-User-Id=target，X-Impersonator-Id=actor。

## 领域模型影响

- Aggregate 根 ImpersonationSession 必须编码「至多一个 open session / actor」。
- 幂等键存在聚合上，而不是 handler 旁路。
