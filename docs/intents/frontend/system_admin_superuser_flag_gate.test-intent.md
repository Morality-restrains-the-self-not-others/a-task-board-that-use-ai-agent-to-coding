# 测试意图：编辑用户「超级用户」仅系统管理员可改

- **日期**: 2026-08-23
- **对应功能意图**: `system_admin_superuser_flag_gate.intent.md`

| 编号 | 场景 | 期望 |
|------|------|------|
| T1 | `canSet=false` 时 `omitUnauthorizedSuperuser` | 去掉 `is_superuser` |
| T2 | `canSet=true` 时保留 `is_superuser` | 字段仍在 |
| T3 | 非 `super_admin` 打开编辑模态 | `#edit-is-superuser` disabled |
| T4 | `super_admin` 打开编辑模态 | `#edit-is-superuser` 可勾选 |
