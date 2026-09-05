# NFR 澄清 — 超管微信分账与动账通知

日期：2026-08-25  
价值流：`docs/superpowers/plans/2026-08-25-admin-profit-sharing-wechat-ids-and-change-notify-value-stream.md`

资金路径默认 **L3**。

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | L | 动作 |
|------|---------|--------|---|------|
| GET `/api/system-admin/profit-sharing/` | 无租户键（超管全局队列）；可选 `referrer_user_id` | 超管跨租户是需求 | L0 | 理由：平台员工全局只读，limit≤50；升级触发：单次 >500 行需按 tenant_id 强制过滤 |
| POST `/api/system-admin/profit-sharing/{id}/share/` | `id`（分账记录 PK） | 是（行级） | L1 | 按 id 定位单行；不扫全表 |
| POST `/api/billing/profitsharing/change-notify/` | `out_order_no` / `notify.id` | 是 | L1 | 按商户分账单号更新；inbox 以 notify.id 去重 |
| FE `/system-admin/users/` | 无 | 管理壳全局 | L0 | 已有 staff 门禁 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| GET list | 无 | — | — | L0 | 只读 |
| POST share | 出站 CreateOrder + 审计行 | 双击、网关重试 | 同一分账记录一次成功打款 | `Idempotency-Key` UNIQUE；processing/finished 空操作 | 同键 200 idempotent；不同键且已 finished 200 空操作 |
| POST change-notify | 更新台账 status | 微信重试 | 同一微信通知 ID | `notify.id` PK | 已处理直接 SUCCESS |
| 推荐人 share（未改） | 已有 | — | `out_profit_sharing_no` | 保持 |

前端按钮：同步门闩 + 同一次意图同一 Idempotency-Key。

## 类别支撑程度

| 类别 | 级别 | 说明 |
|------|------|------|
| 安全性 | L3 | 验签解密；staff 门禁；reason 必填 |
| 数据一致性 | L3 | 行锁 + notify inbox；不重复 CreateOrder |
| 可观测性 | L2 | slog 事件名稳定；禁止密钥/openid |
| 可伸缩性 | L0/L1 | 见路径表 |
| 容错 | L2 | 微信重试直至 SUCCESS；出站失败 502 + 本地 failed |

## 质量场景

1. 微信连续 POST 同一 `id` 3 次 → 仅第一次更新台账，三次均 `{code:SUCCESS}`。
2. Staff 对同一 pending 行用同一 Idempotency-Key 点两次 → 只一次 CreateOrder。
3. 缺 reason → 400，零出站。
