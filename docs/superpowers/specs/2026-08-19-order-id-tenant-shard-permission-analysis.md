# 订单号租户基因 — 角色权限分析

- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-order-id-tenant-shard-gene-design.md`

无新角色、无新公网 endpoint、无新资源组。展示号第 3 段只作路由提示，**不能**代替鉴权。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `POST .../billing/orders/` 写入四段 order_number | 租户管理员 billing:manage | Tenant | write | 路径 tid + 账单权限 | ✅ | 号码内 tenant 必须等于 path tid |
| `GET .../orders/{id}/` | billing:view | Tenant | read | 现 IDOR：tid 与行归属 | ✅ | SQL `tenant_id AND id`；错租户 404 |
| `POST .../pay/` `.../cancel/` | 下单用户 / 管理员 | Tenant | write | path tid | ✅ | 同上复合查询 |
| 管理端粘贴 ORD 号 | superuser | System | read | 管理端鉴权 | ✅ | 解析基因后跳转租户 URL；禁止匿名公网按号查单 |
| `loadOrderByID` | 支付回调 / 超管 | System | read | pending 已存 tenant 或超管身份 | ✅ | 单库 PK；分片后改定位表。回调 `tenant_id=0` 须用行上 tenant 发配额 |
| 解析出的 tenantHint | 任意持有号码的人 | — | — | 行上 tenant 必须等于基因 | ✅ | 改中间段 → 不存在；不授权登录 |

**IDOR：** 租户面不得先按 id 命中再 403（可探测他租户订单存在）。复合查询无行即 404。
