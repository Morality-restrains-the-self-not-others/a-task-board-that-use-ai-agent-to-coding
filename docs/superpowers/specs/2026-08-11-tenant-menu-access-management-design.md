# 租户控制台「访问管理」— 菜单级 RBAC 落地设计

- **Status:** superseded
- **Superseded by:** `docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md`（ADR-0003；页面组≠资源组；A1/B2）
- **Date:** 2026-08-11
- **Iteration:** tenant-menu-access-management
- **Based on:** v63 RBAC（`2026-08-05-rbac-merged-v63-design.md`）
- **Architecture impact:** 无新服务/新表；复用 taskAuth 自定义角色 + taskTenant member/group-role + 前端侧栏/路由门禁
- **Note:** 本文件仅为早期「菜单↔粗码」桥接方案；实现须按 v72 逻辑资源组重做，入口「访问管理」可保留。

---

## 1. Goal / Success Criteria

1. 人员管理子菜单新增 **「访问管理」**，路由 `/tenant/:tenant/people/access/`。
2. 管理员可为 **成员** 或 **小组** 勾选可访问的左侧菜单（及对应页面资源权限码），保存后生效。
3. 无对应权限码者：**侧栏隐藏**该菜单；直链进入展示「无权限」空态；API 仍由现有 `RequirePerm` 拒绝。
4. 实现完全建立在现有 v63 RBAC（权限码 + PDP + 角色分配），不引入平行 ACL 表。
5. 单测覆盖：菜单↔权限映射、侧栏可见性 helper、保存编排、`usePermissions` 失效重载。

---

## 2. Alternatives Considered

| 方案 | 说明 | 决策 |
|------|------|------|
| A. 仅前端按已有权限码隐藏菜单 | 无「配置」UI | 不足，拒 |
| B. 新建 menu ACL 表 | 与 v63 双轨 | 拒 |
| C. 复用工作空间 AccessManagementModal | 语义是 workspace ACL | 拒 |
| **D. 菜单映射 + 自定义角色 + member/group-role** | 产品「勾选菜单」映射到权限码，走现有 API | **采用** |

---

## 3. Decision

### 3.1 菜单 ↔ 权限码映射（SSOT：前端 `tenantConsoleNav.js`）

| 菜单 key | 可见文本 | 所需权限（任一即可见） | 路由片段 |
|----------|----------|------------------------|----------|
| `nav.projects` | 项目列表 | `project:view` | `/projects` |
| `nav.work_panel` | 工作面板 | `task:view` | `/work-panel` |
| `nav.image_market` | 镜像市场 | `cloud:view` | `/image-market` |
| `people.invite` | 邀请人 | `member:manage` | `/people/invite/` |
| `people.manage` | 管理人员 | `member:manage` | `/people/manage/` |
| `people.groups` | 管理分组 | `group:manage` | `/people/groups/` |
| `people.access` | 访问管理 | `member:manage` | `/people/access/` |
| `settings.company` | 公司设置 | `company:view` | `/settings/company/` |
| `settings.cloud` | 云平台绑定 | `cloud:manage` | `/settings/cloud-platform/` |
| `settings.gitlab` | GitLab | `company:manage` | `/settings/gitlab-connection/` |
| `settings.task_panel` | 工作空间管理 | `workspace:manage` | `/settings/task-panel/` |
| `settings.feature_params` | 智能体资源配置 | `cloud:manage` | `/settings/feature-params/` |
| `settings.deliverable` | 交付物体系 | `company:manage` | `/deliverable-systems/` |
| `settings.status` | 进度体系 | `company:manage` | `/settings/status/` |
| `billing.*` | 资源与订单（概览/订单/流水/用量） | `billing:view` | `/billing/...` |

勾选菜单保存时：将勾选集合 **展开为权限码去重并集**（含 view/manage 语义：勾选 manage 类菜单时写入对应 manage 码；view 类写入 view 码）。

### 3.2 访问管理页 UX

1. 需 `member:manage`；否则「无权限」空态。
2. 左侧 Tab：**成员** / **小组**；列表来自既有 members/groups API。
3. 选中主体后：展示菜单树 checkbox（按上表），预填当前角色 permissions。
4. **内置角色**（`tenant_admin` / `member` / `group_admin`）只读提示：「系统角色不可直接改菜单；保存将创建自定义访问角色并替换分配」。
5. **自定义角色**：直接 `PUT /api/auth/roles/role_id/{rid}/` 更新 permissions。
6. 保存编排（成员）：
   - 若当前为系统角色或无角色 → `POST /api/auth/roles/`（display_name=`访问·{成员名}`）→ `PUT .../member-role/...` 绑定 `name`
   - 若当前为「访问·*」或其它自定义 → `PUT` 更新 permissions；必要时再 PUT 绑定
7. 小组同理，绑定走 `group-role`（需调用方具备 `group:manage`；本页入口已要求 `member:manage`，保存小组时额外校验/依赖管理员通常兼有）。
8. 保存成功后 `usePermissions.invalidate()`（若改的是自己）并 toast。

### 3.3 侧栏与路由

- `Sidebar.vue`：`usePermissions().load`；每项 `canSeeMenu(key)`；人员管理整块在任一 people.* 可见时显示；访问管理链到 `/people/access/`。
- 路由 meta `requiredPerms` + 轻量 guard：无权限则仍进入页面，由页内空态处理（与 PeopleInvite 一致），避免硬跳转丢失上下文。
- 安全真相仍在 API `RequirePerm`。

### 3.4 顺带修复

- `PeopleGroups.vue` 权限码 `group:members:manage` → `group-members:manage`（与 `shareLib/authz` 一致）。

### 3.5 不在范围

- 不改 PDP / 权限码种子（不新增 `menu:*` 码）。
- 不改工作空间 AccessManagementModal。
- 不新增 Kafka 业务意图事件以外的表（角色变更已有 `RoleChanged` / `TenantRoleChanged`）。

---

## 4. Consequences

**正向：** 产品可配置菜单访问；与 v63 一致；零新表。  
**负向：** 每成员可能产生一个自定义角色（命名 `访问·*`）；需避免重复创建（按 display_name / 已绑角色复用）。  
**缓解：** 保存时优先更新已分配的自定义角色；系统角色保护提示。

---

## 5. Architecture note

无 Application_Component 增减 → **不更新** `docs/architecture/` ArchiMate 版本；记入本设计即可。
