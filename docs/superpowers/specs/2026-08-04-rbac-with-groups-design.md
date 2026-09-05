# RBAC 身份角色权限体系 — 含小组与资源分配

- **状态**: 🎯 target (v62)
- **迭代**: rbac-identity-role-permission-system-with-groups
- **作者**: claude
- **设计日期**: 2026-08-04
- **基于**: v61 RBAC 基础设计 + 现有 `tenant_company_group` 表

---

## 1. 需求与概念模型

### 1.1 完整角色层级

```
Platform (平台层)                   Tenant (租户层)
super_admin (p=100)                 tenant_admin (p=100)
  │ 继承                                │ 创建+管理
employee (p=50)                         ├── 小组 (group)
                                        │     ├── group_admin (p=75) ← 管理组内成员
                                        │     └── member (p=50) ← 继承组资源访问权
                                        │
                                        └── 资源分配
                                              ├── project → group (可访问/管理)
                                              ├── cloud → group
                                              └── task → group
```

### 1.2 核心流程

```
tenant_admin 创建小组 → 指派 group_admin → 分配资源给小组
                                              ↓
                              group_admin 管理成员（添加/移除）
                                              ↓
                              小组成员自动获得组关联资源的访问权限
```

### 1.3 角色定义

| 角色 | 层级 | 优先级 | 范围 | 能力 |
|------|------|--------|------|------|
| super_admin | platform | 100 | 全局 | 全部 |
| employee | platform | 50 | 全局 | 运维+审计 |
| tenant_admin | tenant | 100 | 公司 | 创建小组、分配资源、管理全部成员 |
| **group_admin** 🆕 | tenant | **75** | **指定小组** | 管理组内成员、查看组资源 |
| member | tenant | 50 | 公司 | 基本使用 + 继承组权限 |

---

## 2. 数据模型

### 2.1 新增表

```sql
-- ═══ group_admin: 谁管理哪个小组 ═══
CREATE TABLE tenant_group_admin (
    id VARCHAR(64) PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL REFERENCES tenant_company_group(id),
    user_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, user_id),
    INDEX idx_company (company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ═══ 资源→小组分配 ═══
-- 租户把项目/云资源/任务分配给特定小组
CREATE TABLE tenant_resource_group_assignment (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    resource_type ENUM('project','cloud','task','workspace') NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    group_id VARCHAR(64) NOT NULL REFERENCES tenant_company_group(id),
    permission VARCHAR(32) NOT NULL DEFAULT 'view',  -- view | edit | manage
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(resource_type, resource_id, group_id),
    INDEX idx_group (group_id),
    INDEX idx_resource (resource_type, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 2.2 更新现有表

```sql
-- tenant_company_group: 增加管理员关联
-- (可选，group_admin 已通过 tenant_group_admin 表表达)
-- 无需修改现有表结构，通过 JOIN 查询即可

-- tenant_company_group_member: 无需修改
-- 现有表已足够表达成员关系
```

### 2.3 与 v61 角色表的融合

`tenant_group_role` (v61 已有) 用于组→角色关联：
- `group_id → "group_admin"` → 该组有一个管理员角色
- `group_id → "member"` → 组内成员继承 member 角色（Phase 2）

`tenant_group_admin` 是 `tenant_group_role` 的**具体实例**：记录哪个 user 是哪个 group 的 admin。

---

## 3. 权限判定模型

### 3.1 用户有效角色计算

```go
func getEffectiveRoles(userID, companyID string) []ResolvedRole {
    var roles []ResolvedRole

    // 1. 平台角色（全局）
    platformRoles := getUserPlatformRoles(userID)
    for _, r := range platformRoles {
        roles = append(roles, ResolvedRole{Role: r, Scope: "platform", ScopeID: ""})
    }

    // 2. 租户角色（公司级）
    memberRole := getTenantMemberRole(userID, companyID)
    if memberRole != "" {
        roles = append(roles, ResolvedRole{Role: memberRole, Scope: "tenant", ScopeID: companyID})
    }

    // 3. 🆕 组管理员角色（组级）
    groupAdminRoles := getGroupAdminRoles(userID, companyID)
    for _, g := range groupAdminRoles {
        roles = append(roles, ResolvedRole{Role: "group_admin", Scope: "group", ScopeID: g.GroupID})
    }

    // 4. 🆕 组继承资源权限（通过组成员身份）
    groupIDs := getGroupIDsForMember(userID, companyID)
    for _, gid := range groupIDs {
        resourcePerms := getGroupResourcePermissions(gid)
        for _, rp := range resourcePerms {
            roles = append(roles, ResolvedRole{Role: "member", Scope: rp.ResourceType, ScopeID: rp.ResourceID, Perm: rp.Permission})
        }
    }

    return roles
}
```

### 3.2 鉴权中间件增强

```go
// RequireGroupAdmin — 要求用户是指定小组的管理员
func RequireGroupAdmin(w http.ResponseWriter, r *http.Request, companyID, groupID string) error

// RequireGroupResourceAccess — 要求用户对某资源有组级访问权
func RequireGroupResourceAccess(w http.ResponseWriter, r *http.Request, companyID, resourceType, resourceID string) error
```

### 3.3 判定流程

```
请求: PUT /api/tenant/groups/{gid}/members/add  (添加成员到小组)

1. authz.RequireTenantRole(w, r, cid, "tenant_admin", "group_admin")
2. → FromContext(r) → AuthContext
3. → 用户有 group_admin 角色? 检查 scopeID == gid?
4. → 是 → 通过
5. → 否 → 检查是否有 tenant_admin (p=100 >= 75)? → 是 → 通过
6. → 都不是 → 403
```

---

## 4. API 设计

### 4.1 taskTenantService 新增/修改 API

**小组管理 API** (serviceName=`tenant`):

| 方法 | 路径 | 用途 | 鉴权 |
|------|------|------|------|
| GET | `/api/tenant/groups/company_id/{cid}/` | 小组列表 | tenant_admin / group_admin (只看自己的组) |
| POST | `/api/tenant/groups/company_id/{cid}/` | 创建小组 | tenant_admin |
| DELETE | `/api/tenant/groups/company_id/{cid}/group_id/{gid}/` | 删除小组 | tenant_admin |
| PUT | `/api/tenant/groups/company_id/{cid}/group_id/{gid}/` | 编辑小组信息 | tenant_admin |

**小组管理员 API**:

| 方法 | 路径 | 用途 | 鉴权 |
|------|------|------|------|
| PUT | `/api/tenant/group-admin/company_id/{cid}/group_id/{gid}/user_id/{uid}/` | 设置/更换小组管理员 | tenant_admin |
| DELETE | `/api/tenant/group-admin/company_id/{cid}/group_id/{gid}/user_id/{uid}/` | 移除小组管理员 | tenant_admin |
| GET | `/api/tenant/group-admin/company_id/{cid}/group_id/{gid}/` | 查看小组管理员 | tenant_admin / group_admin / member |

**小组内成员管理 API** (鉴权变更):

| 方法 | 路径 | 用途 | 鉴权 |
|------|------|------|------|
| POST | `/api/tenant/group-members/company_id/{cid}/group_id/{gid}/user_id/{uid}/` | 添加成员 | tenant_admin / group_admin(本组) |
| DELETE | `/api/tenant/group-members/company_id/{cid}/group_id/{gid}/user_id/{uid}/` | 移除成员 | tenant_admin / group_admin(本组) |
| GET | `/api/tenant/group-members/company_id/{cid}/group_id/{gid}/` | 查看成员列表 | tenant_admin / group_admin / member |

**资源→小组分配 API**:

| 方法 | 路径 | 用途 | 鉴权 |
|------|------|------|------|
| POST | `/api/tenant/resource-group/company_id/{cid}/` | 分配资源给小组 | tenant_admin |
| DELETE | `/api/tenant/resource-group/company_id/{cid}/assignment_id/{aid}/` | 撤销分配 | tenant_admin |
| GET | `/api/tenant/resource-group/company_id/{cid}/group_id/{gid}/` | 查看小组关联资源 | tenant_admin / group_admin / member |
| GET | `/api/tenant/resource-group/company_id/{cid}/resource_type/{type}/resource_id/{rid}/` | 查看资源分配给了哪些组 | tenant_admin / group_admin |

### 4.2 API 合规

所有新增 API 均遵循 `/api/${serviceName}/${funcName}/key/value/...` 规范。

---

## 5. 系统角色种子数据（更新）

| name | level | priority | 初始权限 |
|------|-------|----------|---------|
| super_admin | platform | 100 | platform:manage, tenant:audit, user:impersonate, system:config, billing:audit, employee:manage |
| employee | platform | 50 | tenant:audit, user:impersonate, billing:audit |
| tenant_admin | tenant | 100 | company:manage/view, member:manage/view, **group:manage**, project:manage/view, task:manage/view, cloud:manage/view, billing:manage/view, workspace:manage |
| **group_admin** 🆕 | **tenant** | **75** | **group-members:manage, group-resources:view** |
| member | tenant | 50 | company:view, member:view, project:view, task:manage/view, cloud:view, billing:view |

### 新增权限码

```go
// Group management
PermGroupMembersManage PermCode = "group-members:manage"  // 管理组内成员
PermGroupResourcesView PermCode = "group-resources:view"  // 查看组关联资源
PermGroupResourcesMng  PermCode = "group-resources:manage" // 管理组关联资源 (tenant_admin)
```

---

## 6. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 |
|---------|----------------|--------|--------------|
| 设置小组管理员 | GroupAdminAssigned | taskTenant UpdateGroupAdmin | 审计日志 + 缓存失效 |
| 移除小组管理员 | GroupAdminRevoked | taskTenant DeleteGroupAdmin | 审计日志 |
| 资源分配给小组 | ResourceAssignedToGroup | taskTenant AssignResource | 组内成员权限刷新 |
| 撤销资源分配 | ResourceRevokedFromGroup | taskTenant RevokeResource | 组内成员权限刷新 |
| 成员加入小组 | MemberAddedToGroup | taskTenant AddGroupMember | 权限刷新 (继承组资源) |
| 成员移出小组 | MemberRemovedFromGroup | taskTenant RemoveGroupMember | 权限刷新 (失去组资源) |

---

## 7. 现有 handler 鉴权变更

### 7.1 group_handlers.go 改造

| 函数 | 当前鉴权 | 改为 |
|------|---------|------|
| `handleGroupsListOrCreate` | `requireCompanyMember()` | LIST: member+ / CREATE: `RequireTenantRole(tenant_admin)` |
| `handleGroupDelete` | `requireCompanyMember()` | `RequireTenantRole(tenant_admin)` |
| `handleGroupMembers` | `requireCompanyMember()` | 不变 |
| `handleGroupAddMember` | `requireCompanyMember()` | `RequireGroupAdmin(tenant_admin, group_admin)` |
| `handleGroupRemoveMember` | `requireCompanyMember()` | `RequireGroupAdmin(tenant_admin, group_admin)` |

### 7.2 资源 handler 增强

资源服务 (taskProjectService, taskCloudService, taskTaskService) 在处理资源访问时需要额外检查：
- 用户是否有直接的资源访问权限？(现有逻辑)
- 用户是否通过小组继承获得了资源访问权限？(新增检查)

```go
// 资源访问判定增强
func checkResourceAccess(userID, companyID, resourceType, resourceID string) bool {
    // 1. 直接权限（现有逻辑）
    if hasDirectAccess(userID, resourceType, resourceID) { return true }
    // 2. 🆕 组继承权限
    if hasGroupResourceAccess(userID, companyID, resourceType, resourceID) { return true }
    return false
}
```

---

## 8. 架构变更影响

- **迭代版本**: v62 🎯 target
- **基于**: v61 (RBAC 基础)
- **变更明细**:
  - 🟢 [NEW] group_admin 角色 — priority=75, 组内成员管理权限
  - 🟢 [NEW] tenant_group_admin 表 — 小组管理员指派
  - 🟢 [NEW] tenant_resource_group_assignment 表 — 资源→组分配
  - 🟢 [NEW] 6 个 Domain Events (GroupAdmin*, ResourceAssigned*, MemberAddedToGroup*)
  - 🟡 [MODIFIED] group_handlers.go — 鉴权从 requireCompanyMember → RequireGroupAdmin
  - 🟡 [MODIFIED] 资源服务 — 增加组继承权限检查
  - 🟡 [MODIFIED] shareLib/authz — 新增 RequireGroupAdmin, RequireGroupResourceAccess
  - 🆕 [NEW] 10 个 API (tenant groups + group-admin + group-members + resource-group)
- **新增 API**: 10 个，全部 Go，全部合规路径
- **实施周期**: 在 v61 基础上 +3 天

---

## 9. Domain Concept Inventory

| 概念 | 类型 | BC | 说明 |
|------|------|-----|------|
| Group | Entity | Tenant | 公司内小组 |
| GroupAdmin | Role | Tenant | 组管理员角色，scope=group |
| GroupMember | Entity | Tenant | 组成员关系 |
| ResourceGroupAssignment | Aggregate | Tenant | 资源→组分配关系 |
| GroupAdminAssigned | DomainEvent | Tenant | → 缓存失效 + 通知 |
| ResourceAssignedToGroup | DomainEvent | Tenant | → 组内成员权限刷新 |
| MemberAddedToGroup | DomainEvent | Tenant | → 继承组资源权限 |

---

## 10. 安全约束

| 约束 | 实现 |
|------|------|
| group_admin 仅管理自己组的成员 | ScopeCheck: `group_admin.group_id == targetGroupID` |
| 组资源继承 | 成员加入组时自动获得组关联资源的 view 权限 |
| 组资源撤销 | 成员移出组时自动失去组关联资源权限 |
| tenant_admin 穿透 | tenant_admin 自动拥有所有组的 group_admin 权限 |
| 交叉组隔离 | group_admin_A 不能管理 group_admin_B 的组 |
