# NFR：推荐人手动分账

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| GET/POST `/api/billing/profit-sharing/referrer-orders/` | `referrer_user_id`（当前用户） | 合适：按推荐人列表，与数据行归属一致 | 查询必须带 referrer 等值，禁止扫全表 |
| 内部 process-pending 兜底 | 无租户键（平台级扫描） | L0：小时级、LIMIT 50 与现网一致；升级触发：待扫描积压 > 1k 行 | 维持现网 LIMIT；不在本增量改分片 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| POST share | 微信 CreateOrder + 行状态 | 双击、刷新、网关重试 | 一条 `billing_profit_sharing` 行一次成功打款 | `out_profit_sharing_no`；前端 `Idempotency-Key` 同一次意图不变 | finished → 200 空操作；processing → 409；失败可同单号重试 |
| process-pending 25d 兜底 | 同上 | timer 重入 | 同上 | `out_profit_sharing_no` | 与现网一致 |

资金路径 L3：唯一约束已在 `out_profit_sharing_no` UNIQUE。

## 其它

- 可用性：列表 p95 与现网管理端队列同级即可；微信超时按失败可重试。
- 安全：禁止回传 openid；禁止查他人行。
- 日志：授权拒绝 WARN；状态变更 INFO；微信调用 DEBUG/INFO 含 duration；禁止记录 openid 明文。
