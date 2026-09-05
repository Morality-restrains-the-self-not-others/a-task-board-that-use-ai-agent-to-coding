# NFR 澄清：推荐资格服务号动态 scene 码

- **Date:** 2026-08-26
- **Default level:** L2；身份绑定 L3

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| POST `/api/auth/wechat/mp/follow-qr/` | `user_id`（会话） | 是，用户级票据 | L0 | 每用户复用未过期 pending 票 |
| GET `/api/auth/wechat/mp/follow-status/` | `user_id` | 是 | L0 | 单用户点查 |
| POST `/api/auth/wechat/mp/callback/` | scene `temp_id` / mp openid | temp_id 是票主键，适合点查 | L1 | 全站关注量远低于百万/年；升级触发：票表 >100 万行再按 expire_at 分区 |
| `WECHAT_MP_SUBSCRIBED` bound | mp openid | 是 | L1 | 禁止用 user_id 作消费键 |
| `WECHAT_MP_SUBSCRIBED` conflict | temp_id | 是，与冲突票同粒度 | L1 | |
| `WECHAT_IDENTITY_CONFLICT` | owner + openid | 是 | L1 | |

L0 理由（follow-qr/status）：平台个人身份，非租户膨胀；单用户一行热票。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 业务边界 | 幂等键 | 重放语义 | 前端防重放 |
|------|--------|----------|----------|--------|----------|------------|
| POST follow-qr | 写票 + 调微信创码 | 刷新/双击 | 该用户一张未过期 pending 票 | 用户 pending 票；Idempotency-Key 必填 | 复用票，不再打微信 | 同步锁 + 同一意图同一 key |
| GET follow-qr | n/a | — | — | L0 | — | — |
| POST callback subscribe/SCAN | 写 identity/票 + 事件 | 微信重试 | 同一 temp_id 或同一 mp openid | 票 PK + `(app_key,openid)` UNIQUE | 已 bound 则 no-op；conflict 保持 | n/a |
| GET follow-status | claim 无 scene pending；或用 pending temp_id 对账粉丝 qr_scene_str 后 bind/conflict | 连点 | 该用户 pending 票 / unionid pending | 票 PK + `(app_key,openid)` | 已绑定 no-op | createClickGuard |
| POST apply | 已有 | 双击 | 该用户申请 | 既有 + Idempotency-Key | 已有 | 已有 |

资金路径：绑定 openid 供后续分账，绑定幂等 ≥ L3。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L3 | 验签、不建号、不 IDOR、冲突不抢绑 |
| 数据一致性 | L3 | 票用户 vs unionId 占用 |
| 可用性 | L2 | 回调 5s 内 success；创码失败 503 |
| 可观测性 | L2 | ticket_status / conflict_code / 指纹 / trace_id |
| 可伸缩性 | L1 | 见上表 |
| 容错 | L2 | 微信重试安全；复用 pending 票 |

## 领域模型影响

- 聚合：FollowTicket（temp_id 根）+ WechatIdentity。
- 关注不创建 AuthUser。
