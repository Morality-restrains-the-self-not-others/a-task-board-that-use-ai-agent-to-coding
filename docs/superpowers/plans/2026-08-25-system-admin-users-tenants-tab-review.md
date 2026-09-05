# Review — 超管用户页租户 Tab

- **日期**: 2026-08-25
- **对照计划**: `docs/superpowers/plans/2026-08-25-system-admin-users-tenants-tab-plan.md`

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | Tab/深链/鉴权/分页/空态/错误 trace 有测。tenant-options 契约未改。 |
| Readability | handler 与 options 同构；Tab 抽到独立组件。 |
| Architecture | 查询留在 taskTenantService；无跨库。无新聚合/事件（纯查询例外已文档化）。 |
| Security | 平台员工门禁；日志无 search 原文/邮箱/手机；401/403 warn。 |
| Performance | 默认 50 条；搜索合并上限 500。 |

## 安全审计

- [x] 无密钥在代码/日志
- [x] 边界鉴权 401/403
- [x] SQL 参数化
- [x] 错误 DOM data-traceId
- [x] 只读 GET，无 CSRF 写
- [x] 无 SSRF 新出站（复用既有 Auth 内部 URL）

## Log Audit

- info：`admin_tenants listed count/total/limit/offset/search_len`
- warn：未鉴权 / 非平台员工
- error：db error（不带 SQL 原文）
- 禁止 PII

## Intent→Event

纯查询例外，对照表已写「无对应事件」。

## CRG

Step 0 `code-review-graph update --brief` 成功（7 files, risk 0）。本增量新符号 `handleAdminTenants` 仅由 mountRoutes 挂载。

## Simplify & Harden

- 未保留旧 Tab 双路径。
- 搜索分页走内存切片仅在 `search != ""`；空搜索 SQL OFFSET。
- 认证拒绝补 warn。

## 发现

- Required（已修）：`searchCompaniesPage` 曾把 `limit>200` 折成 80，搜索目录无法凑满设计上限 500。已改为 `limit<=0 → 80`、`limit>500 → 500`；补 `TestAdminTenantsSearchDoesNotCapAtEighty`。
- Nit：行内「进入租户」未做 → OPT-20260825-026
- Nit：公网硬刷新需 SPA build + 精准重启后验收 → OPT-20260825-027

Critical：无。
