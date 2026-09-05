# NFR 澄清：25 天分账兜底

支撑等级：资金路径 L3。无新 HTTP 对外路径。

## 路径分片键审视

| 路径 | 是否携带分片 ID | 该 ID 是否合适 | 可伸缩性 | 动作 |
|------|-----------------|----------------|----------|------|
| `POST /api/internal/taskbill/profit-sharing/process-pending/` | 否（内部一次性扫描） | n/a | L0 | 小时级、LIMIT 50；升级触发：单次扫描 p95>2s 或 pending/failed 积压>1万 |
| 兜底 SQL `paid_at` 窗口 + status | 业务键 `order_id` | 订单级，适合资金幂等，不适合租户分片 | L0 | 全库时间窗扫描；升级：按 `paid_at` 分区裁剪或按 `tenant_id` 分片后二次扫 |
| 出站微信 CreateOrder | `out_profit_sharing_no` | 是（商户分账单号） | L3 | 与微信幂等同粒度 |

Hard Gate：通过（内部扫描标 L0 有理由与升级触发；资金出站 L3）。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 等级 |
|------|--------|------------|--------------|--------|----------|------|
| timer → process-pending | 可能 CreateOrder | 小时 tick、人工重放 POST、进程重试 | 一笔订单一次成功分账 | `out_profit_sharing_no`（= `order_id` 行唯一） | 已 `finished` 或已有微信单号 → skip；`failed` 重试用**同一**键 | L3 |
| 无行补 `markOrderForProfitSharing` | INSERT 一行 | 同上 | 一单最多一行 | `order_id` UNIQUE/COUNT 门闩 | 已有行 → 不再 INSERT | L3 |
| 无 openid skip | 无出站写 | tick 重放 | — | — | 每次 skip，不标 failed | L0 |

禁止用 `tenant_id` / `user_id` 作幂等键。前端无新按钮（Anti-Replay-OK: 无 UI 写操作）。

Hard Gate：通过。

## 其他

- 安全：内部密钥；不新增 PII 出站字段。
- 窗口：`paid_at > now-30d`，过期不打款（OPT 跟踪告警）。
- 无新 MQ。
