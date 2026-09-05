# Step 7 — 实施计划

## Slice A — 设置表 + Go API（TDD）
- [ ] `005_marketplace_settings.sql`
- [ ] DB Get/Set helpers
- [ ] GET marketplace-settings / GET|PATCH admin-marketplace-settings
- [ ] vendor-status 嵌入字段
- [ ] UpsertVendorFromBridge 条件建档
- [ ] Go 测试全绿

## Slice B — SystemAdmin UI
- [ ] SystemAdminContainerImages：SSO + 开关
- [ ] Sidebar 移除 SSO 链
- [ ] 单元测试

## Slice C — ImageMarket
- [ ] 按 review_enabled 切换按钮
- [ ] 更新 ImageMarket.vendorStatus.test.js

## Slice D — 迁移与重启登记
- [ ] 9999 / migrate 说明
- [ ] precise_restart：taskAiProvider taskFE
