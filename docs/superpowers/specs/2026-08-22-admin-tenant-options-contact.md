# 管理端租户下拉展示创建者邮箱/手机号

- **Date:** 2026-08-22
- **Status:** accepted（/goal 自动采用）
- **架构变更:** 否（既有 GET 富化字段 + 既有 internal 批量查询增补 phone）

## Context

赠送资源页下拉只显示「我的公司」+ Snowflake ID。前端已按 `t.phone`/`t.email` 渲染，但 Go `handleAdminTenantOptions` 未富化。邮箱/手机属 taskAuth `auth_login_method`。

## Decision

1. `POST /api/internal/users/batch/details/` 响应增补 `phone`（缺省空串）。
2. `handleAdminTenantOptions` 收集 `creator_id`，一次内部 POST 批量取联系方式，写入选项 JSON。
3. `?search=`：公司名 LIKE **或** 公司 ID LIKE **或** taskAuth 用户搜索命中的创建者名下公司；去重后仍限 80。
4. taskAuth 失败 fail-open。
5. 权限不变：网关已验证 + `authz.IsPlatformStaff`。

拒绝方案：taskTenantService 直连 auth 库（违反单服务数据所有权）；N+1 GET 用户详情（已有 batch）；新建独立 API（前端契约已是 tenant-options）。

## Role / Permission

| 路径 | Who | 资源 | 动作 | 条件 |
|------|-----|------|------|------|
| GET tenant-options | super_admin / employee | 全平台公司列表 + 创建者联系方式 | 读 | `X-Gateway-Auth-Verified=1` 且平台角色 |
| POST batch/details | 内部服务 | 用户登录标识 | 读 | `X-TaskAuth-Internal-Secret` |

普通租户成员不可见。联系方式仅管理端展示，日志只记条数。

## Alternatives considered

- 只改前端二次请求用户列表：N 次调用、权限面扩大。
- 把邮箱冗余进 `tenant_company`：双写与一致性成本高。
