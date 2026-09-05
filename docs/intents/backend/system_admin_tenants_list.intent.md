# 功能意图：管理端租户目录分页列表

## 背景与目标

超管租户 Tab 需要可翻页的公司目录。现有 `tenant-options` 是下拉（数组、上限 80），不能承担目录。

目标：`GET /api/system-admin/accounts/admin/tenants/` 返回 `{items, total, limit, offset}`，联系方式语义与 tenant-options 一致。

## 范围与边界

- 范围内：分页、search（名/ID/创建者联系方式）、平台员工鉴权、Auth fail-open。
- 范围外：改 tenant-options 响应形状、写公司、成员列表。

## 约束与风险

- 单库所有权：公司在 taskTenantService；联系方式经内部 HTTP。
- 日志禁止 search 原文与明文邮箱/手机。
- 带搜索时合并结果上限 500。

## 验收标准

- 未鉴权 401，非平台员工 403。
- 空库 `items=[]` `total=0`。
- 分页 offset 不重复。
- 有创建者联系方式时 items 含 email/phone；缺失为空串。
- taskAuth 5xx 不导致 500。

## 实施计划

1. `countCompanies` + `searchCompaniesPage`。
2. `handleAdminTenants` + 路由双挂载。
3. 网关登记。
4. 单测。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 管理员查询租户目录 | — | — | `handleAdminTenants` 只读 GET | 无 | 纯查询，无状态变更；无对应事件 |
