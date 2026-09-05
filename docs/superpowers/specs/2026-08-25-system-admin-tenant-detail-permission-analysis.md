# 权限分析：系统管理租户详情

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-system-admin-tenant-detail-design.md`
- **状态**: accepted（goal-mode）

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| SPA `/system-admin/tenants/:id/` | 平台员工 | System | read | 系统管理壳（与 users 页同） | ✅ 充分 | 不新增租户 RBAC page 组（非租户控制台） |
| GET tenants/{id}/ | 平台员工 | Tenant 元数据 | read | 新增 `IsPlatformStaff` + 网关 verified | ✅ | 非员工 403；未知 id 404 |
| GET tenant-quotas | 平台员工 | Tenant 配额 | read | 新增 staff 闸 | ✅ | 禁止走租户 PDP 成员接口 |
| GET tenant-workspaces | 平台员工 | Workspace 列表 | read | 新增 staff 闸 + URL tenant_id | ✅ | 禁止 mine 裁剪 |
| GET orders?tenant_id= | 平台员工 | Order | read | 既有 staff + 新增 tenant 过滤 | ✅ | 非法 id 400 |

## 新增角色/权限建模

无新角色。沿用 `super_admin` / `employee`（`authz.IsPlatformStaff`）。不新增 `page:*` / `region:*` 种子（系统管理域，非租户控制台）。

## 安全检查结论

- [x] **IDOR**: 路径 tenant id 仅 staff 可读任意租户；非 staff 403。工作空间查询 `WHERE company_id=?` 绑定 URL id。
- [x] **权限提升**: 无写接口。
- [x] **跨租户泄露**: staff 本职即可跨租户只读；非 staff 不可。
- [x] **403 vs 404**: 非 staff 一律 403（不按 id 是否存在区分，避免探测）。staff + 未知公司 header → 404。
- [x] **user_id 注入**: 无。
- [x] **敏感操作**: 无删除/资金写；日志禁止 PII（联系方式已在目录接口存在，详情沿用）。

## 测试用例清单

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 平台员工读 header | super_admin | GET tenants/{id}/ | 200 |
| 普通用户读 header | 无平台角色 | GET tenants/{id}/ | 403 |
| 未鉴权 | — | GET | 401 |
| 员工读未知公司 | staff | GET tenants/nope/ | 404 |
| 员工读他租户配额 | staff | GET tenant-quotas | 200 |
| 员工读他租户工作空间 | staff | GET tenant-workspaces | 200 全量（含无 access 行） |
| 员工按 tenant_id 列订单 | staff | GET orders?tenant_id= | 仅该租户 |

## 风险评级

| 项 | 级别 | 缓解 |
|----|------|------|
| 平台员工可见任意租户配额/订单 | 中（产品意图） | 仅 staff；结构化日志 `admin_tenant_detail_*` 带 tenant_id + user_id |
| 配额接口无公司存在性检查 | 低 | header 404 由 FE 主导；配额零值与现租户 API 一致 |
