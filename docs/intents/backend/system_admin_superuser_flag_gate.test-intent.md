# 测试意图：仅系统管理员可设置 is_superuser

- **日期**: 2026-08-23
- **对应功能意图**: `system_admin_superuser_flag_gate.intent.md`

| 编号 | 场景 | 期望 |
|------|------|------|
| T1 | 仅遗留 `is_superuser` 标志、RBAC 无 `super_admin` 的操作者 PATCH `is_superuser: true` | 403，目标仍非超管 |
| T2 | `super_admin` 操作者 PATCH `is_superuser: true` | 200，目标为超管 |
| T3 | 遗留标志操作者 PATCH 仅 `is_staff`（不含 `is_superuser`） | 200 |
| T4 | 遗留标志操作者 POST create 且 `is_superuser: true` | 403 |
