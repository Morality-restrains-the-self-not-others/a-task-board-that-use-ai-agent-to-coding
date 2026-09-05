# 实施计划 — 账单页 GitLab 按区用量

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-billing-gitlab-quota-region-usage-design.md`

## 事件任务（书面例外）

无 publish / consumer。意图文档记录「纯展示」。

## 任务清单

- [ ] **T1** 意图文档 `docs/intents/frontend/billing_gitlab_quota_region_usage.intent.md` + test-intent；INDEX F-100
- [ ] **T2** Red：扩展 `BillingDashboard.gitlabRegions.test.js` — 按区卡显示 `已用 / 配额`、区域名；有列表时不出现无区域汇总「GitLab 磁盘」大卡（或汇总卡不在 `gitlab-aggregate-*`）
- [ ] **T3** Red：`gitlabQuotaSplit` 测试仍能看到赠送/购买
- [ ] **T4** Red：无 `gitlab_resources` 时回退卡显示 used/quota
- [ ] **T5** Green：`GitlabRegionQuotaCards.vue` + 改 `BillingDashboard.vue` 映射 `diskUsedGb`/`trafficUsedGb`
- [ ] **T6** `formatUsedGb` 抽到 `utils/formatUsedGb.js`（设置页复用，口径一致）+ 单测
- [ ] **T7** Go：`TestHandleResourceQuotasMultiRegion` 断言 `disk_used_gb`/`traffic_used_gb` 存在
- [ ] **T8** 跑 vitest 相关文件 + Go 该测例；`gofmt`/`py_compile` 不适用处用对应校验
- [ ] **T9** 登记精准重启 `taskFE`；公网 SPA `npm run build`（taskFE）
