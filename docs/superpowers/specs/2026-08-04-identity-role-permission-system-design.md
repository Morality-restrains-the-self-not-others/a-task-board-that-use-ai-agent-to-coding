# 身份角色与权限体系设计

- **状态**: 🎯 target（设计阶段，待审批）
- **迭代**: identity-role-permission-system
- **作者**: claude
- **设计日期**: 2026-08-04 14:00
- **类型**: 架构设计 + 数据模型 + API 设计

---

## 1. 需求概述

### 1.1 核心概念

建立**两层四角色**的身份权限体系，明确 SaaS 使用方与平台维护方的边界：

```
┌─────────────────────────────────────────────────┐
│                  平台层 (Platform)                │
│  SuperAdmin (超管) ──权限继承──> Employee (员工)   │
│  维护 SaaS 服务本身                               │
├─────────────────────────────────────────────────┤
│                  租户层 (Tenant)                  │
│  Tenant (租户管理员) ──权限继承──> User (用户)      │
│  使用 SaaS 服务                                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ Tenant A │  │ Tenant B │  │ Tenant C │  ...  │
│  │  (隔离)   │  │  (隔离)   │  │  (隔离)   │       │
│  └──────────┘  └──────────┘  └──────────┘       │
└─────────────────────────────────────────────────┘
```

### 1.2 四角色定义

| 角色 | 层 | 归属 | 职责 | 权限范围 |
|------|-----|------|------|---------|
| **SuperAdmin** (超管) | Platform | — (root) | 平台全局管理、运维、跨租户审计 | 全部权限 |
| **Employee** (员工) | Platform | SuperAdmin | 日常运维、客服、租户支持 | ≤ SuperAdmin（可配置子集） |
| **Tenant** (租户管理员) | Tenant | Company | 公司内成员/组/资源/计费管理 | 租户内全部权限 |
| **User** (用户/成员) | Tenant | Tenant/Company | 使用项目/任务/云资源等 SaaS 功能 | ≤ Tenant（可配置子集） |

### 1.3 设计原则

| # | 原则 | 说明 |
|---|------|------|
| P1 | **权限继承** | User 权限 ≤ Tenant 权限；Employee 权限 ≤ SuperAdmin 权限 |
| P2 | **归属清晰** | User 隶属于 Tenant (Company)；Employee 隶属于平台 |
| P3 | **租户隔离** | 不同租户的资源互相不可见，交叉访问必须在 API 层阻断 |
| P4 | **Go-First** | 新增接口全部落 Go（扩展现有 taskAuth + taskTenantService） |
| P5 | **渐进迁移** | 兼容现有 `is_superuser`/`is_admin`/`is_staff` 字段，平滑过渡 |
| P6 | **最小惊讶** | 管理员创建者自动获得 tenant_admin 角色；邀请可预分配角色 |

---

## 2. 数据模型

### 2.1 ER 图（新增表）

```
auth_user ──< auth_user_role (平台级角色指派)
auth_role ──< auth_user_role
auth_role ──< auth_role_permission >── auth_permission

tenant_company_member ──< tenant_member_role (租户级角色指派)
tenant_company ──< tenant_role (租户级自定义角色，可选)
```

### 2.2 核心表（taskAuth 库 — 平台级）

#### auth_role（角色定义表）

| 列 | 类型 | 说明 |
|----|------|------|
| id | VARCHAR(64) PK | 雪花ID |
| name | VARCHAR(128) UNIQUE | 角色名，如 `super_admin`, `employee` |
| display_name | VARCHAR(255) | 展示名，如「超级管理员」「平台员工」 |
| level | ENUM('platform','tenant') | 角色层级 |
| priority | INT | 权限优先级（越大越高） |
| is_system | TINYINT | 系统内置角色（不可删除） |
| description | TEXT | 角色说明 |
| created_at / updated_at | DATETIME | 时间戳 |

**种子数据（4 系统角色）：**

| name | level | priority | 说明 |
|------|-------|----------|------|
| super_admin | platform | 100 | 超级管理员 — 平台全部权限 |
| employee | platform | 50 | 平台员工 — 运维/客服权限子集 |
| tenant_admin | tenant | 100 | 租户管理员 — 租户内全部权限 |
| member | tenant | 50 | 普通成员 — 租户内受限权限 |

#### auth_permission（权限定义表）

| 列 | 类型 | 说明 |
|----|------|------|
| id | VARCHAR(64) PK | 雪花ID |
| codename | VARCHAR(255) UNIQUE | 权限码，如 `company:manage`, `billing:view` |
| name | VARCHAR(255) | 展示名 |
| resource_type | VARCHAR(64) | 资源类型（company/project/task/cloud/billing/user） |
| action | VARCHAR(64) | 动作（view/create/edit/delete/manage） |
| level | ENUM('platform','tenant') | 权限层级 |
| description | TEXT | 说明 |

**初始权限码清单：**

| 层级 | 权限码 | 说明 |
|------|--------|------|
| platform | `platform:manage` | 平台全局管理 |
| platform | `tenant:audit` | 跨租户审计 |
| platform | `user:impersonate` | 模拟用户登录 |
| platform | `system:config` | 系统配置管理 |
| platform | `billing:audit` | 跨租户账单审计 |
| platform | `employee:manage` | 员工管理 |
| tenant | `company:manage` | 公司信息管理 |
| tenant | `company:view` | 公司信息查看 |
| tenant | `member:manage` | 成员管理（邀请/移除/角色） |
| tenant | `member:view` | 成员列表查看 |
| tenant | `group:manage` | 分组管理 |
| tenant | `project:manage` | 项目管理 |
| tenant | `project:view` | 项目查看 |
| tenant | `task:manage` | 任务管理 |
| tenant | `task:view` | 任务查看 |
| tenant | `cloud:manage` | 云资源管理 |
| tenant | `cloud:view` | 云资源查看 |
| tenant | `billing:manage` | 计费充值管理 |
| tenant | `billing:view` | 账单查看 |
| tenant | `workspace:manage` | 工作空间管理 |

#### auth_role_permission（角色-权限关联）

| 列 | 类型 | 说明 |
|----|------|------|
| id | VARCHAR(64) PK | |
| role_id | VARCHAR(64) FK → auth_role | |
| permission_id | VARCHAR(64) FK → auth_permission | |
| UNIQUE(role_id, permission_id) | | |

#### auth_user_role（用户-角色指派 — 平台级）

| 列 | 类型 | 说明 |
|----|------|------|
| id | VARCHAR(64) PK | 雪花ID |
| user_id | VARCHAR(64) FK → auth_user | |
| role_id | VARCHAR(64) FK → auth_role | |
| company_id | VARCHAR(64) NULL | NULL=平台级角色；非NULL=租户级（仅 system 关联用） |
| assigned_by | VARCHAR(64) | 指派人 user_id |
| assigned_at | DATETIME | |
| expires_at | DATETIME NULL | 过期时间（临时提权） |
| UNIQUE(user_id, role_id, company_id) | | 同一用户+同一角色+同一作用域唯一 |

### 2.3 租户级表（taskTenantService 库）

#### tenant_member_role（租户成员角色 — 替代现有 is_admin）

| 列 | 类型 | 说明 |
|----|------|------|
| id | VARCHAR(64) PK | |
| member_id | VARCHAR(64) FK → tenant_company_member | |
| role_name | VARCHAR(64) | `tenant_admin` 或 `member` |
| company_id | VARCHAR(64) | 冗余，加速查询 |
| assigned_by | VARCHAR(64) | 指派人 user_id |
| assigned_at | DATETIME | |
| UNIQUE(member_id, role_name) | | |

> **迁移策略**：现有 `tenant_company_member.is_admin=1` → `tenant_member_role(role_name='tenant_admin')`  
> `tenant_company.creator_id` → 同时插入 `tenant_member_role(role_name='tenant_admin')`

### 2.4 租户级自定义角色（可选，Phase 2）

#### tenant_role（租户自定义角色模板）

| 列 | 类型 | 说明 |
|----|------|------|
| id | VARCHAR(64) PK | |
| company_id | VARCHAR(64) | |
| name | VARCHAR(128) | 角色名 |
| permissions | JSON | 权限码列表 `["task:view","task:manage",...]` |
| created_at / updated_at | DATETIME | |

---

## 3. 权限检查流程

### 3.1 鉴权中间件架构

```
请求 → APISIX Gateway
       │
       ├─ /api/system-admin/* → gateway forward-auth
       │    └─ 校验 Platform 角色 (super_admin | employee)
       │
       └─ /api/tenant/{company_id}/* → gateway forward-auth
            └─ 校验 Tenant 角色 (tenant_admin | member)
                 + 校验 user 属于 company_id（租户隔离）
```

### 3.2 API 鉴权函数（shareLib 新增）

```go
// shareLib/authz/permission.go

// RequirePlatformRole 要求用户具有平台级角色
func RequirePlatformRole(r *http.Request, roles ...string) error

// RequireTenantRole 要求用户在指定租户中具有某角色
func RequireTenantRole(r *http.Request, companyID string, roles ...string) error

// HasPermission 检查用户是否具有某权限码
func HasPermission(ctx context.Context, userID, companyID, perm string) (bool, error)

// GetUserRoles 获取用户的所有角色
func GetUserRoles(ctx context.Context, userID, companyID string) ([]Role, error)
```

### 3.3 租户隔离强制点

| 层级 | 强制方式 |
|------|---------|
| **API 网关** | APISIX forward-auth 校验 session → taskAuth 返回 user + platform roles |
| **Go 服务中间件** | 每个 `/api/tenant/{company_id}/*` handler 入口调用 `RequireTenantMember()` |
| **数据访问层** | 所有查询必须带 `WHERE company_id = ?` 或等价租户过滤 |
| **Kafka 事件** | 事件携带 `tenant_id`，消费者按 `tenant_id` 路由/过滤 |

---

## 4. 权限继承实现

### 4.1 角色优先级模型

```
权限检查 = max(用户直接权限, 继承自上级角色的权限)

Platform: SuperAdmin(priority=100) > Employee(priority=50)
Tenant:    TenantAdmin(priority=100) > Member(priority=50)

检查逻辑:
  userRoles = getUserRoles(userID, companyID)
  effectivePriority = max(r.priority for r in userRoles)
  requiredPriority = requiredRole.priority
  → effectivePriority >= requiredPriority → 通过
```

### 4.2 权限集合模型（Phase 2，更细粒度）

```
User 权限集合 ⊆ Tenant 权限集合
Employee 权限集合 ⊆ SuperAdmin 权限集合

检查逻辑:
  userPerms = getEffectivePermissions(userID, companyID)
  → requiredPerm ∈ userPerms → 通过
```

### 4.3 优先级策略

- **Phase 1（本次）**: 基于角色优先级的粗粒度鉴权（满足 P1-P3）
- **Phase 2（后续）**: 基于权限码的细粒度鉴权 + 租户自定义角色（满足高级需求）

---

## 5. API 设计

### 5.1 新增 API（全部 Go）

| 方法 | 路径 | 归属服务 | 说明 |
|------|------|---------|------|
| GET | /api/system-admin/roles | taskAuth | 平台角色列表 |
| POST | /api/system-admin/roles/{role_id}/assign | taskAuth | 指派平台角色给用户 |
| DELETE | /api/system-admin/roles/{role_id}/users/{user_id} | taskAuth | 撤销平台角色 |
| GET | /api/system-admin/roles/{role_id}/users | taskAuth | 查看某角色下的用户列表 |
| GET | /api/system-admin/employees | taskAuth | 员工列表 |
| POST | /api/system-admin/employees | taskAuth | 添加员工 |
| GET | /api/tenant/{company_id}/roles | taskTenantService | 租户内角色列表 |
| PUT | /api/tenant/{company_id}/members/{member_id}/role | taskTenantService | 修改成员角色（替代现有 update_role） |
| GET | /api/auth/user/permissions | taskAuth | 当前用户的权限列表 |
| GET | /api/auth/user/roles | taskAuth | 当前用户的角色列表 |

### 5.2 修改现有 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/tenant/{company_id}/members | 响应增加 `roles: ["tenant_admin"]` 字段 |
| POST | /api/tenant/{company_id}/invitations | `role` 字段增加校验（只能是 tenant_admin/member） |
| GET | /api/system-admin/users | 增加角色筛选参数 `?role=super_admin` |

### 5.3 Python 新增接口评估

本次设计**不新增 Python 接口**。所有新 API 均落在 Go 微服务：
- 平台级角色管理 → `taskAuth`
- 租户级角色管理 → `taskTenantService`
- 权限检查中间件 → `shareLib` (Go)

> ✅ 不触发「Python 新增接口清单与 Go 替代评估」额外审批。

---

## 6. 迁移方案

### 6.1 数据迁移 SQL（dataMigrate）

```sql
-- 1. 创建角色定义表
-- dataMigrate/taskAuth/023_role_permission.sql

-- 2. 创建权限表
-- dataMigrate/taskAuth/023_role_permission.sql

-- 3. 种子数据：4 系统角色 + 初始权限码

-- 4. 回填现有数据：
--    auth_user.is_superuser=1 → auth_user_role(role='super_admin')
--    auth_super_admin → auth_user_role(role='super_admin')
--    tenant_company_member.is_admin=1 → tenant_member_role(role='tenant_admin')
--    tenant_company.creator_id → tenant_member_role(role='tenant_admin')
```

### 6.2 兼容期策略

| 旧字段 | 兼容方式 | 废弃时间 |
|--------|---------|---------|
| `auth_user.is_superuser` | 保留；写入时同步 `auth_user_role` | Phase 2 |
| `auth_user.is_staff` | 保留；映射为 `employee` 角色 | Phase 2 |
| `tenant_company_member.is_admin` | 保留；写入时同步 `tenant_member_role` | Phase 2 |
| `tenant_company.creator_id` | 保留；创建公司时自动分配 `tenant_admin` 角色 | 长期 |

---

## 7. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 平台角色被指派 | PlatformRoleAssigned | taskAuth AssignRoleHandler | 审计日志、通知 | — |
| 平台角色被撤销 | PlatformRoleRevoked | taskAuth RevokeRoleHandler | 审计日志、session 失效 | — |
| 租户角色被修改 | TenantRoleChanged | taskTenantService UpdateRoleHandler | 前端 SSE 刷新权限 | — |
| 员工入职 | EmployeeOnboarded | taskAuth CreateEmployeeHandler | 通知、初始化员工资源 | — |
| 员工离职 | EmployeeOffboarded | taskAuth DeleteEmployeeHandler | 撤销所有权限、审计 | — |

---

## 8. 架构变更影响

- **迭代版本**: v59 🎯 target
- **迭代名称**: identity-role-permission-system
- **作者**: claude
- **设计日期**: 2026-08-04 14:00
- **新增文件**（每个视图三类伴生格式）:
  - 🆕 `docs/architecture/v59-application-integration-20260804-1400-claude.puml`
  - 🆕 `docs/architecture/v59-application-integration-20260804-1400-claude.archimate`（含 Plateau/Gap/WP 架构变迁视图）
  - 🆕 `docs/architecture/v59-application-integration-20260804-1400-claude.mermaid.md`
- **已有文件（未修改）**:
  - `docs/architecture/v57-application-integration-20260730-0229-claude.puml` (current)
- **变更明细**:
  - 🟡 [MODIFIED] taskAuth — 新增 auth_role/auth_permission/auth_role_permission/auth_user_role 表；角色 CRUD API；权限查询 API
  - 🟡 [MODIFIED] taskTenantService — 新增 tenant_member_role 表；角色管理 API；迁移 is_admin→role
  - 🟢 [NEW] shareLib/authz — 权限检查中间件 (Go)；RequirePlatformRole/RequireTenantRole/HasPermission
  - 🟡 [MODIFIED] taskAuth DB — 新增 4 张角色权限表
  - 🟡 [MODIFIED] taskTenant DB — 新增 1 张 tenant_member_role 表
  - 🔴 [DEPRECATED] 旧 is_admin/is_superuser 直接鉴权 → 改为通过角色优先级
  - 🆕 数据迁移: `dataMigrate/taskAuth/023_role_permission.sql` + `dataMigrate/taskTenantService/016_member_role.sql`

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v57** | Current 基线 — 纯 Go 微服务，二元 is_admin/is_superuser 鉴权 |
| **Plateau v59** | Target — RBAC 角色权限体系，两级四角色，租户隔离 |
| **Gap** | 缺分层角色体系、缺细粒度权限、缺租户隔离强制点 |
| **WorkPackage** | Phase 1: 数据模型+API+中间件；Phase 2: 细粒度权限+自定义角色 |
| **视图** | `架构变迁 v57→v59 — 身份角色权限体系`（可导入 Archi） |

---

## 9. Domain Concept Inventory

| 概念 | 类型 | 归属 BC | 说明 |
|------|------|---------|------|
| User | Entity | Identity (taskAuth) | 平台用户，有唯一 ID |
| Role | Entity | Identity (taskAuth) | 角色定义，含优先级和层级 |
| Permission | Entity | Identity (taskAuth) | 权限码定义 |
| UserRole | Aggregate | Identity (taskAuth) | 用户-角色指派（平台级） |
| MemberRole | Aggregate | Tenant (taskTenantService) | 租户成员-角色指派 |
| Company/Tenant | Aggregate Root | Tenant (taskTenantService) | 租户隔离边界 |
| CompanyMember | Entity | Tenant (taskTenantService) | 租户成员 |
| PlatformRoleAssigned | DomainEvent | Identity | 平台角色指派事件 |
| PlatformRoleRevoked | DomainEvent | Identity | 平台角色撤销事件 |
| TenantRoleChanged | DomainEvent | Tenant | 租户角色变更事件 |
| EmployeeOnboarded | DomainEvent | Identity | 员工入职事件 |
| EmployeeOffboarded | DomainEvent | Identity | 员工离职事件 |

---

## 10. 安全约束

| 约束 | 实现 |
|------|------|
| **租户隔离** | shareLib/authz 中间件强制 `company_id` 校验；DB 查询强制 tenant 过滤 |
| **权限最小化** | 新用户默认 `member` 角色；新员工默认 `employee` 角色 |
| **提权审计** | 所有角色变更记录 `assigned_by` + `assigned_at`，写入审计日志 |
| **Session 同步** | 角色变更后触发 token/session 权限刷新（可选：立即失效旧 session） |
| **API 暴露面** | 平台管理 API (`/api/system-admin/*`) 仅平台角色可访问 |

---

## 11. 附录

### A. 现有权限检查点审计（需改造）

| 位置 | 当前检查 | 新检查 |
|------|---------|--------|
| taskTenantService/policy.go:requireCompanyAdmin | is_admin==1 OR creator | HasTenantRole(tenant_admin) |
| taskTenantService/policy.go:requireCompanyMember | member exists | HasTenantRole(member+) |
| taskAuth 各 handler | is_superuser==1 | HasPlatformRole(super_admin) |
| taskCloudService system-admin | 自行校验 | 迁移到 shareLib/authz |
| taskBill system-admin | 自行校验 | 迁移到 shareLib/authz |
| taskProjectService workspace ACL | permission VARCHAR | 迁移到 role 模型 |

### B. 与现有架构版本的关系

- v58 (email-invite-enhancement) 中的 `assigned_role` 字段与本设计一致
- v56 (Django→Go final migration) 已确立 taskAuth 为 auth 真源，本设计在其上叠加角色层
- 与现有 `auth_super_admin` 表可共存，逐步迁移到 `auth_user_role`
