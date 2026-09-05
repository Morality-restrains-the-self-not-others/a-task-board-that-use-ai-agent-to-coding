# NFR — 订单号租户基因

- **Date:** 2026-08-19
- **价值流:** `docs/superpowers/plans/2026-08-19-order-id-tenant-shard-value-stream.md`
- **Design:** `docs/superpowers/specs/2026-08-19-order-id-tenant-shard-gene-design.md`

资金域默认 L3；本增量不改支付入账，只改号码与查询谓词。

## 路径分片键审视

| 路径 | 携带分片 ID？ | 是否合适 | 可伸缩性 | 动作 |
|------|---------------|----------|----------|------|
| `POST/GET /api/tenant/{tid}/billing/orders/` | `tid` | 是（tenant） | L3 对齐租户分片 | 保持 |
| `GET/POST .../orders/{oid}/` 及 pay/cancel | `tid` + `oid` | `tid` 为分片键；`oid` 实体键 | L3 | SQL 必须含 `tenant_id` |
| 前端 `/tenant/{tid}/billing/orders/{oid}/` | `tid` | 是 | L3 | 不变 |
| 管理端 `/api/system_admin/orders/` | 无 tenant | 不适合当租户分片键 | **L0**：超管低频全表；升级触发=task_bill 水平分片 | 分片后 locator 或管理连接 |
| 粘贴四段 ORD 号（内部） | 解析出 `tid` | 是 | L2 | 解析后当租户路径 |
| 微信/PayPal 回调 | pending.`tenant_id` | 是 | L3 | 不改 `out_trade_no` |
| Kafka | 本增量无新消息键 | — | L0 | 无新事件 |

## 幂等性审视

| 路径 | 副作用 | 等级 | 重复边界 / 键 | 重放 |
|------|--------|------|----------------|------|
| 生成 order_number | 随 INSERT 一次 | L1 自然 | `(id)` UNIQUE + `order_number` UNIQUE | 1062 换新 Snowflake |
| `GET` 订单 / Parse | 无 | L0 | — | 只读 |
| `loadOrder` 复合查询 | 无 | L0 | — | 只读 |
| 下单/支付/退款 | 既有路径，本增量不改状态机 | 维持既有 ≥L3 | 订单 `id` / pending `out_trade_no` | 不在本增量重做 |

## 其它类别

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L3 | 基因不可当授权；错租户 404 |
| 性能 | L2 | PK + tenant 谓词；号码纯函数无 MAX |
| 可观测性 | L2 | 撞号已有 `resource_order_number_collision`；基因不匹配走不存在 |
| 可用性 | L2 | 存量三节号仍可展示 |

## 领域模型影响

- `OrderNumber` VO 含 tenant 拷贝；聚合根 PK 仍是订单 Snowflake。
- 仓储加载端口：`Load(tenantID, orderID)`；管理端另 `LoadByID`。
