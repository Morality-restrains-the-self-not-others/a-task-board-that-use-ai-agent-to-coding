# 功能意图：管理端租户下拉返回创建者邮箱与手机号

## 背景与目标

系统管理员在「赠送资源」「订单记录」页通过 `GET /api/system-admin/accounts/admin/tenant-options/` 选择租户。现网大量公司名为「我的公司」，下拉仅显示名称与 ID，无法区分账号。Django 时代曾用 `batch_resolve` 附带创建者联系方式；迁 Go 后 `companyToJSON` 只返回 `id/name/creator_id/created_at/updated_at`，前端已渲染 `phone`/`email` 但始终为空。

目标：选项 JSON 带创建者 `email`、`phone`（缺省空串）；`?search=` 除公司名外还可按公司 ID、创建者邮箱/手机号命中。

## 范围与边界

- 范围内：`handleAdminTenantOptions` 富化；`POST /api/internal/users/batch/details/` 增补 `phone`（向后兼容）；按联系方式搜创建者再并入公司列表。
- 范围外：新 HTTP 路径、租户成员全量联系人、对非平台员工开放、Kafka 事件。

## 约束与风险

- 单库所有权：邮箱/手机在 taskAuth，taskTenantService 不得直连 auth 库。
- 内部调用 fail-open：taskAuth 失败仍返回公司列表，联系字段为空，打 warn 日志（禁止记录明文邮箱/手机）。
- 仅平台角色（`super_admin` / `employee`）可访问。

## 验收标准

- 有创建者邮箱/手机时，选项含对应字段；缺失为空串，不返回「无」。
- `search` 命中公司名、公司 ID、创建者邮箱或手机号。
- 未鉴权 401，非平台员工 403。
- taskAuth 超时/5xx 不导致 500。

## 实施计划

1. batch/details 补 `phone`。
2. tenant-options 批量查创建者并合并搜索结果。
3. 单测覆盖富化、搜索、鉴权、fail-open。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 管理员查询租户选项（含联系方式） | — | — | `handleAdminTenantOptions` 只读 GET | 无 | 纯查询，无状态变更；无对应事件 |
