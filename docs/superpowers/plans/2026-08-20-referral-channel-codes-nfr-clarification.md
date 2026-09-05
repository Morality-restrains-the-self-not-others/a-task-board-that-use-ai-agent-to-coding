# NFR 澄清 — 多渠道推荐码

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-referral-channel-codes-design.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| GET/POST `/api/referral/channels/` | 无 tenant；按 `X-User-Id` | 用户级小表（≤20 行/用户） | L0：每用户渠道极少；升级触发：单用户 >100 渠道或全表扫描 >100 万行时改为 `HASH(user_id)` |
| POST `/api/referral/channels/code/{code}/disable/` | code 全局 UNIQUE | code 适合点查，非租户分片 | L0 同上 |
| GET `/api/referral/stats/user_id/{userId}/` | 实际过滤 `referrer_user_id`=调用者 | 边表按 referred PK；推荐人查询用现有 idx_referrer | 保持；禁止用路径 userId 越权 |
| SPA `/profile/referral/` | 无 tenant | 个人中心 | L0 |
| POST `/api/internal/referral/bind-from-code/` | referred_user_id + access_code | 边 PK=referred | 保持 |
| POST `/api/internal/taskbill/referral/sync-edge/` | referred_user_id | 是（一被推荐人一边） | 保持 |
| 微信支付后 markOrderForProfitSharing | order_id + 买家 user_id | 订单实体键 + 边 PK | JOIN 必须用订单买家，禁止空 referred |

## 幂等性审视

| 路径 | 副作用 | 重复边界 | 幂等键 | 重放 | 等级 |
|------|--------|----------|--------|------|------|
| GET channels / GET stats | 无 | — | — | 只读 | L0 |
| POST `/api/referral/channels/` | 插入渠道码 | 同一用户同一渠道名至多 1 行 | `(user_id, channel_name)` UNIQUE；HTTP `Idempotency-Key` 复用同名结果 | 再 POST 返回已有渠道 200 | L2 |
| POST disable | status→disabled | 该 code 一次禁用 | code + owner | 再执行仍 disabled | L2 |
| bind-from-code / sync-edge | upsert 边 | 一被推荐人一边 | `referred_user_id` PK；不覆盖 channel/eligible | 再绑空操作 | L3 |
| accrue / 微信分账 | 写计提/分账 | 一笔消费一次佣金 | `source_txn_id` UNIQUE；且 `commission_eligible=1` | 再执行 ON DUPLICATE 空操作 | L3 |
| 退款 void | 作废计提 | 一订单一次 | `order:`+order_number | 再 void 已 voided | L3 |

## 质量场景

场景 ID: QS-01  
类别: 安全性  
等级: L3  
刺激：用户 A 请求 user B 的 stats 路径  
响应：只返回 A 自己的边与计提  
响应度量：单测断言 referral_count 不含 B

场景 ID: QS-02  
类别: 一致性/资金  
等级: L3  
刺激：无资格推荐人码注册后被推荐人消费  
响应：边存在且 commission_eligible=0；accrual 0 行；不分账  
响应度量：单测

场景 ID: QS-03  
类别: 幂等  
等级: L2  
刺激：同一渠道名 POST 两次（含同一 Idempotency-Key）  
响应：仅 1 行；第二次 200 同一 code  
响应度量：单测

## 领域模型影响

| NFR 决策 | 领域模型影响 | 具体动作 |
|----------|-------------|---------|
| 副作用路径资金 L3 | 边快照资格，计提看快照不看当前资格 | Edge.commission_eligible 不变式 |
| 用户路径 L0 | Channel 聚合按 user_id，不引入 tenant | 仓储 ListByOwner(userID) |
| POST 渠道幂等 | 渠道名是创建边界 | UNIQUE(user_id, channel_name) |
