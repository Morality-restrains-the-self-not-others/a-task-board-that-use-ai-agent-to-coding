# RBAC 权限体系 — v63 综合设计（自定义角色 + 小组资源分配）

- **状态**: 🎯 target（合并终稿，待审批）
- **迭代**: rbac-merged-v63
- **作者**: claude
- **设计日期**: 2026-08-05 02:38
- **架构版本**: v63（合并 v61/v62 并行设计线）
- **合并来源**:
  - 并行线 v61 (`2026-08-04-rbac-final-design.md`) + v62 (`2026-08-04-rbac-with-groups-design.md`) — 4 固定角色 + group_admin + 资源→组分配
  - 本会话线 (`2026-08-05-rbac-custom-role-design.md`) — 租户自定义角色 + RequirePerm 权限码判定 + 硬切换 + 组→角色继承 + 前端 37 处
- **用户决策**: 合并成 v63 综合设计（AskUserQuestion 2026-08-05）

---

## 1. 合并结果总览

| 维度 | 采用来源 | 决策 |
|------|---------|------|
| 判定模型 | 本会话线 | **RequirePerm 权限码集合判定**（自定义角色无优先级可言） |
| 角色模型 | 本会话线 + v62 | **租户自定义角色**（勾选权限码）+ 内置 5 角色（含 group_admin） |
| 组的作用 | 双线合并 | 组既是**角色继承单元**（组→角色）又是**资源分配单元**（资源→组） |
| is_admin/is_superuser | 本会话线 | **硬切换**（回填后代码删除） |
| 前端 | 本会话线 | taskFE 37 处改造（usePermissions） |
| 平台侧 | 并行线 v61 | 内置 super_admin / employee 不变（Phase 1 不做平台自定义角色） |

**最终角色层级**：

```
Platform (平台层)                         Tenant (租户层)
super_admin (内置, 锁定)                   tenant_admin (内置, 锁定)
  │ 继承                                    │ 创建+管理
employee (内置, 锁定)                        ├── 自定义角色 (租户创建, 勾选权限码)
                                            ├── 小组 (group)
                                            │     ├── group_admin (内置, 组范围)
                                            │     └── 组→角色继承 (tenant_group_role)
                                            └── 资源→组分配 (resource_group_assignment)
                                                  ├── project → group (view/manage)
                                                  ├── cloud → group
                                                  └── task → group
```

**核心判定链**：

```
用户请求 → APISIX forward-auth → taskAuth PDP 计算权限码集合
  → 注入 X-User-Roles + X-Tenant-Perms (cid→码集合, 含组角色继承+组资源继承)
  → authz.RequirePerm(码) — O(1) Context 读取
  → 资源级: authz.RequireGroupResourceAccess(资源) — 组资源分配命中判定
```

---

## 2. 角色模型

### 2.1 内置系统角色（is_system=1，锁定）

| 角色 | level | scope | 权限码集合 |
|------|-------|-------|-----------|
| `super_admin` | platform | 全局 | 全部 platform 码 (6) |
| `employee` | platform | 全局 | tenant:audit, user:impersonate, billing:audit |
| `tenant_admin` | tenant | 公司 | 全部 tenant 码 (17：14 资源码 + 3 组码) |
| `group_admin` 🆕 | tenant | 指定组 | group-members:manage, group-resources:view（组范围判定） |
| `member` | tenant | 公司 | 全部 view 码 (7) + task:manage |

### 2.2 租户自定义角色（is_system=0，Phase 1）

- tenant_admin 创建：display_name + 勾选 tenant 级权限码（17 个中任意组合，**不含组码**——组码仅内置 group_admin 使用）
- 归属：`auth_role.company_id`（新列），同租户内 display_name 唯一
- 生命周期：创建/更新权限码/删除（有成员或组引用时拒绝）

### 2.3 角色分配与继承

```
auth_user_role (taskAuth)              — 平台角色分配 (company_id=NULL)
tenant_member_role (taskTenant)        — 租户成员直接角色（member_id ↔ 角色名）
tenant_group_role (taskTenant)         — 组→角色继承（组内成员自动获得组角色）
tenant_group_admin (taskTenant)        — group_admin 指派（谁管理哪个组，v62 表）
tenant_resource_group_assignment       — 资源→组分配（项目/云/任务 → 组，view/manage）
```

用户在某租户的有效权限 = **直接角色 ∪ 组角色继承** 的权限码并集 + **组资源继承**（所在组被分配的资源的访问权）。

---

## 3. 权限码（23 码 = v60 20 码 + v62 3 组码）

| level | 权限码 | 数量 |
|-------|--------|------|
| platform | platform:manage, tenant:audit, user:impersonate, system:config, billing:audit, employee:manage | 6 |
| tenant 资源 | company/manage+view, member/manage+view, group:manage, project/manage+view, task/manage+view, cloud/manage+view, billing/manage+view, workspace:manage | 14 |
| tenant 组 🆕 | group-members:manage（组内成员管理）, group-resources:view（查看组资源）, group-resources:manage（组资源分配管理） | 3 |

> 组码仅内置角色使用：group_admin={group-members:manage, group-resources:view}；tenant_admin 全 17 码（含组码，穿透）。

---

## 4. shareLib/authz — 判定 API

```go
// 租户级权限码判定（主路径，O(1) Context 读集合）
func RequirePerm(w, r, perm PermCode, companyID string) bool
func RequirePlatformPerm(w, r, perm PermCode) bool

// 组范围判定（v62 增强）
func RequireGroupAdmin(w, r, companyID, groupID string) bool      // tenant_admin 穿透
func RequireGroupPerm(w, r, companyID, groupID, perm PermCode) bool
func HasGroupResourceAccess(w, r, companyID, resourceType, resourceID string) bool
//    — 判定: 成员所在组被分配该资源（permission view/manage → 资源 view/manage 码）
//    — 与 RequirePerm 组合使用: RequirePerm(码) 通过 或 HasGroupResourceAccess 通过

// 布尔版（前端/逻辑复用）
func HasPerm / HasPlatformPerm
```

资源访问判定模式（handler 内）：

```go
if !authz.RequirePerm(w, r, authz.PermProjectView, cid) &&
   !authz.HasGroupResourceAccess(w, r, cid, "project", projectID) {
    write403(w); return
}
```

---

## 5. 表结构（合并）

### 5.1 taskAuth (027_rbac_roles.sql)

```sql
-- v60/v61 4 表 + 新列
ALTER TABLE auth_role ADD COLUMN company_id VARCHAR(64) DEFAULT NULL;
ALTER TABLE auth_role DROP INDEX name, ADD UNIQUE KEY uk_name_company (name, COALESCE(company_id,''));
-- 种子: 5 内置角色 (含 group_admin) + 23 权限码 + 角色→权限关联
-- 种子权限: tenant_admin → 17 tenant 码; group_admin → 2 组码; member → view 码 + task:manage
```

### 5.2 taskAuth (028_rbac_backfill.sql — 硬切换)

```sql
-- is_superuser=1 → super_admin; is_staff=1 且非 super → employee (v60 沿用)
-- 回填后 is_admin/is_superuser 列停用，代码删除
```

### 5.3 taskTenantService (017_member_role + 018_group_role + 019_group_admin + 020_resource_assignment)

```sql
tenant_member_role            -- v60 沿用
tenant_group_role             -- v60: 组→角色
tenant_group_admin            -- v62: group_id ↔ user_id 指派
tenant_resource_group_assignment  -- v62: (resource_type, resource_id, group_id, permission view/manage)
-- 硬切换回填: tenant_company_member.is_admin=1 → tenant_member_role(tenant_admin)
```

---

## 6. API 清单（合并后 24 个 Go API，零 Python）

### taskAuth（10 个）

| # | 方法 | 路径 | 所需权限 | 来源 |
|---|------|------|---------|------|
| 1 | POST | `/api/auth/roles/` | tenant_admin | 自定义角色创建（本会话线） |
| 2 | PUT | `/api/auth/roles/role_id/{rid}/` | tenant_admin | 更新权限码（内置 403） |
| 3 | DELETE | `/api/auth/roles/role_id/{rid}/` | tenant_admin | 删除（有引用 422） |
| 4 | GET | `/api/auth/roles/company_id/{cid}/` | 租户成员 | 角色列表（内置+自定义） |
| 5 | POST | `/api/auth/user-roles/user_id/{uid}/` | super_admin | 分配平台角色（v61 沿用） |
| 6 | DELETE | `/api/auth/user-roles/user_id/{uid}/role_name/{name}/` | super_admin | 撤销平台角色 |
| 7 | GET | `/api/auth/role-users/role_name/{name}/` | super_admin | 角色成员列表 |
| 8 | GET | `/api/auth/user-roles/` | 当前用户 | 我的角色 |
| 9 | GET | `/api/auth/user-permissions/` | 当前用户 | 我的权限码（前端判定） |
| 10 | POST | `/api/internal/authz/check` | internal | PDP 判定（HTTP fallback） |

### taskTenantService（14 个：本会话线 4 + v62 新增 7 + v62 改造 3 计入）

| # | 方法 | 路径 | 所需权限 | 来源 |
|---|------|------|---------|------|
| 11 | PUT | `/api/tenant/member-role/company_id/{cid}/member_id/{mid}/` | tenant_admin | 成员角色（本会话线） |
| 12 | PUT | `/api/tenant/group-role/company_id/{cid}/group_id/{gid}/` | tenant_admin | 组角色（组继承） |
| 13 | GET | `/api/tenant/member-role/company_id/{cid}/` | member:view | 成员角色列表 |
| 14 | GET | `/api/tenant/group-role/company_id/{cid}/` | member:view | 组角色列表 |
| 15 | PUT | `/api/tenant/group-admin/company_id/{cid}/group_id/{gid}/user_id/{uid}/` | tenant_admin | 指派 group_admin（v62） |
| 16 | DELETE | `/api/tenant/group-admin/company_id/{cid}/group_id/{gid}/user_id/{uid}/` | tenant_admin | 移除 group_admin |
| 17 | GET | `/api/tenant/group-admin/company_id/{cid}/group_id/{gid}/` | tenant_admin/group_admin/member | 查看组管理员 |
| 18 | POST | `/api/tenant/group-members/company_id/{cid}/group_id/{gid}/user_id/{uid}/` | tenant_admin/group_admin(本组) | 添加成员（v62 改造） |
| 19 | DELETE | `/api/tenant/group-members/company_id/{cid}/group_id/{gid}/user_id/{uid}/` | tenant_admin/group_admin(本组) | 移除成员（v62 改造） |
| 20 | GET | `/api/tenant/group-members/company_id/{cid}/group_id/{gid}/` | tenant_admin/group_admin/member | 成员列表（v62 改造） |
| 21 | POST | `/api/tenant/resource-group/company_id/{cid}/` | tenant_admin | 资源→组分配（v62） |
| 22 | DELETE | `/api/tenant/resource-group/company_id/{cid}/assignment_id/{aid}/` | tenant_admin | 撤销分配 |
| 23 | GET | `/api/tenant/resource-group/company_id/{cid}/group_id/{gid}/` | tenant_admin/group_admin/member | 小组关联资源 |
| 24 | GET | `/api/tenant/resource-group/company_id/{cid}/resource_type/{type}/resource_id/{rid}/` | tenant_admin/group_admin | 资源分配到的组 |

### 存量端点改造（40+ 端点 → RequirePerm，沿用 v61 映射表 + v62 §7 组改造）

- `group_handlers.go`：handleGroupAddMember/RemoveMember → `RequireGroupAdmin`（v62 §7.1）
- 资源服务（taskProject/taskCloud/taskTask）：`checkResourceAccess` 增加组继承分支（v62 §7.2）
- `/api/tenant/{cid}/todos/*` DELETE 等 → `RequirePerm(task:manage)`

---

## 7. 事件驱动（合并 9 事件）

| 事件 | 发布点 | 作用 | 来源 |
|------|--------|------|------|
| `CompanyCreated` (已有) | taskTenant | creator → tenant_admin | 两线共用 |
| `CompanyMemberAdded` (已有) | taskTenant | 新成员 → member | 本会话线 |
| `TenantMemberRoleChanged` | taskTenant | 成员角色变更 → 缓存失效 | 两线共用 |
| `TenantGroupRoleChanged` | taskTenant | 组角色变更 → 组内成员失效 | 两线共用 |
| `RoleChanged` | taskAuth | 自定义角色增改删 → PDP 缓存失效 | 本会话线 |
| `GroupAdminAssigned` / `GroupAdminRevoked` | taskTenant | 组管理员指派/移除 → 缓存失效 | v62 |
| `ResourceAssignedToGroup` / `ResourceRevokedFromGroup` | taskTenant | 资源↔组 → 组内成员资源权限刷新 | v62 |
| `MemberAddedToGroup` / `MemberRemovedFromGroup` | taskTenant | 组成员变动 → 继承权限刷新 | v62 |

缓存模型：`authz:perms:{uid}:{cid}`（含 rev）；任一角色/组/资源事件 → rev++ → 下次 forward-auth 重算。

---

## 8. 硬切换迁移（沿用本会话线）

- 服务端：policy.go 删除（RequirePerm + 事件分配）；taskAuth is_superuser 查表 → RequirePlatformPerm；taskProject/taskTask/taskCloud/taskBill is_admin → RequirePerm
- 前端：taskFE 37 处（13 文件）→ usePermissions() composable（HasPerm/HasPlatformPerm）
- 完成判据：全仓 grep is_admin/is_superuser（代码）返回 0 处

---

## 9. 实施周期（10 天 = v62 的 8 天 + 组/资源 2 天）

| 天 | 内容 |
|----|------|
| D1 | shareLib/authz（RequirePerm/RequireGroupAdmin/ResourceAccess）+ taskAuth 表/种子（5 角色 23 码）/迁移 SQL |
| D2 | taskAuth PDP（权限码集合 + 组继承 + 组资源继承计算 + X-Tenant-Perms）+ 自定义角色 CRUD |
| D3 | taskTenant member-role/group-role/group-admin API + CompanyCreated/MemberAdded consumer |
| D4 | taskTenant resource-group API（4 个）+ 6 个组/资源事件 |
| D5 | 逐服务判定替换（taskProject/taskTask/taskCloud/taskBill）+ group_handlers 改造 + policy.go 删除 |
| D6 | APISIX forward-auth 增强 + taskFE usePermissions + 路由守卫 |
| D7 | taskFE 剩余按钮显隐（37 处收尾） |
| D8 | 回填校验 + 硬切换收尾 + 单元测试（12） |
| D9 | E2E 用例矩阵（16：含自定义角色 + 组资源端到端） |
| D10 | 回归 + 文档收尾 |

**测试矩阵**：12 单元 + 16 E2E（角色 CRUD 权限、自定义角色授予后端点可达、组继承级联、组资源访问/撤销、事件缓存失效、硬切换 0 引用、前端守卫/按钮、交叉组隔离）。

---

## 10. 🏛️ 架构变更影响

- **迭代版本**: v63 🎯 target（基于 v62 并行线；合并本会话自定义角色线）
- **设计日期**: 2026-08-05 02:38
- **变更明细 (vs v62)**:
  - 🟢 [NEW] 租户自定义角色 — auth_role.company_id + 4 CRUD API（本会话线）
  - 🟢 [NEW] RequirePerm 权限码判定（替代 RequireRole 优先级主路径）+ X-Tenant-Perms 注入
  - 🟢 [NEW] taskFE 前端改造（usePermissions，37 处）
  - 🟢 [NEW] RoleChanged 事件（自定义角色缓存失效）
  - 🟡 [MODIFIED] v62 既有: group_admin/tenant_group_admin/resource_group_assignment/6 事件（保留）
  - 🟡 [MODIFIED] shareLib/authz — RequirePerm + RequireGroupAdmin + HasGroupResourceAccess
  - 🟡 [MODIFIED] APISIX — forward-auth 注入 X-User-Roles + X-Tenant-Perms
  - 🟡 [MODIFIED] taskProject/taskTask/taskCloud/taskBill — RequirePerm + 组资源继承检查
  - 🔴 [DEPRECATED] is_admin/is_superuser 直接鉴权（硬切换）、policy.go creator 硬编码、RequireRole 优先级判定
- **新增接口**: 24 个 Go API（taskAuth 10 + taskTenantService 14），**零 Python 接口**（🐍 门禁不触发）
- **架构文件**: `v63-application-integration-20260805-0238-claude.{puml,archimate,mermaid.md}`

---

## 11. 📈 价值流影响

| 域 | 流 | 影响 |
|----|----|------|
| 系统管理与策略 | — | 🆕 角色/权限管理流（自定义角色、成员角色、组管理员） |
| 用户与认证 | 认证流 | 🟡 PDP 权限码计算 + X-Tenant-Perms |
| 组织与成员 | 成员/分组 | 🟡 member_role/group_role/group_admin + 资源→组分配 |
| 云平台/项目/任务 | 现有流 | 🟡 鉴权码化 + 组资源继承分支 |

---

## 12. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 |
|---------|--------|--------|--------------|
| 公司创建 | CompanyCreated | taskTenant | creator → tenant_admin |
| 成员加入公司 | CompanyMemberAdded | taskTenant | 默认 member 角色 |
| 设置成员角色 | TenantMemberRoleChanged | taskTenant | 缓存失效 |
| 设置组角色 | TenantGroupRoleChanged | taskTenant | 组内成员级联失效 |
| 创建/改/删自定义角色 | RoleChanged | taskAuth | PDP 缓存失效 |
| 指派/移除组管理员 | GroupAdminAssigned / GroupAdminRevoked | taskTenant | 缓存失效 |
| 资源↔组分配 | ResourceAssignedToGroup / ResourceRevokedFromGroup | taskTenant | 组内成员资源权限刷新 |
| 组成员变动 | MemberAddedToGroup / MemberRemovedFromGroup | taskTenant | 继承权限刷新 |

---

## 13. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 硬切换无回退 | 回填 SQL 可逆（归档表备份旧值）；E2E 先行 |
| 自定义角色误授 manage 码 | 默认空权限集合；UI 提示；审计日志 |
| 组资源继承判定复杂度（每资源请求 2 次查询） | PDP 一次性计算组资源集合注入；缓存 rev 失效 |
| X-Tenant-Perms 体积（多租户+多组） | 上限保护 + 截断 + PDP fallback；实测估算 <1.5KB |
| 双线合并遗漏（v62 表/事件 vs 本会话线表/事件） | API/表/事件清单逐项对照（§5/§6/§7 三表合并） |
| group_admin 与自定义角色边界 | 组码仅内置角色；自定义角色 UI 不展示组码 |
