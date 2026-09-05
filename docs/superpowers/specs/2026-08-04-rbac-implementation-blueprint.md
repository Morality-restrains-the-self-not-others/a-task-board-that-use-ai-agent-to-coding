# RBAC 身份角色权限体系 — 完整实施方案 (v4)

- **状态**: 🎯 target（v4 终稿，可直接开工）
- **迭代**: rbac-identity-role-permission-system
- **作者**: claude
- **设计日期**: 2026-08-04
- **架构版本**: v60（组件拓扑不变，本文是 v60 的实施深化）
- **替代**: v1 高层设计、v2 框架实施、v3 完整设计 — 全部合并到本文

---

## 1. 设计总览

```
用户请求
  → APISIX (forward-auth → taskAuth PDP → 注入 X-User-Roles)
  → Go Service Handler
  → authz.RequireRole(w, r, "tenant_admin")  // 一行调用，读 Context
  → 业务逻辑
```

**四个核心角色**：super_admin(平台全权) > employee(平台运维) ‖ tenant_admin(租户全权) > member(租户使用)

**三条设计主线**：
1. 行级改动最小：每个 handler 加一行 `authz.RequireXxx()`
2. 零网络开销：网关 forward-auth 一次注入，后续纯内存读取
3. 事件驱动：角色变更 → Kafka → 缓存失效 + 审计日志

---

## 2. Go 代码骨架

### 2.1 包结构

```
shareLib/authz/                  ← 每个 Go 服务 go.mod 添加 replace → 本地路径
├── authz.go                     ← 包文档
├── permissions.go               ← PermCode 枚举 (const)
├── roles.go                     ← 系统角色定义
├── context.go                   ← AuthContext 注入/提取
├── middleware.go                ← RequireRole / RequireTenantRole / RequireTenantMember
├── client.go                    ← CheckPermission HTTP 回退 (调用 taskAuth PDP)
└── middleware_test.go           ← 单元测试
```

### 2.2 permissions.go — 权限码枚举

```go
package authz

// PermCode 权限码。编译时类型安全。
type PermCode string

// ════════════════ Platform ════════════════
const (
    PermPlatformManage   PermCode = "platform:manage"
    PermTenantAudit      PermCode = "tenant:audit"
    PermUserImpersonate  PermCode = "user:impersonate"
    PermSystemConfig     PermCode = "system:config"
    PermBillingAudit     PermCode = "billing:audit"
    PermEmployeeManage   PermCode = "employee:manage"
)

// ════════════════ Tenant ════════════════
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

// AllPermissions 返回所有权权限码（用于种子数据）。
func AllPermissions() []struct{ Code PermCode; Name, Resource, Action, Level string } {
    return []struct{...}{
        {PermPlatformManage, "平台全局管理", "platform", "manage", "platform"},
        // ... 20 entries total
    }
}
```

### 2.3 roles.go — 角色定义

```go
package authz

type RoleLevel string

const (
    LevelPlatform RoleLevel = "platform"
    LevelTenant   RoleLevel = "tenant"
)

type RoleDef struct {
    Name        string
    DisplayName string
    Level       RoleLevel
    Priority    int      // Phase 1: 优先级判定。越大越强。
    IsSystem    bool
}

// SystemRoles 系统内置角色注册表。
var SystemRoles = map[string]RoleDef{
    "super_admin":  {"super_admin", "超级管理员", LevelPlatform, 100, true},
    "employee":     {"employee", "平台员工", LevelPlatform, 50, true},
    "tenant_admin": {"tenant_admin", "租户管理员", LevelTenant, 100, true},
    "member":       {"member", "成员", LevelTenant, 50, true},
}

// HasRolePriority 判定用户是否具有 >= requiredRole 的优先级。
func HasRolePriority(userRoles []string, requiredRole string) bool {
    required, ok := SystemRoles[requiredRole]
    if !ok { return false }
    maxPriority := 0
    for _, r := range userRoles {
        if def, ok := SystemRoles[r]; ok && def.Priority > maxPriority {
            maxPriority = def.Priority
        }
    }
    return maxPriority >= required.Priority
}
```

### 2.4 context.go — AuthContext

```go
package authz

import (
    "context"
    "net/http"
    "strings"
)

type AuthContext struct {
    UserID       string
    PlatformRoles   []string            // ["super_admin"] or ["employee"]
    TenantRoles     map[string]string   // companyID → "tenant_admin" | "member"
    AllTenantRoles  map[string][]string // companyID → ["tenant_admin"] (含组继承)
}

type ctxKey struct{}

// FromContext 从请求 Context 提取鉴权信息。网关已注入。
func FromContext(r *http.Request) (*AuthContext, bool) {
    ac, ok := r.Context().Value(ctxKey{}).(*AuthContext)
    return ac, ok && ac != nil
}

// ParseFromHeaders 从网关注入的 Header 解析 AuthContext。
func ParseFromHeaders(r *http.Request) *AuthContext {
    ac := &AuthContext{
        UserID:      r.Header.Get("X-User-Id"),
        TenantRoles: make(map[string]string),
    }
    if v := r.Header.Get("X-User-Roles"); v != "" {
        ac.PlatformRoles = strings.Split(v, ",")
    }
    if v := r.Header.Get("X-Tenant-Memberships"); v != "" {
        for _, pair := range strings.Split(v, ",") {
            parts := strings.SplitN(pair, ":", 2)
            if len(parts) == 2 {
                ac.TenantRoles[parts[0]] = parts[1]
            }
        }
    }
    return ac
}
```

### 2.5 middleware.go — HTTP 中间件

```go
package authz

import (
    "encoding/json"
    "net/http"
)

// RequireRole 要求用户拥有指定的平台角色（至少一个）。
// 零网络调用 —— 从 r.Context() 读取网关注入的鉴权信息。
//
// Usage:
//   if err := authz.RequireRole(w, r, "super_admin", "employee"); err != nil {
//       return
//   }
func RequireRole(w http.ResponseWriter, r *http.Request, roles ...string) error {
    ac, ok := FromContext(r)
    if !ok || ac.UserID == "" {
        writeAuthError(w, 401, "未登录")
        return ErrUnauthenticated
    }
    for _, userRole := range ac.PlatformRoles {
        for _, required := range roles {
            if userRole == required || HasRolePriority(ac.PlatformRoles, required) {
                return nil
            }
        }
    }
    writeAuthError(w, 403, "权限不足，需要角色: "+strings.Join(roles, " 或 "))
    return ErrForbidden
}

// RequireTenantRole 要求用户在指定租户中拥有指定角色（至少一个）。
func RequireTenantRole(w http.ResponseWriter, r *http.Request, companyID string, roles ...string) error {
    ac, ok := FromContext(r)
    if !ok || ac.UserID == "" {
        writeAuthError(w, 401, "未登录")
        return ErrUnauthenticated
    }

    // 平台超管自动穿透
    if HasRolePriority(ac.PlatformRoles, "super_admin") {
        return nil
    }

    // 租户角色检查
    tenantRole, inTenant := ac.TenantRoles[companyID]
    if !inTenant {
        writeAuthError(w, 403, "您不属于该公司")
        return ErrForbidden
    }
    for _, required := range roles {
        if tenantRole == required || HasRolePriority([]string{tenantRole}, required) {
            return nil
        }
    }
    writeAuthError(w, 403, "权限不足")
    return ErrForbidden
}

// RequireTenantMember 要求用户属于指定租户（最低权限：member）。
func RequireTenantMember(w http.ResponseWriter, r *http.Request, companyID string) error {
    return RequireTenantRole(w, r, companyID, "member", "tenant_admin")
}

// writeAuthError 写 JSON 错误响应。
func writeAuthError(w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

var (
    ErrUnauthenticated = &AuthError{Code: 401, Message: "未登录"}
    ErrForbidden       = &AuthError{Code: 403, Message: "权限不足"}
)

type AuthError struct{ Code int; Message string }
func (e *AuthError) Error() string { return e.Message }
```

### 2.6 client.go — PDP HTTP 回退

```go
package authz

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "time"
)

// PDPClient 调用 taskAuth 集中授权 API（仅在 Context 无信息时回退）。
type PDPClient struct {
    BaseURL string        // "http://taskAuth:8003"
    HTTP    *http.Client  // timeout=2s
}

// CheckPermission 向 taskAuth PDP 发起授权判定。
func (c *PDPClient) CheckPermission(ctx context.Context, userID, companyID string, perm PermCode) (bool, error) {
    body, _ := json.Marshal(map[string]string{
        "user_id":    userID,
        "company_id": companyID,
        "permission": string(perm),
    })
    req, _ := http.NewRequestWithContext(ctx, "POST",
        c.BaseURL+"/api/internal/auth/permission-check/", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.HTTP.Do(req)
    if err != nil { return false, err }
    defer resp.Body.Close()

    var result struct {
        Allowed bool   `json:"allowed"`
        Reason  string `json:"reason,omitempty"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    return result.Allowed, nil
}
```

---

## 3. API 路径合规审计

> 规范要求：`/api/${serviceName}/${funcName}/${key1}/${value1}/${key2}/${value2}/...`
> 存量路径不强制迁移，新增端点必须合规。

### 3.1 违规总览

| # | 原设计路径 | 违规类型 | 修正后路径 |
|---|-----------|---------|-----------|
| 1 | `/api/system-admin/roles` | `system-admin` 非 serviceName | `/api/auth/roles/` |
| 2 | `/api/system-admin/users/{id}/roles` | 深度嵌套 + 位置参数 | `/api/auth/user-roles/user_id/{uid}/` |
| 3 | `/api/system-admin/users/{id}/roles/{role}` | 深度嵌套 | `/api/auth/user-roles/user_id/{uid}/role_name/{name}/` |
| 4 | `/api/system-admin/roles/{id}/users` | 深度嵌套 | `/api/auth/role-users/role_name/{name}/` |
| 5 | `/api/system-admin/employees` | `system-admin` 前缀 | `/api/auth/employees/` |
| 6 | `/api/system-admin/employees` | 同上 | `/api/auth/employees/user_id/{uid}/` |
| 7 | `/api/system-admin/employees/{id}` | 位置参数 | `/api/auth/employees/user_id/{uid}/` |
| 8 | `/api/auth/user/roles` | `user/roles` 位置嵌套 | `/api/auth/user-roles/` |
| 9 | `/api/auth/user/permissions` | `user/permissions` 位置嵌套 | `/api/auth/user-permissions/` |
| 10 | `/api/internal/authz/check` | `authz` 非有效 serviceName + `check` 应为 funcName | `/api/internal/auth/permission-check/` |
| 11 | `/api/tenant/{cid}/members/{mid}/role` | 深度嵌套 + 位置参数 | `/api/tenant/member-role/company_id/{cid}/member_id/{mid}/` |
| 12 | `/api/tenant/{cid}/groups/{gid}/role` | 深度嵌套 + 位置参数 | `/api/tenant/group-role/company_id/{cid}/group_id/{gid}/` |

> ⚠️ 12 个新增 API 中 **12 个全部违规**（100%）。根因：直接沿用了存量 `/api/system-admin/` 和 `/api/tenant/{cid}/...` 的旧模式。修正后全部合规。

### 3.2 修正后完整 API 契约

#### taskAuth 新增 API (serviceName=`auth` / `internal`)

#### POST /api/internal/auth/permission-check/

PDP 集中授权判定端点。由 shareLib 客户端或网关调用。

```
Request:  POST /api/internal/auth/permission-check/
Body:     {"user_id": "u1", "company_id": "c1", "permission": "task:manage"}
Response: {"allowed": true}
          {"allowed": false, "reason": "member role lacks 'task:manage' permission"}
```

#### GET /api/auth/roles/

```
Response: {
  "roles": [
    {"name": "super_admin", "display_name": "超级管理员", "level": "platform", "priority": 100},
    {"name": "employee", "display_name": "平台员工", "level": "platform", "priority": 50},
    {"name": "tenant_admin", "display_name": "租户管理员", "level": "tenant", "priority": 100},
    {"name": "member", "display_name": "成员", "level": "tenant", "priority": 50}
  ]
}
```

#### POST /api/auth/user-roles/user_id/{user_id}/

```
Request:  {"role": "super_admin"}
Response: 201 {"id": "aur-xxx", "user_id": "u1", "role": "super_admin", "assigned_by": "admin1"}
Errors:   400 role 不存在 / 409 已存在 / 403 不是 super_admin
```

#### DELETE /api/auth/user-roles/user_id/{user_id}/role_name/{role_name}/

```
Response: 204
Errors:   404 无此指派 / 403 不能移除最后一个 super_admin
```

#### GET /api/auth/employees/

```
Response: {
  "employees": [
    {"user_id": "u1", "email": "staff@x.com", "phone": "...", "roles": ["employee"], "assigned_at": "..."}
  ]
}
```

#### POST /api/auth/employees/user_id/{user_id}/

```
Request:  {"user_id": "u1"}
Response: 201
Side effect: → Kafka PlatformRoleAssigned
```

#### DELETE /api/auth/employees/user_id/{user_id}/

```
Response: 204
Side effect: → Kafka PlatformRoleRevoked
```

#### GET /api/auth/user-roles/

```
Response: {
  "platform_roles": ["super_admin"],
  "tenant_roles": {"c1": "tenant_admin", "c2": "member"},
  "all_roles": ["super_admin", "tenant_admin", "member"]
}
```

#### GET /api/auth/user-permissions/

```
Response: {
  "permissions": ["platform:manage", "tenant:audit", ..., "task:view", "cloud:view"]
}
```

### 3.2 taskTenantService 新增/修改 API

#### PUT /api/tenant/member-role/company_id/{cid}/member_id/{mid}/

替代现有的 `update_role`。

```
Request:  {"role": "tenant_admin"}
Response: 200 {"member_id": "m1", "role": "tenant_admin"}
Errors:   400 role 非法 / 403 不是 tenant_admin / 422 不能修改创建者角色
Side effect: → Kafka TenantMemberRoleChanged
```

#### GET /api/tenant/{cid}/members（增强）

```
Response: {
  "members": [
    {
      "id": "m1",
      "user_id": "u1",
      "role": "tenant_admin",         // ← 新增
      "group_roles": ["member"],       // ← 新增：通过组继承的
      "is_admin": true,               // 保留兼容
      ...
    }
  ]
}
```

#### PUT /api/tenant/group-role/company_id/{cid}/group_id/{gid}/（新增）

```
Request:  {"role": "member"}  // 组内成员全部获得此角色
Response: 201
Side effect: → Kafka TenantGroupRoleChanged
```

---

## 4. 全端点 → 权限映射（合规路径）

### 4.1 平台管理 API

> 新增端点（🆕）遵循 `/api/auth/...` 规范；存量端点（📦）保留原路径。

| 端点 | 方法 | 状态 | 所需角色 | 权限码 |
|------|------|------|---------|--------|
| `/api/auth/roles/` | GET | 🆕 | super_admin, employee | — |
| `/api/auth/user-roles/user_id/{uid}/` | POST | 🆕 | super_admin | `employee:manage` |
| `/api/auth/user-roles/user_id/{uid}/role_name/{name}/` | DELETE | 🆕 | super_admin | `employee:manage` |
| `/api/auth/role-users/role_name/{name}/` | GET | 🆕 | super_admin | `employee:manage` |
| `/api/auth/employees/` | GET | 🆕 | super_admin | `employee:manage` |
| `/api/auth/employees/user_id/{uid}/` | POST | 🆕 | super_admin | `employee:manage` |
| `/api/auth/employees/user_id/{uid}/` | DELETE | 🆕 | super_admin | `employee:manage` |
| `/api/auth/user-roles/` | GET | 🆕 | (当前用户) | — |
| `/api/auth/user-permissions/` | GET | 🆕 | (当前用户) | — |
| `/api/system-admin/users` | GET | 📦 | super_admin, employee | `tenant:audit` |
| `/api/system-admin/users` | POST | 📦 | super_admin | `employee:manage` |
| `/api/system-admin/resource-pricing/*` | * | 📦 | super_admin, employee | `billing:audit` |
| `/api/system-admin/grant-points` | POST | 📦 | super_admin | `billing:audit` |
| `/api/system-admin/refund-applications` | GET | 📦 | super_admin, employee | `billing:audit` |
| `/api/system-admin/order-records` | GET | 📦 | super_admin, employee | `billing:audit` |
| `/api/system-admin/tenant-options` | GET | 📦 | super_admin, employee | `tenant:audit` |
| `/api/system-admin/cloud/*` | * | 📦 | super_admin, employee | `platform:manage` |
| `/api/system-admin/*` | * | 📦 | super_admin | `platform:manage` |

### 4.2 租户 API

> 新增端点（🆕）遵循 `/api/tenant/{funcName}/key/value/...` 规范；存量端点（📦）保留原路径。

| 端点 | 方法 | 状态 | 所需角色 | 权限码 |
|------|------|------|---------|--------|
| `/api/tenant/member-role/company_id/{cid}/member_id/{mid}/` | PUT | 🆕 | tenant_admin | `member:manage` |
| `/api/tenant/group-role/company_id/{cid}/group_id/{gid}/` | PUT | 🆕 | tenant_admin | `group:manage` |
| `/api/tenant/{cid}/members` | GET | 📦 | member, tenant_admin | `member:view` |
| `/api/tenant/{cid}/members/{mid}/toggle-status` | POST | 📦 | tenant_admin | `member:manage` |
| `/api/tenant/{cid}/members/{mid}` | DELETE | 📦 | tenant_admin | `member:manage` |
| `/api/tenant/{cid}/company` | GET/PUT | 📦 | member+ / tenant_admin | `company:view`/`manage` |
| `/api/tenant/{cid}/invitations` | GET/POST | 📦 | tenant_admin | `member:view`/`manage` |
| `/api/tenant/{cid}/invitations/{iid}/resend` | POST | 📦 | tenant_admin | `member:manage` |
| `/api/tenant/{cid}/groups` | GET/POST | 📦 | member+ / tenant_admin | `company:view`/`group:manage` |
| `/api/tenant/{cid}/groups/{gid}` | PUT/DELETE | 📦 | tenant_admin | `group:manage` |
| `/api/tenant/{cid}/groups/{gid}/members` | POST/DELETE | 📦 | tenant_admin | `group:manage` |
| `/api/tenant/{cid}/projects` | GET/POST | 📦 | member+ / tenant_admin | `project:view`/`manage` |
| `/api/tenant/{cid}/todos/*` | GET/POST | 📦 | member+ | `task:view`/`manage` |
| `/api/tenant/{cid}/todos/*` | DELETE | 📦 | tenant_admin | `task:manage` |
| `/api/tenant/{cid}/cloud/*` | GET | 📦 | member+ | `cloud:view` |
| `/api/tenant/{cid}/cloud/*` | POST/PUT/DELETE | tenant_admin | `cloud:manage` |
| `/api/tenant/{cid}/billing/*` | GET | member, tenant_admin | `billing:view` |
| `/api/tenant/{cid}/billing/*` | POST | tenant_admin | `billing:manage` |
| `/api/tenant/{cid}/workspace/*` | PUT | tenant_admin | `workspace:manage` |

---

## 5. 可执行数据库迁移

### 5.1 taskAuth: 027_rbac_roles.sql

```sql
-- ═══════════════════════════════════════════
-- taskAuth RBAC: 角色 + 权限 + 关联表
-- ═══════════════════════════════════════════

CREATE TABLE IF NOT EXISTS auth_role (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    level ENUM('platform','tenant') NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    is_system TINYINT NOT NULL DEFAULT 0,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_permission (
    id VARCHAR(64) PRIMARY KEY,
    codename VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    level ENUM('platform','tenant') NOT NULL,
    description TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_role_permission (
    id VARCHAR(64) PRIMARY KEY,
    role_id VARCHAR(64) NOT NULL,
    permission_id VARCHAR(64) NOT NULL,
    UNIQUE KEY uk_rp (role_id, permission_id),
    CONSTRAINT fk_rp_role FOREIGN KEY (role_id) REFERENCES auth_role(id),
    CONSTRAINT fk_rp_perm FOREIGN KEY (permission_id) REFERENCES auth_permission(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_user_role (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    role_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) DEFAULT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME DEFAULT NULL,
    UNIQUE KEY uk_urc (user_id, role_id, COALESCE(company_id, '')),
    CONSTRAINT fk_aur_user FOREIGN KEY (user_id) REFERENCES auth_user(id),
    CONSTRAINT fk_aur_role FOREIGN KEY (role_id) REFERENCES auth_role(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 种子: 4 系统角色
INSERT IGNORE INTO auth_role (id, name, display_name, level, priority, is_system, description) VALUES
('role-super-admin',  'super_admin',  '超级管理员', 'platform', 100, 1, '平台全局管理、员工管理、系统配置'),
('role-employee',     'employee',     '平台员工',   'platform', 50,  1, '日常运维、客服、租户支持'),
('role-tenant-admin', 'tenant_admin', '租户管理员', 'tenant',   100, 1, '公司成员/组/项目/云资源/计费管理'),
('role-member',       'member',       '成员',       'tenant',   50,  1, '协作使用 SaaS 功能');

-- 种子: 20 权限码
INSERT IGNORE INTO auth_permission (id, codename, name, resource_type, action, level) VALUES
('perm-plat-manage',   'platform:manage',   '平台全局管理',   'platform', 'manage',      'platform'),
('perm-tenant-audit',  'tenant:audit',      '跨租户审计',     'platform', 'audit',       'platform'),
('perm-impersonate',   'user:impersonate',  '模拟用户登录',   'user',     'impersonate', 'platform'),
('perm-sys-config',    'system:config',     '系统配置管理',   'system',   'manage',      'platform'),
('perm-bill-audit',    'billing:audit',     '跨租户账单审计', 'billing',  'audit',       'platform'),
('perm-emp-manage',    'employee:manage',   '员工管理',       'employee', 'manage',      'platform'),
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

-- 种子: 角色→权限关联
-- super_admin: 所有 platform 权限
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-sa-', id), 'role-super-admin', id FROM auth_permission WHERE level='platform';

-- employee: platform 中除 platform:manage, system:config, employee:manage
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-emp-', id), 'role-employee', id FROM auth_permission
WHERE level='platform' AND codename NOT IN ('platform:manage','system:config','employee:manage');

-- tenant_admin: 所有 tenant 权限
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-ta-', id), 'role-tenant-admin', id FROM auth_permission WHERE level='tenant';

-- member: 所有 view 权限 + task:manage
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-mb-', id), 'role-member', id FROM auth_permission WHERE level='tenant' AND action='view';
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
VALUES ('rp-mb-task-manage', 'role-member', 'perm-task-manage');
```

### 5.2 taskAuth: 028_rbac_backfill.sql

```sql
-- 回填现有超管
INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-sa-', id), id, 'role-super-admin', NULL, 'system', NOW()
FROM auth_user WHERE is_superuser = 1;

-- 回填 auth_super_admin 表
INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-sa2-', user_id), user_id, 'role-super-admin', NULL, 'system', NOW()
FROM auth_super_admin
WHERE user_id NOT IN (SELECT user_id FROM auth_user_role WHERE role_id='role-super-admin');

-- 回填 is_staff → employee
INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-emp-', id), id, 'role-employee', NULL, 'system', NOW()
FROM auth_user WHERE is_staff = 1 AND is_superuser = 0;
```

### 5.3 taskTenantService: 017_member_role.sql

```sql
CREATE TABLE IF NOT EXISTS tenant_member_role (
    id VARCHAR(64) PRIMARY KEY,
    member_id VARCHAR(64) NOT NULL,
    role_name VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_member_role (member_id, role_name),
    CONSTRAINT fk_tmr_member FOREIGN KEY (member_id) REFERENCES tenant_company_member(id),
    INDEX idx_company (company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tenant_group_role (
    id VARCHAR(64) PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL,
    role_name VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_group_role (group_id, role_name),
    CONSTRAINT fk_tgr_group FOREIGN KEY (group_id) REFERENCES tenant_company_group(id),
    INDEX idx_company (company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 回填: is_admin=1 → tenant_admin
INSERT IGNORE INTO tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-', id), id, 'tenant_admin', company_id, 'system', NOW()
FROM tenant_company_member WHERE is_admin = 1;

-- 回填: creator → tenant_admin
INSERT IGNORE INTO tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-cr-', m.id), m.id, 'tenant_admin', m.company_id, 'system', NOW()
FROM tenant_company_member m
JOIN tenant_company c ON c.id = m.company_id
WHERE m.user_id = c.creator_id AND m.id NOT IN (SELECT member_id FROM tenant_member_role WHERE role_name='tenant_admin');
```

### 5.4 回滚 SQL

```sql
-- ═══ taskAuth 回滚 ═══
DROP TABLE IF EXISTS auth_user_role;
DROP TABLE IF EXISTS auth_role_permission;
DROP TABLE IF EXISTS auth_permission;
DROP TABLE IF EXISTS auth_role;

-- ═══ taskTenantService 回滚 ═══
DROP TABLE IF EXISTS tenant_group_role;
DROP TABLE IF EXISTS tenant_member_role;
```

---

## 6. 网关配置变更

```yaml
# APISIX routes.yaml 片段 — forward-auth 增强

# taskAuth forward-auth 插件配置
plugins:
  forward-auth:
    uri: "http://taskAuth:8003/api/internal/auth/permission-check/"
    request_headers: ["X-User-Id", "Cookie"]
    upstream_headers: ["X-User-Roles", "X-Tenant-Memberships", "X-User-Permissions"]
    client_headers: ["X-Forwarded-For"]
```

taskAuth forward-auth handler 返回：

```json
{
  "status": "allowed",
  "headers": {
    "X-User-Id": "user-xxx",
    "X-User-Roles": "super_admin",
    "X-Tenant-Memberships": "c1:tenant_admin,c2:member"
  }
}
```

---

## 7. 服务迁移清单

每个 handler 的具体改造：

### 7.1 taskAuth (自身改造)

| 文件 | 行 | 当前代码 | 改为 |
|------|-----|---------|------|
| `registration_invite_handlers.go:19-33` | `requireSuperuser()` → `db.QueryRow(...)` | `authz.RequireRole(w, r, "super_admin")` |
| `handlers_system_admin.go:20,74` | `requireSuperuser()` | `authz.RequireRole(w, r, "super_admin")` |
| `system_feature_policy.go:115-168` | 自行校验 | `authz.RequireRole(w, r, "super_admin")` |

### 7.2 taskTenantService

| 文件 | 行 | 当前代码 | 改为 |
|------|-----|---------|------|
| `policy.go:70-97` | `requireCompanyAdmin()` 查 DB | `authz.RequireTenantRole(w, r, companyID, "tenant_admin")` |
| `policy.go:99-118` | `checkCompanyAdmin()` 查 DB | `authz.RequireTenantRole(w, r, companyID, "tenant_admin")` |
| `policy.go:120-136` | `requireCompanyMember()` 查 DB | `authz.RequireTenantMember(w, r, companyID)` |
| `policy.go:138-144` | `isCreator()` 查 company_store | 删除, 由 CompanyCreated consumer 负责角色分配 |
| `member_handlers.go:50-55` | 手算 role 字符串 | 从 `ac.TenantRoles[companyID]` 读取 |
| `member_handlers.go:110-158` | `handleUpdateRole()` 改 is_admin | 改为写 `tenant_member_role` + 发事件 |

### 7.3 taskCloudService

| 文件 | 当前代码 | 改为 |
|------|---------|------|
| system-admin handlers | `checkSuperAdmin()` 查 DB | `authz.RequireRole(w, r, "super_admin", "employee")` |

### 7.4 taskBill

| 文件 | 当前代码 | 改为 |
|------|---------|------|
| system-admin handlers | 自行校验 | `authz.RequireRole(w, r, "super_admin", "employee")` |

### 7.5 taskProjectService

| 文件 | 当前代码 | 改为 |
|------|---------|------|
| `workspace_access_handlers.go` | 自由文本 permission | Phase 2 迁移到 role 模型 |

### 7.6 taskTaskService

| 文件 | 当前代码 | 改为 |
|------|---------|------|
| `task_handlers.go:35,57,187,200` | `hasWorkspaceAccess()` | `authz.RequireTenantMember(w, r, companyID)` |

---

## 8. 测试矩阵

### 8.1 单元测试 (shareLib/authz/middleware_test.go)

```go
func TestRequireRole_SuperAdmin(t *testing.T) {
    r := withAuthContext("u1", []string{"super_admin"}, nil)
    err := RequireRole(httptest.NewRecorder(), r, "super_admin")
    assert.Nil(t, err)
}

func TestRequireRole_Employee(t *testing.T) {
    r := withAuthContext("u1", []string{"employee"}, nil)
    err := RequireRole(httptest.NewRecorder(), r, "super_admin")
    assert.Equal(t, ErrForbidden, err) // employee < super_admin
}

func TestRequireTenantRole_CrossTenant(t *testing.T) {
    r := withAuthContext("u1", nil, map[string]string{"c1": "member"})
    err := RequireTenantRole(httptest.NewRecorder(), r, "c2", "member")
    assert.Equal(t, ErrForbidden, err) // 租户隔离
}

func TestRequireTenantRole_SuperAdminPenetrates(t *testing.T) {
    r := withAuthContext("u1", []string{"super_admin"}, nil)
    err := RequireTenantRole(httptest.NewRecorder(), r, "c1", "tenant_admin")
    assert.Nil(t, err) // 超管穿透
}

func TestHasRolePriority_MemberCannotBeAdmin(t *testing.T) {
    assert.False(t, HasRolePriority([]string{"member"}, "tenant_admin"))
}

func TestHasRolePriority_AdminCanBeMember(t *testing.T) {
    assert.True(t, HasRolePriority([]string{"tenant_admin"}, "member"))
}
```

### 8.2 E2E 测试矩阵

| # | 角色 | 请求（合规路径） | 预期 |
|---|------|------|------|
| 1 | super_admin | GET /api/auth/roles/ | 200 |
| 2 | employee | GET /api/auth/roles/ | 200 |
| 3 | employee | POST /api/auth/employees/user_id/u1/ | 403 |
| 4 | member(c1) | GET /api/tenant/c1/members (📦存量) | 200 |
| 5 | member(c1) | PUT /api/tenant/member-role/company_id/c1/member_id/m1/ | 403 |
| 6 | member(c1) | GET /api/tenant/c2/members (📦存量) | 403 (跨租户) |
| 7 | tenant_admin(c1) | PUT /api/tenant/member-role/company_id/c1/member_id/m1/ | 200 |
| 8 | tenant_admin(c1) | GET /api/tenant/c2/projects (📦存量) | 403 (跨租户) |
| 9 | super_admin | PUT /api/tenant/member-role/company_id/c1/member_id/m1/ | 200 (穿透) |
| 10 | 未登录 | GET /api/auth/user-roles/ | 401 |

---

## 9. 事件流

```
CompanyCreated (已有)
  → taskEvents Consumer → POST /api/internal/tenant/{cid}/members (creator → tenant_admin)

PlatformRoleAssigned (新增)
  → taskEvents Consumer → Redis DEL authz:{userID}:roles → 缓存失效

PlatformRoleRevoked (新增)
  → taskEvents Consumer → Redis DEL authz:{userID}:roles → 缓存失效 + session 失效

TenantMemberRoleChanged (新增)
  → taskEvents Consumer → Redis DEL authz:{userID}:tenant_roles

TenantGroupRoleChanged (新增)
  → taskEvents Consumer → 批量 Redis DEL authz:{memberID}:tenant_roles (组内所有成员)
```

---

## 10. 实施计划（最终版）

| Day | 任务 | 产出 | 验证 |
|-----|------|------|------|
| 1 | shareLib/authz: permissions.go + roles.go + context.go + middleware.go | Go 包可 import | `go test ./...` |
| 1 | 迁移 SQL: 027_rbac_roles.sql + 028_rbac_backfill.sql + 017_member_role.sql | DB 就绪 | 本地跑迁移 |
| 2 | taskAuth: PDP API + 角色管理 API + forward-auth 增强 | /api/internal/auth/permission-check/ 可用 | curl 测试 |
| 2 | taskAuth: 自身鉴权替换 (requireSuperuser → RequireRole) | | 现有测试通过 |
| 3 | taskTenant: member role API + group role API + 替换 policy.go | | 现有测试通过 |
| 3 | APISIX forward-auth 增强 + Redis 缓存 | 网关注入 Header | curl -v 查看 |
| 4 | taskCloud + taskBill + taskProject + taskTask 迁移 | 全服务统一 | |
| 5 | taskEvents: CompanyCreated consumer (creator→role) + RoleChanged consumers (缓存失效) | | 事件追踪 |
| 5-6 | taskFE: usePermission() + UI 显隐 | | 手动测试 |
| 6-7 | E2E 测试: 第 8 节 10 个用例 | | 全部通过 |
| 7 | 文档 + Code Review | | |

**总工期: 7 天**。每个 day 可并行推进（day1-2 可合并为 1 天如果多人协作）。

---

## 11. 附录：与架构版本的关系

- 架构 v60 已正确表达本设计的组件拓扑（taskAuth PDP + shareLib 薄客户端 + 组角色）
- 本文是 v60 的实施深化文档，不改变组件拓扑
- 旧文档 v1/v2/v3 全部作废，本文为唯一权威来源
