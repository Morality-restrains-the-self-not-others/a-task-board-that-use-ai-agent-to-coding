# 设计：系统管理租户列表名称链到租户详情

- **日期**: 2026-08-25
- **页面**: `/system-admin/users/?tab=tenants` → `/system-admin/tenants/:tenantId/`
- **状态**: accepted（goal-mode 自动采用）
- **架构变更**: 是（v109）— 无新服务；新增平台员工只读接口与 SPA 路由

## Context

超管在用户页「租户」Tab 看到公司名称（如「我的公司」）仅为纯文本，无法进入租户详情。运营需要一眼看到该租户的**当前剩余资源**、**工作空间情况**、**订单情况**。

既有能力：

- 租户目录：`GET /api/system-admin/accounts/admin/tenants/`（taskTenantService）
- 配额：`GET /api/tenant/{id}/billing/quotas/`（taskBill）**无平台员工鉴权**，且网关租户 PDP 可能拒绝非成员
- 工作空间：`GET /api/projects/workspaces/tenant_id/{tid}` 按 JWT 当前租户 + 成员裁剪，超管看任意租户会得到空/错数据
- 订单：`GET /api/system-admin/orders/` 跨租户；单租户 FE 现走 `/api/tenant/{id}/billing/orders/`（同样依赖租户成员 PDP）

## Decision

1. **名称改为真实链接**（禁止 `@click` + `router.push`）：
   ```html
   <a :href="`/system-admin/tenants/${row.id}/`">{{ row.name }}</a>
   ```
   `Anti-Replay-OK: real href navigation`。无名称时仍显示 `—`（不可点）。
2. **新 SPA 页** `/system-admin/tenants/:tenantId/`（`adminRoutes`），系统管理壳（Navbar + SystemAdminSidebar）。面包屑：`<a href="/system-admin/users/?tab=tenants">租户列表</a>`。
3. **详情页三块**（并行 GET，互不阻塞；各块独立错误 `data-traceId`）：
   - **剩余资源**：任务帖配额（含赠送/购买拆分）+ GitLab 磁盘/流量；多区域时复用 `GitlabRegionQuotaCards`。
   - **工作空间**：id / 名称 / 默认 / 创建时间；分页；空态「暂无工作空间」。
   - **订单**：最近订单（号、状态、金额、创建时间）；「查看全部」链到 `/system-admin/order-records/?tenant_id=`。
4. **新/扩只读 API**（均 `X-Gateway-Auth-Verified=1` + `authz.IsPlatformStaff`）：

| 方法 | 路径 | Owner | 说明 |
|------|------|-------|------|
| GET | `/api/system-admin/accounts/admin/tenants/{id}/` | taskTenantService | 单租户头（name/creator/email/phone/created_at）；404 不存在 |
| GET | `/api/system-admin/tenant-quotas/tenant_id/{id}/` | taskBill | 与租户 quotas 响应同构；复用 `handleResourceQuotas` 数据装配 |
| GET | `/api/system-admin/tenant-workspaces/tenant_id/{id}/` | taskProjectService | 该租户全部工作空间（**不做 mine 裁剪**）；`limit`/`offset` |
| GET | `/api/system-admin/orders/?tenant_id={id}` | taskBill | **扩展**既有列表：有 `tenant_id` 时走 `listTenantOrders` |

5. **网关**登记 tenant-quotas / tenant-workspaces URI → 对应 upstream；tenants `{id}` 已由既有 `api-system-admin-tenants` 的 `tenants/*` 覆盖。
6. **不做**：模拟登录进租户控制台、编辑公司、成员管理、购买资源、写操作。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 抽屉/弹层不换页 | 三块数据 + 分页需要独立 URL，深链与刷新会丢 |
| 复用 `/api/tenant/{id}/billing/quotas/` 与 workspace 成员 API | 超管通常不是目标租户成员；workspace 列表用 JWT 租户而非 URL 租户 |
| taskTenantService BFF 聚合三域 | 违反单服务数据所有权；改为 FE 并行调 owner |
| 独立 Django 接口 | 元规则 20：新接口落 Go |

## 契约

### GET `/api/system-admin/accounts/admin/tenants/{id}/`

```
200 { id, name, creator_id, email, phone, created_at, updated_at }
401 未鉴权 / 403 非平台员工 / 404 租户不存在
```

### GET `/api/system-admin/tenant-quotas/tenant_id/{id}/`

与 `GET /api/tenant/{id}/billing/quotas/` 字段一致（`task_post_quota*`、`gitlab_resources`、聚合磁盘/流量）。401/403 同上。租户无账户时返回零值配额（与现租户 quotas 一致），不 404。

### GET `/api/system-admin/tenant-workspaces/tenant_id/{id}/?limit=50&offset=0`

```
200 { items: [{ id, name, is_default, created_at }], total, limit, offset }
401 / 403
```

租户无工作空间：`items=[] total=0`（不 404；公司是否存在由 header API 判定）。

### GET `/api/system-admin/orders/?tenant_id={id}&limit=&offset=&status=`

既有 `orders/total/limit/offset`；`tenant_id` 非法 → 400。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|----------|--------|--------|--------|----------|
| 超管打开租户详情 | — | — | — | 纯 GET 只读，无系统事实变更 |
| 超管浏览配额/工作空间/订单 | — | — | — | 同上 |

## 🐍 Python 新增接口清单与 Go 替代评估

**not_applicable** — 全部新接口落 Go（taskTenantService / taskBill / taskProjectService）。无 Python endpoint。

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief` 成功（5 files / 0 nodes）。无 `codegraph_explore` MCP。基线来自源码：

- `SystemAdminTenantsPanel.vue` 名称列为纯文本
- `handleAdminTenants` 仅列表
- `handleResourceQuotas` 无 staff 闸
- `handleListWorkspaces` 用 `getAuthTenant` + mine
- `doAdminListOrders` 无 `tenant_id` 过滤（单租户 FE 走租户 path）

## Value Stream Impact

影响 `system-admin-user-management`（租户 Tab 下游）。新增流步骤 `admin-tenant-detail`（planned→本迭代 active）。无新表字段；读既有 `tenant_company` / `billing_*` / `project_workspace_entries`。

## 🏛️ 架构变更影响

- **迭代版本**: v109 🎯 target
- **迭代名称**: system-admin-tenant-detail
- **作者**: cursor
- **设计日期**: 2026-08-25 19:30
- **新增文件**（每个视图四类伴生格式）:
  - `docs/architecture/v109-application-integration-20260825-1930-cursor.puml`
  - `docs/architecture/v109-enterprise-landscape-20260825-1930-cursor.puml`
  - 伴生 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskFE — 租户名链接 + 详情页
  - 🟡 [MODIFIED] taskGateway — 新 URI
  - 🟡 [MODIFIED] taskTenantService — GET by id
  - 🟡 [MODIFIED] taskBill — staff quotas + orders `tenant_id`
  - 🟡 [MODIFIED] taskProjectService — staff workspace list
  - 🟢 [NEW] Application Interface：上述三条 GET

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v108 → Gap（名称不可点、超管无法看任意租户配额/工作空间/订单）→ WP → Plateau v109；目标拓扑：FE→GW→三 owner |
| **`.full.archimate`** | 同拓扑全量（本迭代为聚焦视图，与 v108 风格一致） |
