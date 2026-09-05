# NFR 澄清：模拟登录审计标识与收信箱

- **日期**: 2026-08-23
- **增量**: impersonation audit label + user inbox

## 路径分片键审视

| 路径 | 是否携带可分片 ID | 该 ID 是否合适分片键 | 可伸缩性 | 动作 |
|------|-------------------|----------------------|----------|------|
| POST /api/system-admin/users/{id}/impersonate/ | 是，target user id | 是（按被模拟用户） | L1 | 维持路径；会话表已有 target_user_id 索引 |
| GET /api/auth/inbox/ | 否（身份来自登录用户） | 登录 user id 即收件人，适合分片 | L1 | 查询必须绑定 recipient=self；升级触发：单用户信件 > 10 万 |
| PATCH /api/auth/inbox/{id}/read/ | 是，message id | message id 非租户键；须同时校验 recipient | L1 | 禁止只按 id 更新 |
| Kafka user-inbox-message-created | 是，message id / recipient | recipient_user_id 适合 | L0 | 本期无消费者 |
| HTTP 访问日志字段注入 | 头内 user / impersonator | 不落业务表 | L0 | 无分片需求 |

L0 理由：日志注入与无消费者事件不产生可分片业务存储。

## 幂等性审视

| 路径 | 副作用 | L | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 键是否与边界同粒度 |
|------|--------|---|------------|--------------|--------|----------|--------------------|
| POST impersonate | 是：会话+信件+事件 | L3 | 双击、HTTP 重试 | 同一操作者对同一目标的同一次模拟开始 | Idempotency-Key + actor | 返回已有会话 | 是 |
| 写收信箱 | 是 | L3 | 同上、会话重放 | 同一 impersonation_session_id 一封信 | impersonation_session_id UNIQUE | 空操作 | 是 |
| PATCH inbox read | 是：read_at | L2 | 双击 | 同一信件已读 | message id + recipient | 已读则保持 | 是 |
| GET inbox | 否 | L0 | — | — | — | 纯查询 | — |
| UserInboxMessageCreated 消费 | 本期无消费 | L0 | — | — | 未来须用 message id | 无自动消费者 | — |

禁止用 tenant_id / user_id 单独作为模拟开始幂等键。
