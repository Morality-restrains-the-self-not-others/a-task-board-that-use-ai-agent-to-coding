# 审查 — 用户列表所属租户公司列

- **日期**: 2026-08-23
- **对照计划**: `docs/superpowers/plans/2026-08-23-system-admin-users-tenant-company-column-plan.md`

## CRG

`code-review-graph update --brief` 已跑。`impact` 子命令参数与本环境用法不匹配（记录后继续）。codegraph explore 确认 `handleSystemAdminListUsers` 调用 `fetchReferrersBatch`；同类点已加 `fetchTenantCompaniesBatch`。调用方仅 `handleSystemAdminUsers` GET 列表。

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | 活跃成员回填、停用剔除、宕机空数组；Go/Vitest 覆盖 |
| Readability | 列表抽到 `handlers_system_admin_list.go`；client 与推荐人列同构 |
| Architecture | 不直连租户表；复用 `members/batch-get` |
| Security | 公开路径仍 `requireSuperuser`；内部 `X-Internal-Secret`；日志无密钥 |
| Performance | 每页一次 batch-get，超时 5s |

## Intent → Event

两份意图均为只读例外，无 MQ。符合元规则 49 的「无消费者则无幂等键」边界。

## 阻断项

无。
