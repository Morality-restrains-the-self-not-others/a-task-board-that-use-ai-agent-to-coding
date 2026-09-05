# GitLab 磁盘价格套餐 — 实施计划

## Task 1: 迁移与种子
- [x] `taskBill/migrations/005_gitlab_disk_pricing.sql`：套餐列、账户锁价列、billing_unit 种子
- [x] 验证 migrate 可重复执行（IF NOT EXISTS / OR IGNORE）

## Task 2: Go 领域与 API（TDD）
- [x] 扩展 `PricingPackage` / `BillingAccount`
- [x] 全路径 SQL + `createPricingPackage` 参数 + JSON
- [x] `pricingPackageUserDescription` 增加一行
- [x] `syncBillingUnitRows` 或独立 sync 更新 `gitlab_disk` 价
- [x] 测试：创建默认 1、显式值、描述、换套餐锁价

## Task 3: Django bridge + views
- [x] `PricingPackageRow` + create body
- [x] `pricing_views` GET/POST/public
- [x] 测试断言新字段

## Task 4: 前端
- [x] `SystemAdminPriceManagement.vue` 表单/表列
- [x] `pricingPackageDisplay.js` helper
- [x] `BillingDashboard.vue` / `SwitchPricingPackageModal.vue` / `Pricing.vue`

## Task 5: OpenAPI + intents + 价值流索引
- [x] intents / 设计文档（task2app/docs）
- [x] 价值流测试点见 value-stream 计划文

## Task 6: 验证
- [x] `go test` taskBill
- [x] Django 相关 pytest
- [x] SPA build + collectstatic
