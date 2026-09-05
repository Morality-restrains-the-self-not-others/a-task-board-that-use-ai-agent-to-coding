# RBAC 身份角色权限体系 — 完整设计方案

- **状态**: 🎯 target（设计阶段 v3，待审批）
- **迭代**: rbac-identity-role-permission-system
- **作者**: claude
- **设计日期**: 2026-08-04 14:00
- **上次修订**: 2026-08-04 (v3 — 集中授权架构 + 组继承 + 工作空间 ACL 迁移)
- **替代**: `2026-08-04-identity-role-permission-system-design.md` (v1 高层设计)、`2026-08-04-rbac-framework-implementation-design.md` (v2 框架实施)
- **类型**: 架构设计 + 数据模型 + 引擎设计 + 集成方案 + 实施计划

---

## 1. 需求与概念模型

### 1.1 核心概念

建立**两层四角色**的身份权限体系，明确 SaaS 使用方与平台维护方的边界：

```
┌───────────────────────────────────────────────────────┐
│               平台层 (Platform — 维护 SaaS)              │
│  SuperAdmin (超管) ◆────权限继承───◆ Employee (员工)     │
├───────────────────────────────────────────────────────┤
│               租户层 (Tenant — 使用 SaaS)               │
│  TenantAdmin (租户管理员) ◆──权限继承──◆ Member (成员)   │
│                                                        │
│  ┌─── Company A ───┐ ┌─── Company B ───┐               │
│  │  组1(role)→继承  │ │                │  ← 数据隔离    │
│  │  组2(role)→继承  │ │                │               │
│  │  ws1(ACL)→映射   │ │                │               │
│  └─────────────────┘ └────────────────┘               │
└───────────────────────────────────────────────────────┘
```

### 1.2 四角色定义

| 角色 | 层 | 典型用户 | 权限范围 |
|------|-----|---------|---------|
| **SuperAdmin** | Platform | CTO、平台负责人 | 全局管理：系统配置、员工管理、跨租户审计 |
| **Employee** | Platform | 运维、客服 | 日常运维：租户支持、模拟登录、账单审计 |
| **TenantAdmin** | Tenant | 公司管理员、创建者 | 租户全权：成员/组/项目/云资源/计费管理 |
| **Member** | Tenant | 普通用户 | 协作使用：查看+创建任务、查看项目/账单/成员 |

### 1.3 设计原则

| P# | 原则 | 实现 |
|----|------|------|
| P1 | User ≤ TenantAdmin（权限继承） | 角色优先级: member:50 < tenant_admin:100 |
| P2 | Employee ≤ SuperAdmin | 角色优先级: employee:50 < super_admin:100 |
| P3 | 用户 → 租户，员工 → 平台 | `auth_user_role.company_id=NULL` (平台) vs `NOT NULL` (租户) |
| P4 | 租户隔离 | 所有 tenant API + DB 查询强制 `company_id` 过滤 |
| P5 | Go-First | 零 Python 接口；taskAuth 扩展 + shareLib 薄客户端 |
| P6 | 组继承角色 | `tenant_company_group` 关联角色 → 成员自动继承 |
| P7 | 渐进迁移 | Phase 1: 优先级鉴权 + 组继承，兼容 is_admin/is_superuser |

---

## 2. 授权架构：taskAuth 集中 PDP

### 2.1 架构决策：集中授权 vs 共享库

| 方案 | 描述 | 优劣 |
|------|------|------|
| A: shareLib 重库 | 每个服务 `import shareLib/authz`，内嵌完整引擎 | 紧耦合，升级需全部重建 |
| **B: taskAuth PDP + 薄客户端** ✅ | taskAuth 是授权决策中心，shareLib 只是 HTTP wrapper + Context reader | 松耦合，规则集中管理，缓存统一 |

### 2.2 v60 目标拓扑

```
                     ┌──────────────────────┐
                     │   APISIX Gateway      │
                     │   forward-auth        │
                     │   → X-User-Id         │
                     │   → X-User-Roles      │  ← 一次查询，全链路复用
                     │   → X-Tenant-Memberships │
                     └──────┬───────────────┘
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │taskCloud │ │ taskBill │ │taskTenant│  ← 读 Context 判定 O(1)
        │RequireRole│ │RequireRole│ │RequireRole│
        └──────────┘ └──────────┘ └──────────┘
              │             │             │
              │  (cache miss 时回调)        │
              ▼             ▼             ▼
        ┌──────────────────────────────────────┐
        │          taskAuth (PDP)               │
        │  ┌────────────────────────────────┐  │
        │  │ /api/internal/authz/check       │  │  集中授权 API
        │  │ POST {user, company, perm}      │  │
        │  │ → {allowed: true/false}         │  │
        │  └────────────────────────────────┘  │
        │  ┌────────────────────────────────┐  │
        │  │ Redis: authz:{userID}:roles     │  │  统一缓存层
        │  │ TTL=5min, 角色变更即失效        │  │
        │  └────────────────────────────────┘  │
        └──────────────────────────────────────┘
```

**关键变化（vs v59）**:
- ❌ 去掉独立的 `shareLib/authz` 引擎（不再内嵌权限逻辑到每个服务）
- ✅ `shareLib/authz` 变为薄客户端：只做 `r.Context()` 读取 + HTTP 调用 taskAuth
- ✅ taskAuth 成为唯一授权决策点（Policy Decision Point）
- ✅ 网关 forward-auth 走 taskAuth 一次性注入角色/权限到 Header

### 2.3 shareLib/authz 客户端接口（薄封装）

```go
// shareLib/authz/authz.go  —— 每个 Go 服务 import 这个即可

// RequireRole 从请求 Context 读取角色并判定（零网络调用，网关已注入）
func RequireRole(w http.ResponseWriter, r *http.Request, roles ...string) error

// RequireTenantMember 校验用户属于指定租户
func RequireTenantMember(w http.ResponseWriter, r *http.Request, companyID string) error

// RequireTenantRole 校验用户在指定租户中具有指定角色
func RequireTenantRole(w http.ResponseWriter, r *http.Request, companyID string, roles ...string) error

// CheckPermission 调用 taskAuth 集中授权 API（带本地缓存，仅在 Context 无权限信息时触发）
func CheckPermission(ctx context.Context, userID, companyID string, perm PermCode) (bool, error)
```

> 关键：99% 的请求中网关已注入完整的角色/权限信息，`RequireRole` 是纯内存操作（~1µs）。仅在缓存均已失效的边缘情况下才回退到 taskAuth HTTP 调用。

---

## 3. 数据模型

### 3.1 表结构总览

```
taskAuth 库 (auth.db)                     taskTenant 库 (tenant.db)
───────────                               ───────────
auth_user                                 tenant_company
  │                                       tenant_company_member
  ├── auth_super_admin (保留，兼容)         │
  │                                       ├── tenant_member_role 🆕
  ├── auth_user_role 🆕                     │   (替代 is_admin)
  │   └── auth_role 🆕                     │
  │       └── auth_role_permission 🆕       └── tenant_company_group (已有)
  │           └── auth_permission 🆕             │
  │                                              └── tenant_company_group_member (已有)
  └── (is_superuser, is_staff 保留兼容)               │
                                                      └── tenant_group_role 🆕
                                                          (组 → 角色关联)
```

### 3.2 新增表 DDL

```sql
-- ═══ taskAuth 库 ═══

CREATE TABLE auth_role (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    level ENUM('platform','tenant') NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    is_system TINYINT NOT NULL DEFAULT 0,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE auth_permission (
    id VARCHAR(64) PRIMARY KEY,
    codename VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    level ENUM('platform','tenant') NOT NULL,
    description TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE auth_role_permission (
    id VARCHAR(64) PRIMARY KEY,
    role_id VARCHAR(64) NOT NULL REFERENCES auth_role(id),
    permission_id VARCHAR(64) NOT NULL REFERENCES auth_permission(id),
    UNIQUE(role_id, permission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE auth_user_role (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES auth_user(id),
    role_id VARCHAR(64) NOT NULL REFERENCES auth_role(id),
    company_id VARCHAR(64) DEFAULT NULL,  -- NULL=平台角色, NOT NULL=租户角色
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME DEFAULT NULL,
    UNIQUE(user_id, role_id, COALESCE(company_id, ''))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ═══ taskTenant 库 ═══

CREATE TABLE tenant_member_role (
    id VARCHAR(64) PRIMARY KEY,
    member_id VARCHAR(64) NOT NULL REFERENCES tenant_company_member(id),
    role_name VARCHAR(64) NOT NULL,  -- 'tenant_admin' | 'member'
    company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(member_id, role_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE tenant_group_role (
    id VARCHAR(64) PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL REFERENCES tenant_company_group(id),
    role_name VARCHAR(64) NOT NULL,  -- 'tenant_admin' | 'member'
    company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, role_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

> 架构变更：v59 将 `auth_user_role` 放在 taskAuth 但租户角色走 `tenant_member_role`。迭代后统一：**平台角色走 `auth_user_role`(company_id=NULL)，租户角色可通过 `auth_user_role`(company_id=xxx) 或 `tenant_member_role` 表达**。`tenant_member_role` 是冗余的快捷表，加速 `getEffectiveRoles(userID, companyID)` 查询。

---

## 4. 权限码目录

### 4.1 Go const 枚举（编译时安全）

```go
// shareLib/authz/permissions.go
package authz

type PermCode string

// ═══ Platform ═══
const (
    PermPlatformManage  PermCode = "platform:manage"   // 全局管理
    PermTenantAudit     PermCode = "tenant:audit"      // 跨租户审计
    PermUserImpersonate PermCode = "user:impersonate"  // 模拟登录
    PermSystemConfig    PermCode = "system:config"     // 系统配置
    PermBillingAudit    PermCode = "billing:audit"     // 账单审计
    PermEmployeeManage  PermCode = "employee:manage"   // 员工管理
)

// ═══ Tenant ═══
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
```

### 4.2 角色-权限映射矩阵

| 权限码 | super_admin | employee | tenant_admin | member |
|--------|:-----------:|:--------:|:------------:|:------:|
| `platform:manage` | ✅ | — | — | — |
| `tenant:audit` | ✅ | ✅ | — | — |
| `user:impersonate` | ✅ | ✅ | — | — |
| `system:config` | ✅ | — | — | — |
| `billing:audit` | ✅ | ✅ | — | — |
| `employee:manage` | ✅ | — | — | — |
| `company:manage` | — | — | ✅ | — |
| `company:view` | — | — | ✅ | ✅ |
| `member:manage` | — | — | ✅ | — |
| `member:view` | — | — | ✅ | ✅ |
| `group:manage` | — | — | ✅ | — |
| `project:manage` | — | — | ✅ | — |
| `project:view` | — | — | ✅ | ✅ |
| `task:manage` | — | — | ✅ | ✅ |
| `task:view` | — | — | ✅ | ✅ |
| `cloud:manage` | — | — | ✅ | — |
| `cloud:view` | — | — | ✅ | ✅ |
| `billing:manage` | — | — | ✅ | — |
| `billing:view` | — | — | ✅ | ✅ |
| `workspace:manage` | — | — | ✅ | — |

---

## 5. 角色组合与继承

### 5.1 多角色并集规则

用户可能同时拥有多个角色来源：

```
getEffectiveRoles(userID, companyID):
  roles = []

  // 1. 平台角色（直接指派）
  roles += auth_user_role WHERE company_id IS NULL AND user_id = userID

  // 2. 租户角色（直接指派给 member）
  roles += tenant_member_role WHERE member.user_id = userID AND member.company_id = companyID

  // 3. 组继承角色（通过 group 间接获得）
  for each group where user is member:
    roles += tenant_group_role WHERE group_id = group.id

  // 4. 创建者自动提升
  if userID == company.creator_id:
    roles += {role_name: "tenant_admin", priority: 100}

  // 5. 平台超管在租户内自动视为 tenant_admin（跨层穿透）
  if "super_admin" in roles:
    roles += {role_name: "tenant_admin", priority: 100}

  // 取最高优先级
  effectivePriority = max(r.priority for r in roles)
  return roles, effectivePriority
```

### 5.2 判定规则

```go
// Phase 1: 优先级判定
func HasRole(ctx, userID, companyID, requiredRole string) bool {
    _, priority := getEffectiveRoles(userID, companyID)
    return priority >= SystemRoles[requiredRole].Priority
}

// Phase 2: 权限码精确判定
func HasPermission(ctx, userID, companyID, perm PermCode) bool {
    roles, _ := getEffectiveRoles(userID, companyID)
    for _, r := range roles {
        if roleHasPermission(r, perm) {
            return true
        }
    }
    return false
}
```

### 5.3 特殊规则

| 场景 | 规则 |
|------|------|
| 公司创建者 | 自动获得 `tenant_admin`（priority=100），不可降级 |
| 平台超管进入租户 | 自动穿透为 `tenant_admin` |
| 被禁用成员 (`is_active=0`) | 所有角色失效 |
| 用户被删除 | 级联删除 `auth_user_role` + `tenant_member_role` |
| 角色过期 (`expires_at`) | 过期角色不计入有效角色 |
| 组被删除 | 级联删除 `tenant_group_role` |

---

## 6. 工作空间 ACL 迁移

### 6.1 现状

`project_workspace_accesses (workspace_id, user_id, group_id, permission VARCHAR(64) DEFAULT 'viewer')`

- 自由文本 permission（前后端不一致: `view`/`viewer`/`edit`/`admin`）
- ACL 同时支持 user 和 group

### 6.2 迁移映射

| 旧 ACL permission | 新 RBAC 角色 | 新 permission 码 |
|-------------------|-------------|-----------------|
| `viewer` / `view` | member | `task:view`, `project:view` |
| `edit` | member+ | `task:manage`, `project:view` |
| `admin` | tenant_admin (workspace-scoped) | `workspace:manage`, `task:manage`, `project:manage` |

### 6.3 迁移策略

```sql
-- Phase 2: 将 workspace admin ACL 提升为租户级 tenant_admin
-- （仅当 workspace ACL 的 permission='admin' 且 scope 是整个 workspace 时）
INSERT INTO tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
SELECT ... FROM project_workspace_accesses WHERE permission = 'admin';
```

> 注意：工作空间 ACL 比租户角色更细粒度。完全替代需要 workspace-scoped roles（Phase 2）。Phase 1 保留 ACL 表，新增 role 表与之共存。

---

## 7. API 设计

### 7.1 taskAuth 新增 API

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/system-admin/roles` | 角色列表 |
| POST | `/api/system-admin/users/{id}/roles` | 指派平台角色 |
| DELETE | `/api/system-admin/users/{id}/roles/{role_id}` | 撤销平台角色 |
| GET | `/api/system-admin/employees` | 员工列表 |
| POST | `/api/system-admin/employees` | 添加员工 |
| DELETE | `/api/system-admin/employees/{id}` | 移除员工 |
| GET | `/api/auth/user/roles` | 当前用户所有角色 |
| GET | `/api/auth/user/permissions` | 当前用户所有权限 |
| POST | `/api/internal/authz/check` | 内部授权判定 (PDP) |

### 7.2 taskTenantService 修改 API

| 方法 | 路径 | 变化 |
|------|------|------|
| PUT | `/api/tenant/{cid}/members/{mid}/role` | body: `{"role": "tenant_admin" \| "member"}` (替代 is_admin) |
| GET | `/api/tenant/{cid}/members` | 响应增加 `role`, `group_roles` 字段 |
| PUT | `/api/tenant/{cid}/groups/{gid}/role` 🆕 | 设置组角色 |
| POST | `/api/tenant/{cid}/invitations` | `role` 参数增加校验 |

### 7.3 Gateway forward-auth 增强

```
# APISIX forward-auth 响应新增 header:
X-User-Id: "user-xxx"
X-User-Roles: "super_admin,tenant_admin"        # 平台角色
X-Tenant-Memberships: "c1:tenant_admin,c2:member"  # 租户:角色对
X-User-Permissions: "task:view,task:manage,..."    # (Phase 2) 预计算权限码
```

### 7.4 Python 接口评估

**零新增 Python 接口。** 所有新 API 落 Go（taskAuth 10 个 + taskTenantService 4 个）。

---

## 8. 缓存策略

```
Redis Key                          Value                          TTL
──────────────────────────────────────────────────────────────────────
authz:user:{uid}:roles             ["super_admin","tenant_admin"]  300s
authz:user:{uid}:tenant_roles      {"c1":"tenant_admin","c2":"member"} 300s
authz:user:{uid}:perms:{cid}       ["task:view","task:manage",...] 300s
authz:role_defs                    {4个角色定义 JSON}               3600s
authz:perm_defs                    {20个权限码 JSON}                 3600s
```

**缓存失效**: 角色变更时发布 Kafka 事件 → taskEvents Consumer 精确失效对应 key + 可选 Redis Pub/Sub 广播到所有节点。

---

## 9. 各服务接入

### 9.1 接入清单

| 服务 | 改动 | 代码量 |
|------|------|--------|
| taskAuth | 新增 internal PDP API + 管理 API + DB 迁移 | ~400行 |
| taskTenantService | 替换 policy.go + 新增组角色 API + DB 迁移 | ~300行 |
| taskCloudService | 替换 system-admin handler 中的 hand-rolled 鉴权 | ~60行 |
| taskBill | 同上 | ~50行 |
| taskProjectService | 共享鉴权中间件 | ~40行 |
| taskTaskService | 共享鉴权中间件 | ~30行 |
| taskEvents | 新增 RoleChanged consumer（缓存失效） | ~50行 |
| APISIX | forward-auth 配置增强 | 配置改动 |
| taskFE | usePermission() + UI 元素显隐 | ~120行 |

### 9.2 代码改造示例

```go
// === 改造前 (taskCloudService system-admin handler) ===
func handleAdminCloudConfig(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-Id")
    var isSuperuser bool
    db.QueryRow("SELECT 1 FROM auth_super_admin WHERE user_id=?", userID).Scan(&isSuperuser)
    if !isSuperuser {
        writeJSON(w, 403, map[string]string{"error": "forbidden"})
        return
    }
    // ...业务逻辑
}

// === 改造后 ===
func handleAdminCloudConfig(w http.ResponseWriter, r *http.Request) {
    if err := authz.RequireRole(w, r, "super_admin", "employee"); err != nil {
        return  // 内部已写 403
    }
    // ...业务逻辑（完全相同）
}
```

---

## 10. 实施计划（精简版）

### Phase 1: 基础设施 + 角色 API + 服务迁移（5-7天）

| Day | 任务 | 产出 |
|-----|------|------|
| 1-2 | 迁移 SQL (taskAuth×3 + taskTenant×2) + 回填; create shareLib/authz 薄客户端 + PermCode const | DB ready, Go 包可 import |
| 2-3 | taskAuth PDP API (`/internal/authz/check` + `/system-admin/roles/*`) | 集中授权上线 |
| 3-4 | taskTenant 角色管理 API + 组角色 + 替换 policy.go | 租户级角色上线 |
| 4-5 | taskCloud + taskBill + taskProject + taskTask 迁移 | 全服务统一鉴权 |
| 5-6 | APISIX forward-auth 增强 + Redis 缓存 | 性能优化 |
| 6-7 | taskFE usePermission() + E2E 测试 | 前端适配 |

### Phase 2: 细粒度权限码（+3-5天，可选）

- 升级引擎支持 PermCode 精确判定
- 租户自定义角色 UI
- 权限审计日志

---

## 11. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 |
|---------|----------------|--------|--------------|
| 指派平台角色 | PlatformRoleAssigned | taskAuth AssignRole | 审计日志 + 缓存失效 |
| 撤销平台角色 | PlatformRoleRevoked | taskAuth RevokeRole | 审计日志 + session 失效 + 缓存失效 |
| 修改成员角色 | TenantMemberRoleChanged | taskTenant UpdateRole | SSE 推送 + 缓存失效 |
| 设置组角色 | TenantGroupRoleChanged | taskTenant UpdateGroupRole | 组内成员权限刷新 |
| 员工入职 | EmployeeOnboarded | taskAuth CreateEmployee | 通知 + 初始化 |
| 员工离职 | EmployeeOffboarded | taskAuth DeleteEmployee | 撤销所有角色 + 审计 |
| 公司创建 | CompanyCreated | taskTenant (已有) | creator → tenant_admin 角色（MQ Consumer） |

> 注意：`CompanyCreated` 事件已存在（Kafka），需要**新增 consumer** 监听该事件 → 自动给 creator 分配 tenant_admin 角色。这替代了现有 policy.go 中硬编码的 creator 检查。

---

## 12. 安全约束

| 约束 | 实现 |
|------|------|
| 租户隔离 | Context 中 company_id 校验；DB 查询强制 `WHERE company_id = ?` |
| 权限最小化 | 新用户默认 `member`；新员工默认 `employee` |
| 提权审计 | `assigned_by` + `assigned_at` + audit_log |
| Session 同步 | 角色变更 → Redis 缓存失效 → 下次请求自动刷新 |
| 过期角色 | `expires_at < NOW()` 自动排除 |
| 被禁成员 | `is_active=0` → 所有角色失效 |

---

## 13. 风险与缓解

| 风险 | 概率 | 缓解 |
|------|------|------|
| 迁移期间双轨鉴权不一致 | 中 | 新检查写在旧检查之后，diff 日志记录不一致 |
| taskAuth PDP 成为单点 | 低 | taskAuth 已有多实例；PDP 是纯读+缓存，延迟 <1ms |
| 缓存延迟 | 低 | 角色变更立即失效缓存，最大延迟 = Redis Pub/Sub 传播 ~10ms |
| 组角色变更影响大量用户 | 中 | 组角色变更时批量失效组内所有用户的缓存 key |

---

## 14. Domain Concept Inventory

| 概念 | 类型 | BC | 说明 |
|------|------|-----|------|
| User | Entity | Identity | 平台用户 |
| Role | Entity | Identity | 角色定义 (Name, Priority, Level, Permissions) |
| Permission | ValueObject | Identity | 权限码 (codename, resource_type, action) |
| UserRole | Aggregate | Identity | 用户-角色指派 |
| GroupRole | Aggregate | Tenant | 组-角色关联，成员继承 |
| MemberRole | Aggregate | Tenant | 租户成员角色（替代 is_admin） |
| Company | AggregateRoot | Tenant | 租户隔离边界 |
| CompanyMember | Entity | Tenant | 租户成员 |
| CompanyCreatorAutoPromoted | DomainEvent | Tenant | 创建者自动提升为 tenant_admin |
| PlatformRoleAssigned | DomainEvent | Identity | 平台角色指派 |
| PlatformRoleRevoked | DomainEvent | Identity | 平台角色撤销 |
| TenantMemberRoleChanged | DomainEvent | Tenant | 成员角色变更 |
| TenantGroupRoleChanged | DomainEvent | Tenant | 组角色变更 → 成员权限级联刷新 |

---

## 15. 架构变更影响

- **迭代版本**: v60 🎯 target
- **迭代名称**: rbac-identity-role-permission-system-v3
- **作者**: claude
- **设计日期**: 2026-08-04 14:00
- **基于**: v59（架构升级: shareLib 重引擎 → taskAuth 集中 PDP + 薄客户端）
- **新增文件**:
  - 🆕 `docs/architecture/v60-application-integration-20260804-1400-claude.puml`
  - 🆕 `docs/architecture/v60-application-integration-20260804-1400-claude.archimate`
  - 🆕 `docs/architecture/v60-application-integration-20260804-1400-claude.mermaid.md`
- **已有文件（未修改）**:
  - `docs/architecture/v57-application-integration-20260730-0229-claude.puml` (current)
  - `docs/architecture/v59-application-integration-20260804-1400-claude.*` (v2 target, 被 v60 替代)
- **变更明细（vs v59）**:
  - 🔄 [REFINED] shareLib/authz — 从重引擎（内嵌判定逻辑）→ 薄客户端（Context 读取 + PDP HTTP 回退）
  - 🟢 [NEW] taskAuth PDP — `/api/internal/authz/check` 集中授权判定端点
  - 🟢 [NEW] tenant_group_role — 组继承角色表
  - 🟢 [NEW] CompanyCreated consumer — 事件驱动 creator → tenant_admin 角色分配
  - 🟡 [MODIFIED] 缓存层 — 从 shareLib 内嵌 → taskAuth 集中 Redis 缓存
  - 🟡 [MODIFIED] 实施计划 — Phase 1 从 8-14天 → 5-7天

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| **Plateau v57** | Current — 二元 is_admin/is_superuser |
| **Plateau v59** | v2 Target — shareLib 重引擎 RBAC (被跳过) |
| **Plateau v60** | v3 Target — taskAuth 集中 PDP + 薄客户端 + 组继承 |
| **Gap** | 缺角色体系、缺集中授权、缺组权限继承 |
| **WorkPackage** | Phase 1: PDP+API+迁移 (5-7天); Phase 2: 权限码+自定义角色 (3-5天) |

---

## 附录 A: 与旧设计文档的关系

| 文档 | 状态 |
|------|------|
| `2026-08-04-identity-role-permission-system-design.md` (v1) | 📦 被本文替代 |
| `2026-08-04-rbac-framework-implementation-design.md` (v2) | 📦 被本文替代 |
| **本文 (v3)** | ✅ 当前唯一权威设计文档 |

## 附录 B: 与现有架构版本的兼容

- v56 (Django→Go final) — taskAuth 已为 auth 真源 ✅
- v57 (Current) — 所有服务纯 Go ✅
- v58 (email-invite-enhancement) — `assigned_role` 与本设计对齐 ✅
- v59 (v2 RBAC) — 被 v60 替代，架构文件保留但设计文档作废
