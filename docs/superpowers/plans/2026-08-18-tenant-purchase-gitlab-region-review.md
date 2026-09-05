# Review — 租户购买 GitLab 须按行选区

- **日期**: 2026-08-18
- **计划**: `docs/superpowers/plans/2026-08-18-tenant-purchase-gitlab-region-plan.md`

## CRG

`code-review-graph update --brief` 增量无新节点。手工链：`handleCreateOrder` → `createOrder` → `markOrderPaid`；FE OrderCreate / OrderDetail / WorkspaceSettingsGitlabConnection。直购 `/purchase/` 已强制 region，未改。

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | GitLab 行 `getGitlabRegionBySlug`；磁盘/流量可不同 slug |
| Readability | 选区贴在对应资源卡片，对齐赠送页 |
| Architecture | 不升版；补齐 v85 I4 |
| Security | slug 白名单；tenant_admin；设置页真实 href |
| Performance | 下单多 1～2 次 slug 查询 |

## 安全审计

- [x] region 边界校验
- [x] 无新公开写接口
- [x] 日志无 token
- [x] 购买入口非 `@click.prevent`

## Intent → Event

书面例外：下单不发 Kafka；支付仍走既有 BILLING_TRANSACTION_CREATED。见 B-049b。

## Log Audit

- 成功：`resource_order_created` 含 `regions`
- 失败：`resource_order_create_failed` warn

## 严重度

- Critical: 0
- Required: 0
- Nit: 账单首页仍聚合单区 → OPT-20260818-022

## 测试

- `go test ./src -run 'CreateOrder_Gitlab'` 通过
- vitest OrderCreate 4 / OrderDetail 5 / WorkspaceSettings 6 通过
