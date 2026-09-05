# Review — 账单页 GitLab 按区用量

- **日期**: 2026-08-25
- **计划**: `docs/superpowers/plans/2026-08-25-billing-gitlab-quota-region-usage-plan.md`

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | 有 `gitlab_resources` 时按区卡显示 used/quota；无列表回退聚合卡也显示 used。Vitest 17 dashboard + 5 formatUsedGb + 14 settings；Go `TestHandleResourceQuotasMultiRegion` 绿 |
| Readability | 展示抽到 `GitlabRegionQuotaCards`；`formatUsedGb` 与设置页共用 |
| Architecture | 无新 API/表/事件；只消费既有 quotas JSON |
| Security | 只读；外链真实 href；无密钥；tenant 路径未扩大 |
| Performance | 无轮询、无 N+1；一次 GET |

## 安全审计

- [x] 无密钥入代码/日志
- [x] 无新用户输入写路径
- [x] 无 SQL
- [x] 输出为文本插值（Vue 默认转义）
- [x] 沿用 quotas 鉴权
- [x] 无新 CORS
- [x] 错误路径未改（fetch 失败仍 console.error，与原页一致）

## Log Audit

无新写路径。quotas GET 既有后端日志。前端失败仍走原 `console.error`（预存在，不在本次扩大）。

## Intent→Event

纯查询，意图文档已写例外。无 MQ。

## CRG

`code-review-graph update --brief` 于设计前已跑。爆炸半径限于 BillingDashboard 渲染 + quotas 列表字段断言。设置页仅复用 formatUsedGb。

## Simplify & Harden

Go 测例合并为单循环断言 region + used 字段。无死代码。used 缺失按 0 展示（合法未上报）。

## 发现

- Critical: 0
- Required: 0
- Nit: 回退聚合卡无进度条（区域数=0 的少见路径）；记 OPT-20260825-009
