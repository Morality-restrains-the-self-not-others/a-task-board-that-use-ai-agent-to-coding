# RBAC 自定义角色权限体系 — 完整设计 (v5)

- **状态**: 🎯 target（重新头脑风暴终稿，待审批）
- **迭代**: rbac-custom-role-system
- **作者**: claude
- **设计日期**: 2026-08-05
- **架构版本**: v61（替代 v60 target；v60 仍是已设计基线）
- **替代**: `2026-08-04-rbac-implementation-blueprint.md`（v4）— 角色模型部分被本文重设计；PDP/网关/事件机制沿用
- **触发背景**: 用户重新头脑风暴 — 4 固定角色不足，需要租户级自定义角色；is_admin/is_superuser 硬切换（不做双轨）；组→角色继承 Phase 1 纳入

---

## 1. 设计总览

```
用户请求
  → APISIX (forward-auth → taskAuth PDP)
  → PDP 计算权限码集合 → 注入 X-User-Roles + X-Tenant-Perms
  → Go Service Handler
  → authz.RequirePerm(w, r, authz.PermProjectManage, cid)  // 读 Context O(1)，权限码判定
  → 业务逻辑
```

**核心变化 (vs v60)**：

| 维度 | v60 (4 固定角色) | v5 (自定义角色) |
|------|-----------------|-----------------|
| 判定依据 | 角色名 + 优先级 (`RequireRole`) | **权限码集合** (`RequirePerm`) — 自定义角色无优先级可言 |
| 网关注入 | X-User-Roles + X-Tenant-Memberships(角色) | X-User-Roles + **X-Tenant-Perms**(cid→权限码集合) |
| 角色来源 | 4 内置角色种子 | 4 内置角色（锁定）+ **租户自定义角色**（勾选权限码） |
| is_admin/is_superuser | 双轨兼容保留 | **硬切换** — 迁移后废弃，代码删除 |
| 组→角色继承 | Phase 1 纳入 | 保留（同 v60） |
| 前端 | 未覆盖 | **taskFE 37 处改造**（路由守卫/按钮显隐） |

**判定路径（性能核心）**：权限码集合随 forward-auth 注入，服务端 `RequirePerm` 纯内存集合判断 O(1)，零 HTTP fallback（PDP fallback 仅作为 header 缺失兜底）。自定义角色天然支持 — PDP 计算时把自定义角色映射为权限码集合。

---

## 2. 角色模型

### 2.1 内置系统角色（is_system=1，锁定：不可改权限、不可删）

| 角色 | level | 权限码集合 | 说明 |
|------|-------|-----------|------|
| `super_admin` | platform | 全部 platform 码 (6) | 平台全权 |
| `employee` | platform | tenant:audit, user:impersonate, billing:audit | 平台运维（不含 platform:manage/system:config/employee:manage） |
| `tenant_admin` | tenant | 全部 tenant 码 (14) | 租户全权 |
| `member` | tenant | 全部 view 码 (7) + task:manage | 租户默认角色 |

### 2.2 租户自定义角色（is_system=0，Phase 1 新增）

- 由 **tenant_admin** 在租户内创建：`display_name` + 勾选权限码（仅 tenant level 码，14 个中任意组合）
- 归属：`auth_role.company_id`（新列）— 角色定义集中存 taskAuth（PDP 单一真源）
- 命名：同一租户内 display_name 唯一（`UNIQUE(company_id, display_name)`）；租户间可重名
- 生命周期：创建 / 更新权限码 / 删除（有成员或组引用时拒绝，须先解绑）
- 判定：成员被分配自定义角色后，其权限码集合 = 直接角色 ∪ 组继承角色 的所有权限码**并集**

### 2.3 角色分配模型

```
auth_user_role (taskAuth)          —— 平台角色分配 (company_id=NULL)
tenant_member_role (taskTenant)    —— 租户成员直接角色 (member_id ↔ role)
tenant_group_role (taskTenant)     —— 租户组角色 (group_id ↔ role) → 组成员继承
```

- 成员加入公司 → 默认分配 `member`（事件驱动，见 §6）
- 公司创建者 → `tenant_admin`（CompanyCreated consumer，见 §6）
- 用户在某租户的权限码集合 = 直接角色 ∪ 组继承角色 的权限码并集（PDP 计算）

---

## 3. 权限码（20 码 manage/view 二分，沿用 v60 种子）

| level | 权限码 | 数量 |
|-------|--------|------|
| platform | platform:manage, tenant:audit, user:impersonate, system:config, billing:audit, employee:manage | 6 |
| tenant | company/manage+view, member/manage+view, group:manage, project/manage+view, task/manage+view, cloud/manage+view, billing/manage+view, workspace:manage | 14 |

> 权限码静态注册（`shareLib/authz/permissions.go` const 枚举 + DB 种子），自定义角色从 14 个 tenant 码中勾选。用户自定义角色无权勾选 platform 码。

---

## 4. shareLib/authz — 判定 API 变更

### 4.1 权限码驱动判定（新增，替代 RequireRole 为主路径）

```go
// RequirePerm 要求用户在指定租户拥有该权限码（Context 读集合，O(1)）
func RequirePerm(w http.ResponseWriter, r *http.Request, perm PermCode, companyID string) bool

// RequirePlatformPerm 要求用户拥有平台权限码
func RequirePlatformPerm(w http.ResponseWriter, r *http.Request, perm PermCode) bool

// RequireTenantMember 要求用户是租户有效成员（原 requireCompanyMember 语义）
func RequireTenantMember(w http.ResponseWriter, r *http.Request, companyID string) bool

// HasPerm / HasPlatformPerm — 布尔版（前端/非 handler 场景）
```

### 4.2 AuthContext 解析（ParseFromHeaders 扩展）

```go
type AuthContext struct {
    UserID         string
    PlatformRoles  []string             // ["super_admin"]
    TenantPerms    map[string]stringSet // companyID → 权限码集合（含组继承，PDP 已算好）
}
```

- `X-User-Roles`: 逗号分隔平台角色
- `X-Tenant-Perms`: `cid1:company:view,task:view;cid2:task:view`（分号=租户，冒号=分隔，逗号=权限码）
- 体积估算：tenant_admin 全量 ≈ 224 字符/租户，1-3 租户 ≈ 700 字符，8KB header 限制内

### 4.3 向后兼容

- 保留 `RequireRole`/`RequireTenantRole` 薄封装（内部转为权限码判定，避免 v60 蓝图已有代码草案失效）— 判定语义统一走权限码

---

## 5. 表结构变更

### 5.1 taskAuth (027_rbac_roles.sql，基于 v60 蓝图 + 新列)

```sql
-- v60 既有 4 表（auth_role / auth_permission / auth_role_permission / auth_user_role）+ 新列
ALTER TABLE auth_role ADD COLUMN company_id VARCHAR(64) DEFAULT NULL;  -- NULL=平台角色/全局内置；非 NULL=租户自定义角色
ALTER TABLE auth_role DROP INDEX name, ADD UNIQUE KEY uk_name_company (name, COALESCE(company_id,''));
-- 种子: 4 内置角色 + 20 权限码 + 角色→权限关联（沿用 v60 5.1 SQL）
```

### 5.2 taskAuth (028_rbac_backfill.sql — 硬切换回填)

```sql
-- is_superuser=1 → super_admin（platform, company_id=NULL）
-- is_staff=1 且非 superuser → employee
-- 沿用 v60 5.2 回填 SQL；额外: 回填后校验无遗漏（计数对比）
```

### 5.3 taskTenantService (017_member_role.sql + 018_group_role.sql)

```sql
CREATE TABLE IF NOT EXISTS tenant_member_role (   -- v60 5.3 沿用
    id VARCHAR(64) PRIMARY KEY, member_id VARCHAR(64) NOT NULL,
    role_name VARCHAR(64) NOT NULL, company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL, assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_member_role (member_id, role_name),
    CONSTRAINT fk_tmr_member FOREIGN KEY (member_id) REFERENCES tenant_company_member(id)
);

CREATE TABLE IF NOT EXISTS tenant_group_role (    -- Phase 1 组继承
    id VARCHAR(64) PRIMARY KEY, group_id VARCHAR(64) NOT NULL,
    role_name VARCHAR(64) NOT NULL, company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL, assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_group_role (group_id, role_name)
);
-- 硬切换回填: tenant_company_member.is_admin=1 → tenant_member_role(tenant_admin)
-- 回填完成后 is_admin 列停用（列保留待后续 DROP，代码先行删除判断）
```

> `role_name` 存角色名（内置 + 自定义角色名）。自定义角色变更权限码时关联表不受影响（按名引用）；删除自定义角色时校验无成员/组引用。

---

## 6. 事件驱动

| 事件 | 发布点 | 消费者/作用 |
|------|--------|------------|
| `CompanyCreated` (已有) | taskTenant | 🟡 增强: 创建者分配 tenant_admin（Kafka consumer） |
| `TenantMemberRoleChanged` (v60 已设计) | taskTenant | 成员角色变更 → PDP 缓存失效 (Redis rev++) |
| `TenantGroupRoleChanged` (v60 已设计) | taskTenant | 组角色变更 → 组内成员权限级联失效 |
| `CompanyMemberAdded` (已有) | taskTenant | 🟡 增强: 新成员默认分配 member 角色 |
| `RoleChanged` (v60 已设计) | taskAuth | 自定义角色创建/更新/删除 → 受影响租户缓存失效 |

缓存模型：`authz:perms:{uid}:{cid}`（Redis）含 rev 校验；角色/成员变更 → 事件 → `membership_rev`/`role_rev` +1 → 下次 forward-auth 重新计算。

---

## 7. API 清单（全 Go，零 Python）

### 7.1 taskAuth 新增（14 个，含 v60 已设计）

| # | 方法 | 路径 | 所需权限 | 说明 |
|---|------|------|---------|------|
| 1 | POST | `/api/auth/roles/` | tenant_admin(member:manage) | 创建租户自定义角色（body: company_id, display_name, permissions[]） |
| 2 | PUT | `/api/auth/roles/role_id/{rid}/` | tenant_admin(member:manage) | 更新权限码（内置角色 403） |
| 3 | DELETE | `/api/auth/roles/role_id/{rid}/` | tenant_admin(member:manage) | 删除（有引用 422） |
| 4 | GET | `/api/auth/roles/company_id/{cid}/` | 租户成员 | 租户角色列表（内置+自定义+权限码） |
| 5 | POST | `/api/auth/user-roles/user_id/{uid}/` | super_admin(employee:manage) | 分配平台角色 |
| 6 | DELETE | `/api/auth/user-roles/user_id/{uid}/role_name/{name}/` | super_admin(employee:manage) | 撤销平台角色 |
| 7 | GET | `/api/auth/role-users/role_name/{name}/` | super_admin(employee:manage) | 角色成员列表 |
| 8 | GET | `/api/auth/user-roles/` | 当前用户 | 我的角色 |
| 9 | GET | `/api/auth/user-permissions/` | 当前用户 | 我的权限码（前端判定用） |
| 10 | POST | `/api/internal/authz/check` | internal | PDP 判定（HTTP fallback 兜底） |

### 7.2 taskTenantService 新增（4 个）

| # | 方法 | 路径 | 所需权限 | 说明 |
|---|------|------|---------|------|
| 11 | PUT | `/api/tenant/member-role/company_id/{cid}/member_id/{mid}/` | tenant_admin(member:manage) | 设置成员角色（body: role_name） |
| 12 | PUT | `/api/tenant/group-role/company_id/{cid}/group_id/{gid}/` | tenant_admin(group:manage) | 设置组角色（组内成员继承） |
| 13 | GET | `/api/tenant/member-role/company_id/{cid}/` | 租户成员(member:view) | 成员角色列表（前端角色配置页） |
| 14 | GET | `/api/tenant/group-role/company_id/{cid}/` | 租户成员(member:view) | 组角色列表 |

### 7.3 端点→权限码映射（存量改造，40+ 端点）

沿用 v60 蓝图 §4 映射表，判定调用改为 `authz.RequirePerm`：

| 存量端点（节选） | v60 所需角色 | v5 判定 |
|-----------------|-------------|---------|
| `/api/system-admin/users` GET | super_admin, employee | RequirePlatformPerm(tenant:audit) |
| `/api/system-admin/resource-pricing/*` | super_admin, employee | RequirePlatformPerm(billing:audit) |
| `/api/tenant/{cid}/members` GET | member+ | RequireTenantMember + HasPerm(member:view) |
| `/api/tenant/{cid}/members/{mid}/toggle-status` | tenant_admin | RequirePerm(member:manage) |
| `/api/tenant/{cid}/groups` POST | tenant_admin | RequirePerm(group:manage) |
| `/api/tenant/{cid}/todos/*` DELETE | tenant_admin | RequirePerm(task:manage) |
| `/api/tenant/{cid}/cloud/*` POST | tenant_admin | RequirePerm(cloud:manage) |
| `/api/tenant/{cid}/billing/*` POST | tenant_admin | RequirePerm(billing:manage) |
| `/api/tenant/{cid}/workspace/*` PUT | tenant_admin | RequirePerm(workspace:manage) |

> 语义变化提示：v60 中「tenant_admin 专属」的操作在 v5 中变为「拥有对应 manage 码的角色即可」— 自定义角色授予 manage 码后获得同等能力。安全边界由权限码勾选保证（默认自定义角色为空集合，须显式授权）。

---

## 8. 硬切换迁移计划

### 8.1 服务端（删除 is_admin/is_superuser 判断）

| 位置 | 现状 | 切换为 |
|------|------|--------|
| taskTenantService/src/policy.go | requireCompanyAdmin/checkCompanyAdmin/isCreator/is_admin 判断 | **删除** — RequirePerm(member:manage) + 事件分配（creator 已是 tenant_admin） |
| taskAuth handlers_system_admin.go | 查表 is_superuser | RequirePlatformPerm |
| taskProjectService/taskTaskService | is_admin via taskTenant lookup | RequirePerm(对应 manage/view) |
| taskCloudService | is_admin + isCreator | RequirePerm(cloud:manage/view) |
| taskBill | is_admin | RequirePerm(billing:manage/view) |
| taskAuth DB | is_superuser/is_staff 列 | 回填后停用（保留列，代码不再读写） |

### 8.2 前端 taskFE（37 处，13 文件）

| 文件（节选） | 改造 |
|-------------|------|
| system_admin_route_guard_service.js | 平台守卫 → HasPlatformPerm(tenant:audit)（用户权限接口） |
| Sidebar.vue / Navbar.logic.vue | 菜单显隐 → 权限码 |
| PeopleManage.vue / PeopleGroups.vue / TenantCompanySettings.vue | 按钮显隐 → HasPerm(company:manage 等) |
| SystemAdminUsers.vue / SystemAdminOrderRecords.vue / UserListRow.vue / PendingInvitations.vue | is_admin 分支 → 租户权限码 |

新增 composable：`usePermissions()`（拉取 `/api/auth/user-permissions/` + `/api/auth/tenant-memberships/`，本地角色/权限码缓存，返回 HasPerm/HasPlatformPerm）。

### 8.3 切换顺序（硬切换点集中）

1. shareLib/authz（判定 API）→ 2. taskAuth（表+种子+PDP+角色 API）→ 3. taskTenantService（member/group role API + 回填）→ 4. 逐服务替换判定（taskProject/taskTask/taskCloud/taskBill）→ 5. APISIX forward-auth 注入 X-Tenant-Perms → 6. taskFE 37 处 → 7. 回填 SQL 校验 + 删除旧判断代码（硬切换完成）→ 8. E2E

**硬切换完成判据**：全仓 `grep is_admin/is_superuser`（代码，排除 DB 列定义）返回 0 处（保留 test 中历史 fixture 除外）。

---

## 9. 实施周期（8 天，vs v60 的 7 天 + 前端）

| 天 | 内容 |
|----|------|
| D1 | shareLib/authz + taskAuth 表/种子/迁移 SQL + 回填 SQL |
| D2 | taskAuth PDP（权限码集合计算 + Redis 缓存 + X-Tenant-Perms）+ 角色 CRUD API |
| D3 | taskTenantService member/group role API + CompanyCreated/MemberAdded consumer 增强 |
| D4 | 逐服务判定替换（taskProject/taskTask/taskCloud/taskBill）+ policy.go 删除 |
| D5 | APISIX forward-auth 增强 + taskFE usePermissions + 路由守卫 |
| D6 | taskFE 剩余按钮显隐（37 处收尾）|
| D7 | 回填校验 + 硬切换收尾 + 单元/集成测试 |
| D8 | E2E 用例矩阵 + 回归（含自定义角色端到端场景）|

**测试矩阵**（10 单元 + 12 E2E）：角色 CRUD 权限校验、自定义角色授予后端点可达、组继承级联、事件缓存失效、硬切换后旧字段无引用、前端守卫/按钮显隐。

---

## 10. 🏛️ 架构变更影响

- **迭代版本**: v61 🎯 target（基于 v57 current；v60 为被替代 target）
- **设计日期**: 2026-08-05
- **变更明细**:
  - 🟢 [NEW] shareLib/authz — RequirePerm 权限码判定（薄客户端 + PermCode 枚举）
  - 🟢 [NEW] taskAuth — PDP 计算权限码集合 + 注入 X-Tenant-Perms + 租户自定义角色 CRUD（4 API）+ auth_role.company_id
  - 🟢 [NEW] taskTenantService — tenant_group_role 表 + group-role API（Phase 1 组继承）
  - 🟡 [MODIFIED] taskAuth — +4 RBAC 表 + 角色管理 API（沿用 v60）+ 硬切换
  - 🟡 [MODIFIED] taskTenantService — +2 表 + 成员/组角色 API + policy.go 删除
  - 🟡 [MODIFIED] taskProject/taskTask/taskCloud/taskBill — hand-rolled → RequirePerm
  - 🟡 [MODIFIED] APISIX — forward-auth 注入 X-User-Roles + X-Tenant-Perms
  - 🟡 [MODIFIED] taskFE — 37 处 is_admin/is_superuser → usePermissions（前端改造首次纳入架构）
  - 🔴 [DEPRECATED] is_admin/is_superuser 直接鉴权（硬切换，代码删除）
  - 🔴 [DEPRECATED] policy.go creator 硬编码（CompanyCreated 事件替代）
- **新增接口**: 14 个 Go API，**零 Python 接口**（🐍 门禁不触发）
- **架构文件**: `v61-application-integration-<ts>-claude.{puml,archimate,mermaid.md}`（含架构变迁 v60→v61 视图）

---

## 11. 📈 价值流影响

| 域 | 流 | 影响 |
|----|----|------|
| 系统管理与策略 | — | 🆕 新增：角色/权限管理流（自定义角色 CRUD、成员角色分配） |
| 用户与认证 | 认证流 | 🟡 forward-auth 注入 X-Tenant-Perms、PDP 权限码计算 |
| 组织与成员 | 成员管理 | 🟡 member_role/group_role、creator→tenant_admin、新成员默认 member |
| 云平台/项目/任务 | 现有流 | 🟡 鉴权实现替换（行为不变，判定码化） |

---

## 12. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 |
|---------|--------|--------|--------------|
| 公司创建 | CompanyCreated (已有) | taskTenant | 🟡 creator 自动分配 tenant_admin |
| 成员加入公司 | CompanyMemberAdded (已有) | taskTenant | 🟡 默认分配 member 角色 |
| 设置成员角色 | TenantMemberRoleChanged | taskTenant | 缓存失效 (rev++) |
| 设置组角色 | TenantGroupRoleChanged | taskTenant | 组内成员权限级联失效 |
| 创建/更新/删除自定义角色 | RoleChanged | taskAuth | 受影响租户 PDP 缓存失效 |
| 分配/撤销平台角色 | PlatformRoleAssigned / PlatformRoleRevoked | taskAuth | 缓存失效 |

---

## 13. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 硬切换无回退路径 | 回填 SQL 可逆（备份原 is_admin/is_superuser 值到归档表）；切换点集中，E2E 先行 |
| 自定义角色误授 manage 码（越权） | 默认角色为空权限集合；UI 明确提示 manage 码影响面；审计日志记录角色变更 |
| X-Tenant-Perms header 体积（多租户+大集合） | 上限保护：超阈值截断 + PDP fallback；实测 1-3 租户 ≈ 700 字符 |
| 组继承级联失效延迟 | 事件即时失效（rev++），forward-auth 每次请求实时重算 |
| 前端 37 处遗漏 | 切换后 grep 校验 + 路由守卫/按钮 E2E 覆盖 |
