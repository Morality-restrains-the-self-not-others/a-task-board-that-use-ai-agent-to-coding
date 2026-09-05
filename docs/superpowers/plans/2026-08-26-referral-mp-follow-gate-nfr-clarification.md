# NFR 澄清：推荐资格服务号关注闸门

- **Date:** 2026-08-26
- **Default level:** L2；身份绑定 L3

## 路径分片键审视

| 路径 | 分片 ID | 适配？ | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| POST `/api/auth/wechat/mp/callback/` | 无 tenant；键为 mp openid / unionid | unionid 是身份键，非租户键 | L1：全站关注量远低于百万/年 | L0 理由：平台级身份表；升级触发：pending 表 >100 万行再按月分区 |
| GET `/api/auth/wechat/mp/follow-status/` | `user_id`（会话） | 是，用户级身份 | L0 | 单用户点查 |
| GET referral status | `user_id` | 是 | L0 | 已有 |
| POST referral apply | `user_id` | 是 | L0 | 已有 |
| `WECHAT_MP_SUBSCRIBED` | `openid`（mp） | 是，业务重复边界=一次关注 openid | L1 | 事件键用 openid 非 user_id |

## 幂等性审视

| 路径 | 副作用 | 重复触发 | 业务边界 | 幂等键 | 重放语义 | 前端防重放 |
|------|--------|----------|----------|--------|----------|------------|
| GET MP callback | 无 | — | — | L0 | echostr | n/a |
| POST MP callback subscribe | 写 identity/pending + 事件 | 微信重试 | 同一 mp openid 关注 | `(app_key, openid)` UNIQUE + pending unionid PK | 再处理=upsert/忽略 | n/a |
| POST MP callback unsubscribe | 可选标记；本期不删 openid | 重试 | 同一 openid | 同上 | 保持别名 | n/a |
| GET follow-status | 可能消费 pending | 连点 | 该 user 的 unionid pending | unionid | 已绑定则 no-op | 同步锁 createClickGuard |
| POST apply | 写申请行 | 双击 | 该用户一笔 pending/active | 既有申请状态 + Idempotency-Key | 已 pending 则 400 | 已有 |

资金路径：申请本身不打款；绑定 openid 供后续分账，幂等 ≥ L3（upsert 唯一键）。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L3 | 验签、不建号、不 IDOR |
| 数据一致性 | L3 | unionid 锁定；冲突事件 |
| 可用性 | L2 | 回调 5s 内 `success`；user/info 失败则 pending 仅 openid 并记 warn |
| 可观测性 | L2 | 结构化日志 event=`wechat_mp_subscribe` + trace_id |
| 可伸缩性 | L1 | 见上表 L0/L1 |
| 容错 | L2 | 微信重试安全 |

## 领域模型影响

- 聚合：WechatIdentity（已有）+ MpSubscribePending（新，unionid 为身份）。
- 关注不创建 AuthUser。
- 事件键 = mp openid，禁止 user_id 作消费幂等键（本增量无消费者，仍按契约写）。
