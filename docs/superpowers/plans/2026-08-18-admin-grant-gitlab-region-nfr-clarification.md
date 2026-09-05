# NFR 澄清 — 管理端赠送 GitLab 须选区域

- **日期**: 2026-08-18
- **价值流**: `docs/superpowers/plans/2026-08-18-admin-grant-gitlab-region-value-stream.md`
- **默认等级**: L2（配额写路径资金相邻 → 幂等/一致性按 L3 审视）

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| `POST /api/tenant/{tid}/billing/accounts/admin_grant_points/` | `tid` = tenant_id | 是（租户配额所有权） | L1：按租户分片即可 | 保持 URL 中 tenant |
| `POST /api/internal/taskbill/admin-grant-resources/` | body `tenant_id` | 是 | L1 | 保持 |
| `GET /api/system-admin/gitlab-regions/` | 无租户 ID | 配置级全局目录 | L0：区域数十条，全表可接受 | 升级触发：区域数 > 1 万再分页 |
| 前端 `/system-admin/grant-points/` | 无 | 管理后台单页 | L0 | — |

`region` 是二级隔离键（与表 PK 对齐），不是跨租户分片键；查询始终带 `tenant_id`。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| admin_grant_points POST | 加配额 + grant 行 + 订单 | 双击提交、网关重试 | 一次管理员确认 = 一次赠送 | 可选 `idempotency_key`（已有） | 相同 key 返回已有结果，不加配额 |
| 内部 admin-grant-resources | 同上 | Kafka/礼包重放 | `new_user_gift:{companyID}` 已有 | 同上 | 礼包仍为 task_post |
| GET gitlab-regions | 无 | — | — | L0 | — |

禁止用 `tenant_id` 单独作幂等键。未传 `idempotency_key` 时两次提交允许两次赠送（现行为，不改）。资金路径默认 L3：本接口金额为 0，但配额累加，键粒度须为「这一次赠送」而非租户。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L1 | 按 tenant_id |
| 数据一致性 | L3 | 单事务：配额 + grant + 订单 |
| 安全 | L2 | 仅系统管理员；region 白名单 |
| 可用性 | L2 | 失败 400 可重试 |
| 可观测性 | L2 | 结构化日志含 tenant_id / region / resource_type（禁止 token） |

## 质量场景

1. 刺激：赠送 `gitlab_disk` 且 body 无 region。响应：400 `region required`，默认区 `disk_gb` 不变。
2. 刺激：同一 `idempotency_key` 重放。响应：`idempotent=true`，配额不双计。
3. 刺激：选 `tencent-sh-1` 赠送 1GB。响应：仅该 region 行增加。

## 领域模型影响

`ResourceGrant` 增加值对象 `GitlabRegionSlug`（GitLab 类型必填）。聚合边界仍为一次赠送事务（租户账户 + 该区域资源行）。
