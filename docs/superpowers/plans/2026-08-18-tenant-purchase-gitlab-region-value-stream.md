# 价值流 — 租户购买 GitLab 须按行选区

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-tenant-purchase-gitlab-region-design.md`

## Related Value Streams

- `2026-08-18-pluggable-multi-region-gitservice-value-stream.md` I4：租户选购必选 region（整单共享）。
- `2026-08-18-admin-grant-gitlab-region-value-stream.md`：管理端按行选区。

本流是 I4 的 **modification**：共享选区 → 磁盘/流量各自选区 + slug 校验 + 购买入口可达下单页。不撤销 VIP1 / 1GB 限购。

## 增量

| # | 增量 | 用户价值 |
|---|------|----------|
| I1 | 下单 API 校验启用 slug | 不能把配额写到幽灵区 |
| I2 | 购买页按行选区 | 磁盘可选「哪个区的盘」 |
| I3 | 订单详情展示 region | 支付前能确认区 |
| I4 | 设置页购买链到 create | 选区路径可走完 |

## Fields

- `billing_resource_order_item.region`
- `billing_tenant_gitlab_resource.(tenant_id, region)`
