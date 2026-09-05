# 实施计划 — 用户列表是否获得分账资格列

- **日期**: 2026-08-23

## 切片

- [x] **T1** taskReferral：`lookupActiveQualificationsBatch` + `POST .../qualification/active/batch/` 单测（Red/Green）
- [x] **T2** taskAuth：`fetchQualificationsBatch` + 列表回填 `has_profit_sharing_qualification`；下游失败为 null
- [x] **T3** FE：表头 + `UserListRow` 单元格（是/否/—）+ 单测
- [x] **T4** 跑测、精准重启登记、提交子仓+meta

## 事件契约

无。publish/consumer 任务不适用（只读）。
