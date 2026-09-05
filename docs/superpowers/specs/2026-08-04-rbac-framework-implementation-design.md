# RBAC 框架与实施方案

- **状态**: 🎯 target（设计阶段，待审批）
- **迭代**: rbac-framework-implementation
- **作者**: claude
- **设计日期**: 2026-08-04 14:00
- **基于**: v59 identity-role-permission-system 高层设计
- **类型**: RBAC 框架架构 + 引擎设计 + 集成方案 + 实施计划

---

## 1. 设计目标

在 v59 确立的两级四角色数据模型基础上，构建一个**可复用的、类型安全的、高性能的 Go RBAC 框架**。框架的核心诉求：

| 目标 | 描述 |
|------|------|
| **一次编写，到处调用** | shareLib/authz 作为共享库，所有 Go 服务 `import` 即可接入 |
| **零拷贝权限判定** | 网关 forward-auth 一次性注入权限上下文，下游服务直接读取，无需重复查库 |
| **编译时安全** | 权限码使用 Go const 枚举，杜绝拼写错误；角色使用类型别名 |
| **极简接入** | 每个 handler 加一行 `authz.Require(w, r, authz.PermTaskView)` 即可 |
| **缓存友好** | Redis 缓存用户权限集合，角色变更即时失效，命中率 >99% |
| **渐进式** | Phase 1 用优先级鉴权（零权限码表查询），Phase 2 切到权限码细粒度 |

---

## 2. RBAC 框架架构

### 2.1 框架分层

```
┌──────────────────────────────────────────────────────────────┐
│                    shareLib/authz                             │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │           HTTP Middleware (authz/http.go)             │     │
│  │  RequireRole()  RequirePerm()  RequireTenantMember() │     │
│  │  → 读 r.Context() 中的 AuthContext，执行判定            │     │
│  └──────────────────────┬──────────────────────────────┘     │
│                         │                                    │
│  ┌──────────────────────▼──────────────────────────────┐     │
│  │          Permission Engine (authz/engine.go)          │     │
│  │  HasPermission(ctx, userID, companyID, perm) → bool   │     │
│  │  GetEffectiveRoles(ctx, userID, companyID) → []Role   │     │
│  │  GetPermissions(ctx, userID, companyID) → []PermCode  │     │
│  └──────────────────────┬──────────────────────────────┘     │
│                         │                                    │
│  ┌──────────────────────▼──────────────────────────────┐     │
│  │         Role Registry (authz/roles.go)                │     │
│  │  RoleDefinition{Priority, Level, Permissions}         │     │
│  │  systemRoles map[string]RoleDefinition               │     │
│  └──────────────────────┬──────────────────────────────┘     │
│                         │                                    │
│  ┌──────────────────────▼──────────────────────────────┐     │
│  │       Permission Catalog (authz/permissions.go)       │     │
│  │  const PermTaskView PermCode = "task:view"            │     │
│  │  AllPermissions() → []PermCode                        │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │         Cache Layer (authz/cache.go)                  │     │
│  │  GetCachedUserRoles(ctx, userID) → *CachedRoles       │     │
│  │  InvalidateUserRoles(ctx, userID)                     │     │
│  │  → Redis: "authz:user:{userID}:roles" (TTL=5min)      │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │      RBAC Client (authz/client.go)                    │     │
│  │  FetchUserRoles(ctx, userID, companyID) → []UserRole  │     │
│  │  → HTTP GET /api/internal/authz/user/{userID}/roles   │     │
│  │  (服务间调用 taskAuth / taskTenantService)              │     │
│  └─────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
```

### 2.2 Go 包结构

```
shareLib/authz/
├── authz.go             // 包文档 + 顶层导出
├── context.go           // AuthContext 结构体，注入 http.Request Context
├── roles.go             // 角色定义 + 系统角色注册表
├── permissions.go       // 权限码枚举 + 权限目录
├── engine.go            // 权限决策引擎
├── http.go              // HTTP 中间件（RequireRole/RequirePerm/RequireTenantMember）
├── cache.go             // Redis 缓存层
├── cache_disabled.go    // build tag: !redis — 无 Redis 时回退到内存 LRU
├── client.go            // taskAuth/taskTenant HTTP 客户端
├── inject.go            // Gateway forward-auth 注入逻辑
├── middleware_test.go
├── engine_test.go
└── roles_permissions_test.go
```

### 2.3 核心接口

```go
// authz/roles.go — 角色定义

type RoleLevel string
const (
    RoleLevelPlatform RoleLevel = "platform"
    RoleLevelTenant   RoleLevel = "tenant"
)

type RoleDefinition struct {
    Name        string      // "super_admin", "tenant_admin", ...
    DisplayName string      // "超级管理员"
    Level       RoleLevel   // platform | tenant
    Priority    int         // 越大权限越高，用于 Phase 1 优先级判定
    IsSystem    bool        // 内置角色不可删除
    Permissions []PermCode  // Phase 2: 关联的权限码列表
}

// 系统内置角色（编译时注册）
var SystemRoles = map[string]RoleDefinition{
    "super_admin":  {Name: "super_admin",  Level: RoleLevelPlatform, Priority: 100, IsSystem: true, ...},
    "employee":     {Name: "employee",     Level: RoleLevelPlatform, Priority: 50,  IsSystem: true, ...},
    "tenant_admin": {Name: "tenant_admin", Level: RoleLevelTenant,   Priority: 100, IsSystem: true, ...},
    "member":       {Name: "member",       Level: RoleLevelTenant,   Priority: 50,  IsSystem: true, ...},
}
```

```go
// authz/permissions.go — 权限码枚举

type PermCode string

// ═══ Platform 权限 ═══
const (
    PermPlatformManage   PermCode = "platform:manage"
    PermTenantAudit      PermCode = "tenant:audit"
    PermUserImpersonate  PermCode = "user:impersonate"
    PermSystemConfig     PermCode = "system:config"
    PermBillingAudit     PermCode = "billing:audit"
    PermEmployeeManage   PermCode = "employee:manage"
)

// ═══ Tenant 权限 ═══
const (
    PermCompanyManage PermCode = "company:manage"
    PermCompanyView   PermCode = "company:view"
    PermMemberManage  PermCode = "member:manage"
    PermMemberView    PermCode = "member:view"
    PermGroupManage   PermCode = "group:manage"
    PermProjectManage PermCode = "project:manage"
    PermProjectView   PermCode = "project:view"
    PermTaskManage    PermCode = "task:manage"
    PermTaskView      PermCode = "task:view"
    PermCloudManage   PermCode = "cloud:manage"
    PermCloudView     PermCode = "cloud:view"
    PermBillingManage PermCode = "billing:manage"
    PermBillingView   PermCode = "billing:view"
    PermWorkspaceMng  PermCode = "workspace:manage"
)

// AllPermissions 返回所有权限码（用于种子数据）
func AllPermissions() []PermCode { ... }
```

```go
// authz/context.go — 请求上下文中的鉴权信息

type AuthContext struct {
    UserID       string
    PlatformRoles []string    // ["super_admin"] or ["employee"]
    TenantRoles  map[string][]string  // companyID → ["tenant_admin"] or ["member"]
    Permissions  []PermCode   // 可选：预计算的权限集合
}

type ctxKey struct{}

// FromContext 从 http.Request 提取 AuthContext
func FromContext(r *http.Request) (*AuthContext, bool)

// WithContext 注入 AuthContext（由 Gateway forward-auth handler 调用）
func WithContext(r *http.Request, ac *AuthContext) *http.Request
```

```go
// authz/engine.go — 权限决策引擎

type Engine struct {
    cache    Cache
    client   Client
}

// HasPermission 是核心判定方法：用户是否具有某权限码？
func (e *Engine) HasPermission(ctx context.Context, userID, companyID string, perm PermCode) (bool, error)

// HasRole 判定用户是否具有某个角色（或更高级别角色）
func (e *Engine) HasRole(ctx context.Context, userID, companyID string, roleName string) (bool, error)

// GetEffectiveRoles 获取用户在指定公司内的有效角色列表
func (e *Engine) GetEffectiveRoles(ctx context.Context, userID, companyID string) ([]string, error)

// GetEffectivePermissions 获取用户在指定公司内的有效权限集合
func (e *Engine) GetEffectivePermissions(ctx context.Context, userID, companyID string) ([]PermCode, error)

// InvalidateUserCache 角色变更后失效缓存
func (e *Engine) InvalidateUserCache(ctx context.Context, userID string) error
```

```go
// authz/http.go — HTTP 中间件（业务 handler 直接调用）

// RequireRole 要求用户具有指定角色（任意一个即可）
// 用法: if err := authz.RequireRole(w, r, "super_admin", "employee"); err != nil { return }
func RequireRole(w http.ResponseWriter, r *http.Request, roles ...string) error

// RequirePerm 要求用户具有指定权限码
// 用法: if err := authz.RequirePerm(w, r, "", authz.PermTaskView); err != nil { return }
func RequirePerm(w http.ResponseWriter, r *http.Request, companyID string, perm PermCode) error

// RequireTenantMember 要求用户属于指定租户
// 用法: if err := authz.RequireTenantMember(w, r, companyID); err != nil { return }
func RequireTenantMember(w http.ResponseWriter, r *http.Request, companyID string) error

// RequireTenantRole 要求用户在指定租户中具有指定角色
func RequireTenantRole(w http.ResponseWriter, r *http.Request, companyID string, roles ...string) error
```

---

## 3. 权限决策引擎算法

### 3.1 Phase 1: 优先级判定（当前实现）

```
HasPermission(userID, companyID, perm):
  1. 从 AuthContext 或缓存获取用户角色列表
  2. 如果 userID == company_creator_id → 自动提升到 tenant_admin priority
  3. effectivePriority = max(role.Priority for role in userRoles)
  4. requiredRole = lookupRoleForPermission(perm)
  5. return effectivePriority >= requiredRole.Priority
```

**时间复杂度**: O(1)，纯内存比较，无 DB 查询  
**准确度**: 粗粒度（admin 能做所有事，member 只能做部分事）  
**适用**: Phase 1 快速上线

### 3.2 Phase 2: 权限码精确判定

```
HasPermission(userID, companyID, perm):
  1. 从 AuthContext 或缓存获取用户权限集合 (Set<PermCode>)
  2. return perm ∈ userPerms

getEffectivePermissions(userID, companyID):
  1. roles = getEffectiveRoles(userID, companyID)
  2. perms = {}
  3. for each role in roles:
       perms = perms ∪ role.Permissions
  4. return perms
```

**时间复杂度**: O(R × P) → 缓存后 O(1)  
**准确度**: 细粒度（可精确控制每个角色的每个权限码）  
**适用**: Phase 2 全面上线

### 3.3 判定流程图

```
HTTP Request
    │
    ▼
┌──────────────────┐
│ APISIX Gateway   │  forward-auth → taskAuth
│ X-User-Id        │  taskAuth 返回: platform roles + tenant memberships
│ X-User-Roles     │  注入 HTTP Header
│ X-Tenant-Roles   │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Go Service       │  业务 handler
│ authz.RequirePerm│
│ (w, r,           │
│  companyID,      │
│  PermTaskView)   │
└──────┬───────────┘
       │
       ▼
  ┌─────────┐     ┌──────────────┐
  │ Context │ ──► │ AuthContext   │  从 r.Context() 读取
  │ 有缓存?  │ YES │ .Permissions  │  → O(1) 内存判定
  └────┬────┘     └──────────────┘
       │ NO
       ▼
  ┌─────────────┐
  │ Redis Cache │  查 "authz:user:{id}:roles"
  │ 命中?        │  → 反序列化 AuthContext
  └────┬────────┘
       │ MISS
       ▼
  ┌─────────────────────┐
  │ taskAuth internal   │  GET /api/internal/authz/user/{id}/roles
  │ + taskTenantService │  + GET /api/internal/tenant/memberships?user_id={id}
  └──────┬──────────────┘
         │
         ▼
  ┌──────────────┐
  │ 写入 Redis   │  TTL = 5 min
  │ 返回判定结果  │
  └──────────────┘
```

---

## 4. 角色-权限映射矩阵

### 4.1 平台层权限矩阵

| 权限码 | super_admin | employee |
|--------|:-----------:|:--------:|
| `platform:manage` | ✅ | ❌ |
| `tenant:audit` | ✅ | ✅ |
| `user:impersonate` | ✅ | ✅ |
| `system:config` | ✅ | ❌ |
| `billing:audit` | ✅ | ✅ |
| `employee:manage` | ✅ | ❌ |

### 4.2 租户层权限矩阵

| 权限码 | tenant_admin | member | 用户可见功能 |
|--------|:------------:|:------:|------------|
| `company:manage` | ✅ | ❌ | 公司设置页 |
| `company:view` | ✅ | ✅ | 公司基本信息 |
| `member:manage` | ✅ | ❌ | 邀请/移除/改角色 |
| `member:view` | ✅ | ✅ | 成员列表 |
| `group:manage` | ✅ | ❌ | 创建/编辑/删除组 |
| `project:manage` | ✅ | ❌ | 创建/删除项目 |
| `project:view` | ✅ | ✅ | 项目列表+详情 |
| `task:manage` | ✅ | ✅ | 创建/编辑/分配任务 |
| `task:view` | ✅ | ✅ | 任务列表+详情 |
| `cloud:manage` | ✅ | ❌ | 云资源创建/管理 |
| `cloud:view` | ✅ | ✅ | 云资源状态查看 |
| `billing:manage` | ✅ | ❌ | 充值/订阅管理 |
| `billing:view` | ✅ | ✅ | 账单查看 |
| `workspace:manage` | ✅ | ❌ | 工作空间设置 |

> 💡 Phase 1 用优先级策略：super_admin/tenant_admin 有全部权限，employee/member 有各自子集。
> Phase 2 可在此基础上按列精确调整（如允许 member 创建项目、限制 employee 特定操作）。

---

## 5. 缓存策略

### 5.1 缓存键设计

```
# 用户平台角色缓存
Key: "authz:user:{userID}:platform_roles"
Val: ["super_admin"]  或  ["employee"]  或  []
TTL: 300s (5 min)

# 用户租户角色缓存
Key: "authz:user:{userID}:tenant_roles"
Val: {"companyA": ["tenant_admin"], "companyB": ["member"]}
TTL: 300s

# 用户权限码缓存 (Phase 2)
Key: "authz:user:{userID}:permissions:{companyID}"
Val: ["task:view", "task:manage", "project:view", ...]
TTL: 300s

# 角色定义缓存（全量，极少变更）
Key: "authz:role_definitions"
Val: { "super_admin": {...}, "employee": {...}, ... }
TTL: 3600s (1 hour)
```

### 5.2 缓存失效策略

```go
// 事件驱动的缓存失效 —— 角色变更时精确失效

// taskAuth 角色变更时
func (h *AssignRoleHandler) Handle(ctx context.Context, cmd AssignRoleCmd) error {
    // 1. 写入 DB
    err := h.repo.AssignRole(ctx, cmd.UserID, cmd.RoleID, cmd.CompanyID)
    // 2. 发布事件
    h.eventBus.Publish(ctx, PlatformRoleAssigned{UserID: cmd.UserID, Role: cmd.RoleName})
    // 3. 精确失效缓存
    h.cache.InvalidateUserRoles(ctx, cmd.UserID)
    return err
}

// taskEvents consumer 收到事件后，通知所有持有该用户缓存的节点
func (c *RoleEventConsumer) OnPlatformRoleAssigned(event PlatformRoleAssigned) {
    c.cache.InvalidateUserRoles(context.Background(), event.UserID)
    // 可选: 广播 Redis Pub/Sub 让其他服务节点也失效本地缓存
    c.redis.Publish("authz:invalidation", event.UserID)
}
```

### 5.3 无 Redis 回退（开发/测试环境）

```go
//go:build !redis

// cache_disabled.go — 使用进程内 sync.Map + LRU
type memCache struct {
    mu    sync.RWMutex
    store map[string]*cacheEntry
    lru   []string  // 最多 10000 条
}
```

---

## 6. 各服务接入指南

### 6.1 接入清单

| 服务 | 接入方式 | 改造范围 |
|------|---------|---------|
| **taskAuth** | import shareLib/authz + 新增 internal API 提供角色查询 | ~200行 |
| **taskTenantService** | import shareLib/authz + 新增 internal API + 替换 policy.go | ~300行 |
| **taskCloudService** | import shareLib/authz + 替换 system-admin handler 中的 hand-rolled check | ~80行 |
| **taskBill** | import shareLib/authz + 替换 system-admin handler 中的 hand-rolled check | ~60行 |
| **taskProjectService** | import shareLib/authz + workspace ACL 迁移到 role 模型 | ~120行 |
| **taskTaskService** | import shareLib/authz + 替换 hasWorkspaceAccess | ~80行 |
| **taskAIComment** | import shareLib/authz（按需） | ~20行 |
| **taskEvents** | import shareLib/authz（按需） | ~20行 |
| **APISIX Gateway** | forward-auth 增强：注入 X-User-Roles + X-Tenant-Roles header | 配置修改 |
| **taskFE** | 根据 API 返回的 roles 控制 UI 元素显隐 | ~150行 |

### 6.2 接入模式示例

**模式 A — 平台级权限检查（taskCloud handler 改造前 vs 改造后）**

```go
// === 改造前 (taskCloud/src/handlers_system_admin.go) ===
func handleAdminCloudConfig(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-Id")
    isAdmin, _ := checkSuperAdmin(userID)  // hand-rolled, 查 DB
    if !isAdmin {
        writeJSON(w, 403, map[string]string{"error": "需要超管权限"})
        return
    }
    // ... 业务逻辑
}

// === 改造后 ===
func handleAdminCloudConfig(w http.ResponseWriter, r *http.Request) {
    if err := authz.RequireRole(w, r, "super_admin", "employee"); err != nil {
        return  // RequireRole 内部已写入 403
    }
    // ... 业务逻辑
}
```

**模式 B — 租户级权限检查（taskTenantService handler 改造）**

```go
// === 改造前 (policy.go) ===
func handleCompanyMembers(w http.ResponseWriter, r *http.Request, tenantID string) {
    userID := r.Header.Get("X-User-Id")
    if !requireCompanyAdmin(w, userID, tenantID) {  // hand-rolled, 查 is_admin
        return
    }
    // ... 业务逻辑
}

// === 改造后 ===
func handleCompanyMembers(w http.ResponseWriter, r *http.Request, tenantID string) {
    if err := authz.RequireTenantRole(w, r, tenantID, "tenant_admin"); err != nil {
        return
    }
    // ... 业务逻辑
}
```

**模式 C — 细粒度权限检查（Phase 2）**

```go
func handleCreateProject(w http.ResponseWriter, r *http.Request, tenantID string) {
    if err := authz.RequirePerm(w, r, tenantID, authz.PermProjectManage); err != nil {
        return
    }
    // ... 业务逻辑
}
```

### 6.3 前端 UI 权限控制

```javascript
// taskFE/src/composables/usePermission.js
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function usePermission() {
  const auth = useAuthStore()

  const canManageCompany = computed(() =>
    auth.currentTenantRoles?.includes('tenant_admin')
  )

  const canCreateProject = computed(() =>
    auth.currentTenantPermissions?.includes('project:manage')
  )

  const isPlatformStaff = computed(() =>
    auth.platformRoles?.some(r => ['super_admin', 'employee'].includes(r))
  )

  return { canManageCompany, canCreateProject, isPlatformStaff }
}
```

---

## 7. 数据库迁移

### 7.1 迁移文件规划

```
dataMigrate/taskAuth/027_rbac_roles.sql       — auth_role + auth_permission + auth_role_permission + auth_user_role
dataMigrate/taskAuth/028_rbac_seed.sql        — 种子数据（4 角色 + 20 权限码 + 关联）
dataMigrate/taskAuth/029_rbac_backfill.sql    — 回填现有 is_superuser/is_staff → auth_user_role
dataMigrate/taskTenantService/017_member_role.sql — tenant_member_role 表 + 回填 is_admin
```

### 7.2 种子数据 SQL（taskAuth/028_rbac_seed.sql）

```sql
-- 4 系统角色
INSERT INTO auth_role (id, name, display_name, level, priority, is_system, description) VALUES
('role-super-admin',  'super_admin',  '超级管理员', 'platform', 100, 1, '平台全部权限，管理员工，系统配置'),
('role-employee',     'employee',     '平台员工',   'platform', 50,  1, '日常运维、客服、租户支持'),
('role-tenant-admin', 'tenant_admin', '租户管理员', 'tenant',   100, 1, '公司内成员/资源/计费管理'),
('role-member',       'member',       '成员',       'tenant',   50,  1, '使用 SaaS 功能，权限受限');

-- 20 权限码
INSERT INTO auth_permission (id, codename, name, resource_type, action, level) VALUES
-- Platform
('perm-plat-manage',   'platform:manage',   '平台全局管理',   'platform', 'manage',      'platform'),
('perm-tenant-audit',  'tenant:audit',      '跨租户审计',     'platform', 'audit',       'platform'),
('perm-impersonate',   'user:impersonate',  '模拟用户登录',   'user',     'impersonate', 'platform'),
('perm-sys-config',    'system:config',     '系统配置管理',   'system',   'manage',      'platform'),
('perm-bill-audit',    'billing:audit',     '跨租户账单审计', 'billing',  'audit',       'platform'),
('perm-emp-manage',    'employee:manage',   '员工管理',       'employee', 'manage',      'platform'),
-- Tenant
('perm-company-manage','company:manage',    '公司信息管理',   'company',  'manage',      'tenant'),
('perm-company-view',  'company:view',      '公司信息查看',   'company',  'view',        'tenant'),
('perm-member-manage', 'member:manage',     '成员管理',       'member',   'manage',      'tenant'),
('perm-member-view',   'member:view',       '成员列表查看',   'member',   'view',        'tenant'),
('perm-group-manage',  'group:manage',      '分组管理',       'group',    'manage',      'tenant'),
('perm-project-manage','project:manage',    '项目管理',       'project',  'manage',      'tenant'),
('perm-project-view',  'project:view',      '项目查看',       'project',  'view',        'tenant'),
('perm-task-manage',   'task:manage',       '任务管理',       'task',     'manage',      'tenant'),
('perm-task-view',     'task:view',         '任务查看',       'task',     'view',        'tenant'),
('perm-cloud-manage',  'cloud:manage',      '云资源管理',     'cloud',    'manage',      'tenant'),
('perm-cloud-view',    'cloud:view',        '云资源查看',     'cloud',    'view',        'tenant'),
('perm-billing-manage','billing:manage',    '计费充值管理',   'billing',  'manage',      'tenant'),
('perm-billing-view',  'billing:view',      '账单查看',       'billing',  'view',        'tenant'),
('perm-workspace-mng', 'workspace:manage',  '工作空间管理',   'workspace','manage',      'tenant');

-- 角色-权限关联
-- super_admin: 所有 platform 权限
INSERT INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-sa-', id), 'role-super-admin', id
FROM auth_permission WHERE level = 'platform';

-- employee: 除 platform:manage, system:config, employee:manage 外的 platform 权限
INSERT INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-emp-', id), 'role-employee', id
FROM auth_permission WHERE level = 'platform'
AND codename NOT IN ('platform:manage', 'system:config', 'employee:manage');

-- tenant_admin: 所有 tenant 权限
INSERT INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-ta-', id), 'role-tenant-admin', id
FROM auth_permission WHERE level = 'tenant';

-- member: 所有 view 权限 + task:manage
INSERT INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-mb-', id), 'role-member', id
FROM auth_permission WHERE level = 'tenant'
AND action IN ('view');
-- member 额外有 task:manage (成员可以创建和编辑自己参与的任务)
INSERT INTO auth_role_permission (id, role_id, permission_id)
VALUES ('rp-mb-task-manage', 'role-member', 'perm-task-manage');
```

### 7.3 回填 SQL（taskAuth/029_rbac_backfill.sql）

```sql
-- 已有超管 → super_admin 角色
INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
SELECT CONCAT('backfill-sa-', id), id, 'role-super-admin', NULL, 'system', NOW()
FROM auth_user WHERE is_superuser = 1;

-- 已有 auth_super_admin 行 → super_admin 角色
INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
SELECT CONCAT('backfill-sa2-', user_id), user_id, 'role-super-admin', NULL, 'system', NOW()
FROM auth_super_admin
WHERE user_id NOT IN (SELECT user_id FROM auth_user_role WHERE role_id = 'role-super-admin');

-- is_staff=1 → employee 角色
INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
SELECT CONCAT('backfill-emp-', id), id, 'role-employee', NULL, 'system', NOW()
FROM auth_user WHERE is_staff = 1 AND is_superuser = 0;
```

---

## 8. 实施计划

### Phase 0 — 基础设施（1-2天）

| # | 任务 | 产出 |
|---|------|------|
| 0.1 | 创建 `shareLib/authz/` 包骨架 | Go 包结构 + 接口定义 |
| 0.2 | 实现 `permissions.go` — 权限码枚举 | 20 个 const + AllPermissions() |
| 0.3 | 实现 `roles.go` — 角色注册表 | SystemRoles map + RoleDefinition |
| 0.4 | 实现 `context.go` — AuthContext 注入/提取 | FromContext / WithContext |
| 0.5 | 单元测试（权限码、角色定义） | 编译时验证 + 无重复权限码验证 |

### Phase 1 — 数据层 + 引擎（2-3天）

| # | 任务 | 产出 |
|---|------|------|
| 1.1 | 编写迁移 SQL（taskAuth x3 + taskTenant x1） | dataMigrate/*.sql |
| 1.2 | taskAuth 新增 internal API: `GET /api/internal/authz/user/{id}/roles` | 角色查询端点 |
| 1.3 | taskAuth 新增管理 API: 角色 CRUD / 指派 / 撤销 | platform role 管理 |
| 1.4 | taskTenantService 新增 internal API: `GET /api/internal/tenant/memberships` | 成员角色查询 |
| 1.5 | taskTenantService 新增管理 API: member role 修改 | tenant role 管理 |
| 1.6 | 实现 `engine.go` — Phase 1 优先级引擎 | HasRole / HasPermission (Priority) |
| 1.7 | 实现 `client.go` — 内部 HTTP 客户端 | FetchUserRoles |
| 1.8 | 实现 `http.go` — RequireRole / RequireTenantRole | HTTP 中间件 |
| 1.9 | 集成测试（engine + HTTP middleware） | |

### Phase 2 — 缓存 + 网关改造（1-2天）

| # | 任务 | 产出 |
|---|------|------|
| 2.1 | 实现 `cache.go` — Redis 缓存层 | GetCachedUserRoles / Invalidate |
| 2.2 | 实现 `cache_disabled.go` — 无 Redis 回退 | 内存 LRU |
| 2.3 | 修改 APISIX forward-auth 配置 | 注入 X-User-Roles / X-Tenant-Roles |
| 2.4 | taskAuth gateway_forward_auth.go — 返回 roles + permissions | |
| 2.5 | 压力测试（缓存命中率 > 99%，延迟 < 1ms） | |

### Phase 3 — 服务迁移（3-5天）

| # | 任务 | 目标服务 | 改造行数 |
|---|------|---------|---------|
| 3.1 | 替换 requireSuperuser → RequireRole(super_admin) | taskAuth | ~30行 |
| 3.2 | 替换 policy.go → RequireTenantRole | taskTenantService | ~80行 |
| 3.3 | 替换 system-admin 鉴权 | taskCloudService | ~50行 |
| 3.4 | 替换 system-admin 鉴权 | taskBill | ~40行 |
| 3.5 | workspace ACL → RequirePerm (Phase 2) | taskProjectService | ~80行 |
| 3.6 | hasWorkspaceAccess → RequirePerm (Phase 2) | taskTaskService | ~50行 |
| 3.7 | 按需接入 | taskAIComment, taskEvents | ~30行 |

### Phase 4 — 前端适配 + 审计（1-2天）

| # | 任务 | 产出 |
|---|------|------|
| 4.1 | usePermission() composable | 前端权限控制 |
| 4.2 | 按 roles 显隐管理菜单/按钮 | UI 适配 |
| 4.3 | 权限变更审计日志 | audit_log 记录 |
| 4.4 | E2E 测试（不同角色登录 → 不同 UI → 不同 API 返回） | |

### Phase 5 — 细粒度权限（Phase 2，可选）

| # | 任务 | 产出 |
|---|------|------|
| 5.1 | 升级 engine.go 支持 PermCode 精确判定 | HasPermission(code) |
| 5.2 | 权限码种子数据 + 回填 | |
| 5.3 | 租户自定义角色（tenant_role 表） | |
| 5.4 | 权限管理 UI（超管可编辑角色-权限关联） | |

---

## 9. 测试策略

### 9.1 单元测试

```go
// shareLib/authz/engine_test.go
func TestPriorityBasedHasRole_SuperAdmin(t *testing.T) {
    // super_admin (priority=100) 应该能通过所有角色检查
    ctx := withMockAuthContext("super_admin", "company1")
    engine := NewEngine(mockCache(), mockClient())
    
    assert.True(t, engine.HasRole(ctx, "u1", "c1", "tenant_admin"))  // 100 >= 100 → true
    assert.True(t, engine.HasRole(ctx, "u1", "c1", "member"))       // 100 >= 50 → true
}

func TestPriorityBasedHasRole_Member(t *testing.T) {
    ctx := withMockAuthContext("member", "company1")
    engine := NewEngine(mockCache(), mockClient())
    
    assert.True(t, engine.HasRole(ctx, "u1", "c1", "member"))        // 50 >= 50 → true
    assert.False(t, engine.HasRole(ctx, "u1", "c1", "tenant_admin")) // 50 < 100 → false
}

func TestCrossTenantIsolation(t *testing.T) {
    // member of company1 不能访问 company2
    ctx := withMockAuthContext("member", "company1")
    engine := NewEngine(mockCache(), mockClient())
    
    assert.False(t, engine.HasRole(ctx, "u1", "company2", "member"))
}
```

### 9.2 集成测试

```go
// taskTenantService/src/member_handlers_test.go  — 增强
func TestCompanyMembersList_RequiresTenantAdmin(t *testing.T) {
    // member 用户请求成员列表 → 403
    // tenant_admin 用户请求成员列表 → 200
}
```

### 9.3 E2E 测试矩阵

| 角色 | 公司 | 操作 | 预期 |
|------|------|------|------|
| super_admin | — | 查看任意租户列表 | 200 |
| super_admin | — | 管理员工 | 200 |
| employee | — | 模拟用户登录 | 200 |
| employee | — | 修改系统配置 | 403 |
| tenant_admin | A | 管理公司A成员 | 200 |
| tenant_admin | A | 查看公司B项目 | 403 (租户隔离) |
| member | A | 创建任务 | 200 |
| member | A | 删除项目 | 403 |
| member | A | 邀请成员 | 403 |

---

## 10. 与现有架构的关系

- **v59 架构文件保持不变**（服务拓扑、数据流与本设计一致）
- 本设计 = v59 的**深化实施文档**，聚焦 RBAC 框架内部引擎架构与集成方案
- 若新增 `shareLib/authz` 被视作新架构组件，可创建 v60 架构文件，否则 v59 已覆盖
- v58 (email-invite-enhancement) 的 `assigned_role` 字段与本设计的 role 模型直接对齐

---

## 11. 风险与缓解

| 风险 | 概率 | 缓解 |
|------|------|------|
| 迁移期间旧鉴权与新鉴权不一致 | 中 | 双轨运行：新检查写在旧检查之后，记录 diff 日志 |
| 缓存延迟导致权限未即时生效 | 中 | 角色变更立即失效缓存 + Redis Pub/Sub 广播 |
| 跨服务内部调用增加延迟 | 低 | 网关层一次性注入 + Redis 缓存，首次 ~5ms，后续 <1ms |
| Phase 2 权限码膨胀难以维护 | 低 | 权限码仅 ~20 个，有 const 编译检查 + CI lint |
