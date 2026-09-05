# 实施计划 — 用户列表所属租户公司列

- **日期**: 2026-08-23

## 切片

- [x] **T1** taskAuth：`groupActiveTenantCompanies` + HTTP client 单测（Red/Green）
- [x] **T2** 列表 handler 回填 `tenant_companies`；将 list 从超 500 行文件抽出
- [x] **T3** FE：表头 + `UserListRow` 单元格 + 单测
- [ ] **T4** 跑测、精准重启登记、提交子仓+meta、SPA build、push

## 事件契约

无。publish/consumer 任务不适用。
