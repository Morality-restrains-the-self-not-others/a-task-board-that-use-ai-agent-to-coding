# RBAC 身份角色权限体系 — 最终设计

- **状态**: 🎯 target
- **迭代**: rbac-identity-role-permission-system
- **作者**: claude
- **设计日期**: 2026-08-04
- **基于**: v57 current (Django 退役，纯 Go 微服务 + API 路径规范迁移已完成)

---

## 1. 现状分析

### 1.1 当前鉴权模型

```
taskAuth (auth.db)                    taskTenantService (tenant.db)
─────────────                         ─────────────
auth_user                             tenant_company
├── is_superuser (TINYINT)            ├── creator_id → 隐式 admin
├── is_staff (TINYINT)                tenant_company_member
└── auth_super_admin (user_id PK)     └── is_admin (TINYINT)
```

**仅有的鉴权原语**：
- `isSuperAdminUser(userID)` — 查 `auth_super_admin` 表
- `loadUserAuthFlags(userID)` — 返回 `isActive, isSuperuser, isStaff, isArchived`
- `requireSuperuser(w, r)` — system-admin handler 守卫
- `requireCompanyAdmin(w, r, userID, companyID)` — 查 creator 或 is_admin=1
- `requireCompanyMember(w, r, userID, companyID)` — 查 membership 存在

**API 路径现状** (已迁移到 `/api/${serviceName}/${funcName}/key/value/` 规范)：
- taskAuth: `/api/auth/`, `/api/accounts/`, `/api/internal/`, `/api/system-admin/` (存量)
- taskTenantService: `/api/tenant/{tid}/accounts/...` (旧模式，仍用位置参数)
- taskCloudService: `/api/cloud/` (新模式)
- taskBill: `/api/billing/` (新模式)

### 1.2 核心缺失

| 缺失 | 现状 |
|------|------|
| 角色分层 | 只有二元 is_admin / is_superuser |
| 细粒度权限 | 无权限码目录 |
| 统一鉴权入口 | 每个 handler 各自手写检查 |
| 组权限继承 | 有 group 表，无角色关联 |
| 租户隔离强制 | 靠 handler 自行 WHERE company_id |
| 审计追踪 | assigned_by / assigned_at 缺失 |

---

## 2. 设计方案

### 2.1 角色模型

**两级四角色**，优先级继承：

```
Platform (平台层)               Tenant (租户层)
super_admin (p=100)             tenant_admin (p=100)
  │ 继承                          │ 继承
employee (p=50)                  member (p=50)
```

**判定规则**：用户有效优先级 = max(所有拥有角色的 priority)  
通过条件：有效优先级 >= 所需角色的 priority

### 2.2 新增数据表

**taskAuth 库** (4 表)：

```sql
auth_role (id, name, display_name, level, priority, is_system)
auth_permission (id, codename, name, resource_type, action, level)
auth_role_permission (id, role_id, permission_id)
auth_user_role (id, user_id, role_id, company_id NULL, assigned_by, assigned_at, expires_at)
```

**taskTenant 库** (2 表)：

```sql
tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
tenant_group_role (id, group_id, role_name, company_id, assigned_by, assigned_at)
```

### 2.3 Go 代码骨架

**shareLib/authz** — 薄客户端（每个 Go 服务 import 即可）：

```
shareLib/authz/
├── permissions.go    # PermCode 枚举 (const)
├── roles.go          # 系统角色注册表
├── context.go        # AuthContext 注入/提取
└── middleware.go      # RequireRole / RequireTenantRole / RequireTenantMember
```

**核心接口**：

```go
// 零网络调用 — 读取网关注入的 Context
func RequireRole(w http.ResponseWriter, r *http.Request, roles ...string) error
func RequireTenantRole(w http.ResponseWriter, r *http.Request, companyID string, roles ...string) error  
func RequireTenantMember(w http.ResponseWriter, r *http.Request, companyID string) error

// 读取 Context 中的鉴权信息
func FromContext(r *http.Request) (*AuthContext, bool)
```

### 2.4 authz 判定流程

```
请求 → APISIX Gateway
         │ forward-auth → taskAuth
         │ ← X-User-Roles: "super_admin"
         │ ← X-Tenant-Memberships: "c1:tenant_admin,c2:member"
         ▼
       Go Service Handler
         │ authz.RequireTenantRole(w, r, "c1", "tenant_admin")
         │ → FromContext(r) → AuthContext{TenantRoles: {"c1":"tenant_admin"}}
         │ → priority("tenant_admin")=100 >= priority("tenant_admin")=100 → OK
         ▼
       业务逻辑
```

---

## 3. 新增 API

### 3.1 taskAuth (serviceName=`auth`)

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/auth/roles/` | 角色列表 |
| POST | `/api/auth/user-roles/user_id/{uid}/` | 指派角色 |
| DELETE | `/api/auth/user-roles/user_id/{uid}/role_name/{name}/` | 撤销角色 |
| GET | `/api/auth/role-users/role_name/{name}/` | 角色成员列表 |
| GET | `/api/auth/employees/` | 员工列表 |
| POST | `/api/auth/employees/user_id/{uid}/` | 添加员工 |
| DELETE | `/api/auth/employees/user_id/{uid}/` | 移除员工 |
| GET | `/api/auth/user-roles/` | 当前用户角色 |
| GET | `/api/auth/user-permissions/` | 当前用户权限 |
| POST | `/api/internal/auth/permission-check/` | PDP 判定 |

### 3.2 taskTenantService (serviceName=`tenant`)

| 方法 | 路径 | 用途 |
|------|------|------|
| PUT | `/api/tenant/member-role/company_id/{cid}/member_id/{mid}/` | 修改成员角色 |
| PUT | `/api/tenant/group-role/company_id/{cid}/group_id/{gid}/` | 设置组角色 |

> 📦 存量 `/api/tenant/{tid}/accounts/members` 等路径不强制迁移，但新端点一律遵循 key/value 规范。

### 3.3 全部新增 API 合规

所有 12 个新增 API 均遵循 `/api/${serviceName}/${funcName}/key/value/...` 规范。

---

## 4. 系统角色种子数据

| name | level | priority | 初始权限 |
|------|-------|----------|---------|
| super_admin | platform | 100 | platform:manage, tenant:audit, user:impersonate, system:config, billing:audit, employee:manage |
| employee | platform | 50 | tenant:audit, user:impersonate, billing:audit |
| tenant_admin | tenant | 100 | company:manage/view, member:manage/view, group:manage, project:manage/view, task:manage/view, cloud:manage/view, billing:manage/view, workspace:manage |
| member | tenant | 50 | company:view, member:view, project:view, task:manage/view, cloud:view, billing:view |

---

## 5. 权限码枚举 (20 个)

```go
// Platform
PermPlatformManage   = "platform:manage"
PermTenantAudit      = "tenant:audit"
PermUserImpersonate  = "user:impersonate"
PermSystemConfig     = "system:config"
PermBillingAudit     = "billing:audit"
PermEmployeeManage   = "employee:manage"

// Tenant
PermCompanyManage = "company:manage"
PermCompanyView   = "company:view"
PermMemberManage  = "member:manage"
PermMemberView    = "member:view"
PermGroupManage   = "group:manage"
PermProjectManage = "project:manage"
PermProjectView   = "project:view"
PermTaskManage    = "task:manage"
PermTaskView      = "task:view"
PermCloudManage   = "cloud:manage"
PermCloudView     = "cloud:view"
PermBillingManage = "billing:manage"
PermBillingView   = "billing:view"
PermWorkspaceMng  = "workspace:manage"
```

---

## 6. 现有鉴权点迁移清单

| 服务 | 文件 | 当前 | 改为 |
|------|------|------|------|
| taskAuth | `handlers_system_admin.go:20` | `requireSuperuser()` | `authz.RequireRole(w, r, "super_admin")` |
| taskAuth | `auth_email_invite.go` (6处) | `isSuperAdminUser()` | `authz.RequireRole()` |
| taskTenant | `policy.go:70` | `requireCompanyAdmin()` | `authz.RequireTenantRole(w, r, cid, "tenant_admin")` |
| taskTenant | `policy.go:120` | `requireCompanyMember()` | `authz.RequireTenantMember(w, r, cid)` |
| taskTenant | `policy.go:138` | `isCreator()` 硬编码 | CompanyCreated consumer 自动分配 tenant_admin |
| taskCloud | system-admin handlers | `checkSuperAdmin()` 查 DB | `authz.RequireRole(w, r, "super_admin", "employee")` |
| taskBill | system-admin handlers | 自行校验 | `authz.RequireRole(w, r, "super_admin", "employee")` |

---

## 7. 数据库迁移

**迁移文件**：
- `dataMigrate/taskAuth/027_rbac_roles.sql` — 建表 + 种子
- `dataMigrate/taskAuth/028_rbac_backfill.sql` — 回填 is_superuser → super_admin
- `dataMigrate/taskTenantService/017_member_role.sql` — 建表 + 回填 is_admin

**回填逻辑**：
```sql
-- is_superuser=1 → super_admin
INSERT INTO auth_user_role SELECT ... FROM auth_user WHERE is_superuser=1;
-- auth_super_admin → super_admin
INSERT INTO auth_user_role SELECT ... FROM auth_super_admin;
-- is_staff=1 → employee
INSERT INTO auth_user_role SELECT ... FROM auth_user WHERE is_staff=1 AND is_superuser=0;
-- is_admin=1 → tenant_admin
INSERT INTO tenant_member_role SELECT ... FROM tenant_company_member WHERE is_admin=1;
-- creator → tenant_admin
INSERT INTO tenant_member_role SELECT ... FROM tenant_company JOIN tenant_company_member;
```

---

## 8. 事件流

```
CompanyCreated (已有)  → new Consumer → assign tenant_admin to creator
PlatformRoleAssigned   → Consumer → Redis DEL authz:{uid}:roles + audit log
PlatformRoleRevoked    → Consumer → Redis DEL + session invalidate
TenantMemberRoleChanged → Consumer → Redis DEL + SSE notify
TenantGroupRoleChanged  → Consumer → Redis DEL for all group members
```

---

## 9. 实施计划

| Day | 任务 | 验证 |
|-----|------|------|
| 1 | shareLib/authz 代码骨架 + 迁移 SQL | go test + mysql < migration |
| 2 | taskAuth: 角色管理 API + forward-auth 注入 roles | curl 测试 |
| 3 | taskTenant: member-role + group-role API + policy.go 替换 | 现有测试 |
| 4 | taskCloud + taskBill + taskProject + taskTask 迁移 | 现有测试 |
| 5 | taskEvents consumers + APISIX forward-auth 增强 | 事件追踪 |
| 6 | taskFE usePermission() + UI 显隐 | 手动测试 |
| 7 | E2E 测试矩阵 (10 用例) + Code Review | 全部通过 |

---

## 10. 安全约束

| 约束 | 实现 |
|------|------|
| 租户隔离 | AuthContext.company_id 校验 + DB WHERE |
| 权限最小化 | 新用户默认 member，新员工默认 employee |
| 提权审计 | assigned_by + assigned_at + Kafka audit event |
| 缓存即时失效 | 角色变更 → Redis DEL + Pub/Sub 广播 |
| 过期角色 | expires_at < NOW() 自动排除 |
| 被禁成员 | is_active=0 → 所有角色失效 |

---

## 11. 架构变更影响

- **迭代版本**: v61 🎯 target
- **基于**: v57 current (Django 退役 + API 路径规范迁移已完成)
- **变更明细**:
  - 🟡 taskAuth — +4 RBAC 表 + 角色管理 API + PDP 端点
  - 🟡 taskTenantService — +2 RBAC 表 + member/group role API
  - 🟢 shareLib/authz — 薄客户端 (permissions.go + roles.go + context.go + middleware.go)
  - 🟡 APISIX Gateway — forward-auth 增强注入 X-User-Roles + X-Tenant-Memberships
  - 🔴 is_admin/is_superuser 直接鉴权 → authz.RequireRole()
  - 🔴 policy.go creator 硬编码 → CompanyCreated consumer 事件驱动
- **新增 API**: 12 个，全部 Go，零 Python，全部合规路径
- **实施周期**: 7 天

---

## 12. Domain Concept Inventory

| 概念 | 类型 | BC | 说明 |
|------|------|-----|------|
| User | Entity | Identity | 平台用户 |
| Role | Entity | Identity | 角色定义 (name, priority, level) |
| Permission | ValueObject | Identity | 权限码 |
| UserRole | Aggregate | Identity | 用户-角色指派 |
| GroupRole | Aggregate | Tenant | 组-角色关联 |
| MemberRole | Aggregate | Tenant | 成员角色 (替代 is_admin) |
| Company | AggregateRoot | Tenant | 租户隔离边界 |
| PlatformRoleAssigned | DomainEvent | Identity | → 缓存失效 + 审计 |
| TenantMemberRoleChanged | DomainEvent | Tenant | → 缓存失效 + SSE |
| TenantGroupRoleChanged | DomainEvent | Tenant | → 批量缓存失效 |
