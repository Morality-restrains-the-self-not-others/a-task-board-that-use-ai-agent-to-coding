# NFR：推荐人渠道聚合分账

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| GET `/api/billing/profit-sharing/referrer-orders/` | `referrer_user_id`（X-User-Id） | 合适 | 等值过滤，禁止全表扫描 |
| POST `/api/billing/profit-sharing/referrer-orders/share-channel/` | `referrer_user_id` + `channel_code` | 合适：渠道是推荐人自己的投放键，不是受推荐人 ID | 查询必须带 referrer 等值 |
| POST `.../{id}/share/` | `referrer_user_id` + 行 id | 合适（存量） | 他人 404 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| POST share-channel | 该渠道每笔 shareable 出站微信 CreateOrder | 双击、刷新、网关重试 | 每条 `billing_profit_sharing` 成功一次 | 行 `out_profit_sharing_no`；前端 `Idempotency-Key` 同一次意图不变 | processing/finished 跳过；pending/failed 在窗口内执行 |
| GET 列表 | 无 | — | — | L0 | — |

资金路径 **L3**。禁止用 `tenant_id`/`user_id`（受推荐人）作幂等键。
