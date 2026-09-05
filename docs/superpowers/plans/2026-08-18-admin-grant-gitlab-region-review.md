# Review — 管理端赠送 GitLab 须选区域

- **日期**: 2026-08-18
- **计划**: `docs/superpowers/plans/2026-08-18-admin-grant-gitlab-region-plan.md`

## CRG

`code-review-graph impact --files taskBill/src/admin_grant.go taskBill/src/handlers_admin_grant.go`：图未含未提交符号（0 nodes）。手工调用链：`handleAdminGrantResources` / `handleInternalAdminGrantResources` / `referral_commission`（仅 task_post）。无遗漏 GitLab 赠送写入点。

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | GitLab 类型强制 `getGitlabRegionBySlug`；配额与订单行按 slug 写入；task_post 不变 |
| Readability | Region 字段与购买路径 `order_payment` 对齐 |
| Architecture | 无新服务；补齐 v85 遗漏路径 |
| Security | slug 白名单（启用区）；参数化 SQL；仅系统管理员既有入口；日志无 token |
| Performance | 单事务；区域列表全局 L0 |

## 安全审计

- [x] 无密钥入仓/日志
- [x] region 在边界校验
- [x] SQL 参数化
- [x] 无新公开写接口
- [x] 错误不暴露内部栈

## Intent → Event

书面例外：既有赠送无 Kafka；本增量只补属性。见 `docs/intents/backend/admin_grant_gitlab_region.intent.md`。

## Simplify & Harden

- 未抽取过早抽象（handler 两处解析各 6 行）
- 未改购买限购 1GB（管理赠送可超限，有意）
- Nit → OPT：赠送后未自动 `ensureTenantGitlabGroupForRegion`

## 严重度

- Critical: 0
- Required: 0（已修 toast 展示 region）
- Nit: 见 OPT

## 测试

- `taskBill` `go test ./src -run 'AdminGrantGitlab|AdminGrantTaskPost|HandleAdminGrantResourcesParsesRegion'` 通过
- `taskFE` vitest `SystemAdminGrantPoints.region.unit.test.js` 4/4 通过
