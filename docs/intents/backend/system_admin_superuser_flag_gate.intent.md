# 功能意图：仅系统管理员可设置 is_superuser

- **日期**: 2026-08-23
- **状态**: 实施中
- **接口**: `POST /api/system-admin/users/create/`、`PUT|PATCH /api/system-admin/users/{id}/`

## 背景与目标

用户编辑/添加表单中的「超级用户」复选框不得由平台员工或仅持有遗留 `is_superuser` 标志、但无 `super_admin` 角色行的账号改写。系统管理员角色（`super_admin` / `platform:manage`）仍可授予或撤销。

## 范围与边界

- 范围内：create/patch 请求体含 `is_superuser` 时的授权闸门。
- 范围外：内部密钥 `handlePatchUser`、邀请码升超管、RBAC 角色指派 API。

## 验收标准

1. 操作者 RBAC 表无 `super_admin` 且无 `platform:manage` 时，请求体含 `is_superuser` → 403，目标用户标志不变。
2. 操作者有 `super_admin` 时，可创建或 PATCH `is_superuser`。
3. 无 `is_superuser` 字段时，员工/遗留超管标志账号仍可改 `is_staff` 等其它白名单字段。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|----------|--------|---------|
| 拒绝或允许改超级用户标志 | （无新事件） | 闸门加在既有 create/patch 上，不新增业务成功路径 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-23 | 初稿 |
