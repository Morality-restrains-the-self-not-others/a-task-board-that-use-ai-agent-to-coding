# 实施计划：用量页工作空间筛选

## 任务

- [x] taskBill `handleUsagesList` 支持 `workspace_id`
- [x] OpenAPI usages 参数同步
- [x] Go 单测：按 workspace 过滤分页
- [x] `useBillingUsage.js` + `BillingUsageFilters.vue` + `BillingUsage.vue`
- [x] Playwright：筛选触发带 `workspace_id`
- [x] intent 文档
- [x] `runall-lifecycle.sh build`（公网 SPA）
- [x] 线上验收：筛选「研发工作空间」后 total 88→77
