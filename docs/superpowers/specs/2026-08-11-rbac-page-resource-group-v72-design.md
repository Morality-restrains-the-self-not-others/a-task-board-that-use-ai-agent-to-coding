# RBAC v72 — 逻辑资源组（页面组 ⊃ UI 组件区域 ⊃ UI/API）前后端统一授权

- **Status:** accepted（2026-08-11；Q1=A1 默认绑 ui_region / 整页展开；Q2=B2 仅 `region:*`/`page:*`，无粗码展开）
- **Date:** 2026-08-11
- **Iteration:** rbac-logical-resource-group-v72
- **Based on:** v63 RBAC；取代 `2026-08-11-tenant-menu-access-management-design.md` 仅前端菜单方案
- **Architecture impact:** **是** — 已写入 ArchiMate v72 四件套 + ADR-0003
- **Supersedes draft mistake:** 不得将「资源组」等同于「页面组」
- **ADR:** `docs/adr/0003-logical-resource-group-page-region-rbac.md`
- **python_api_approval:** not_applicable（全 Go）

---

## 0. 架构现状理解（设计前提）

- Enterprise Landscape **v13 current**；Application Integration 积压至 **v71 target**；RBAC 基线 **v63 ✅**
- 链路：APISIX forward-auth → taskAuth PDP → `X-Tenant-Perms` → `shareLib/authz.RequirePerm`
- 既存 `tenant_resource_group_assignment` = **业务实体 → 人员小组**（数据范围），与本设计「逻辑能力资源组」**正交**，禁止混表/混名

> 本次在 Identity & Access 上引入**分层逻辑资源组**，前后端同源 Enforce。

---

## 1. 核心模型更正（相对上一稿）

### ❌ 错误（已废弃表述）

> 「页面 = 资源组；组件/API 直接挂在页面下」

### ✅ 正确层级（本次权威）

```
逻辑资源组（Resource Group）—— 抽象能力边界，可被角色授予
│
├── 页面组（Page Group）··········· 逻辑组 / 载体
│     └── 绑定多个「UI 组件区域」
│
└── UI 组件区域（UI Region）······· 逻辑组 / **实际授权粒度（资源组落地）**
      ├── UI 组件（可显示/可交互）
      └── API 端点（可调用）
```

| 概念 | 性质 | 职责 |
|------|------|------|
| **资源组 Resource Group** | 抽象 | 可被角色授予的能力单元；本设计中**具体落地为 UI 组件区域**（页面组不直接挂叶子成员） |
| **页面组 Page Group** | 逻辑组 / **载体** | 对应控制台一个页面（路由/侧栏项）；**聚合**其下多个 UI 组件区域；便于「整页授权」快捷展开 |
| **UI 组件区域 UI Region** | 逻辑组 / **资源组** | 页面内一块功能区；**绑定**该区的 UI 组件 + 所需 API |
| **叶子成员** | 非组 | `ui` 组件 key、`api` method+path |

**原则：**

1. **UI 与 API 的可访问性绑定到 UI 组件区域**，不直接绑到页面组。
2. **UI 组件区域绑定到页面组**（父子逻辑归属）。
3. 角色授予的对象是**资源组**（UI Region）；授予页面组 = 展开为其下全部 Region（便利语法，非模型等同）。
4. 前后端均按 Region（及展开出的 api/ui 成员）Enforce，防止仅前端隐藏被绕过。

---

## 2. 问题陈述

| 层 | 现状 | 风险 |
|----|------|------|
| FE | 菜单↔粗码隐藏 | 可绕过 |
| BE | handler 手写粗码，与「勾选页面」无强制绑定 | 权限漂移 |
| 上一稿 | 页面=资源组 | 无法表达页内分区（只读列表 vs 危险按钮） |

**产品要求：**

1. 页面组只是资源组的**载体**，内含多个 UI 组件区域。
2. 每个 UI 组件区域绑定 UI 组件 + API 端点。
3. UI 组件区域（作为资源组）赋予角色；人/组可多角色。
4. tenant_admin 拥有租户内全部资源组（全部 Region；或等价全部 Page→展开）。
5. FE+BE 统一管控。

---

## 3. 领域概念（供 /6-ddd）

| 概念 | 说明 |
|------|------|
| **ResourceGroup** | 可授予单元；`kind=ui_region` 为默认授权对象；`kind=page` 为载体/聚合 |
| **PageGroup** | `kind=page` 的逻辑组；`page_key`、路由、侧栏 |
| **UiRegion** | `kind=ui_region` 的逻辑组；`region_key`；`parent_page_id` |
| **ResourceMember** | Region 下叶子：`ui` / `api` |
| **Role ↔ ResourceGroup** | 多对多；绑 page 时 PDP **展开**为子 region 并集 |
| **Subject ↔ Roles** | 人/组多角色（既有表支持） |
| **Business Resource Assignment** | 既存 v63 业务实体→人员组；**不改** |

### 命名消歧

| 用语 | 内部名 | 勿混用 |
|------|--------|--------|
| 逻辑资源组 | `auth_resource_group` | ≠ `tenant_resource_group_assignment` |
| 页面组 | `kind='page'` | 不是「资源组」的同义词 |
| UI 组件区域 | `kind='ui_region'` | 默认授权粒度 |
| 叶子 | `auth_resource_member` | ≠ 公司成员 |

---

## 4. 方案对比

| 方案 | 说明 | 决策 |
|------|------|------|
| A. 页面=资源组，叶子直接挂 page | 无法分区 | ❌ 拒（上一稿） |
| B. 仅粗码 Bundle | 无 Region 语义 | ❌ 拒 |
| **C. 页面组 ⊃ UI Region（资源组）⊃ ui/api + 角色绑 Region（page 可展开）** | 对齐产品 | ✅ **采用** |
| D. 平行 ACL 绕过 PDP | | ❌ 拒 |

---

## 5. Decision

### 5.1 数据模型（taskAuth）

```
auth_resource_group
  id
  group_key          UNIQUE   -- page: "people.access"；region: "people.access.role_matrix"
  kind               ENUM('page','ui_region')
  parent_id          NULL     -- page: NULL；ui_region: → page 的 id
  display_name
  route_prefix       NULL     -- 仅 page
  sort_order
  is_system
  company_id         NULL     -- 系统目录

auth_resource_member          -- 仅允许挂在 kind=ui_region
  id
  resource_group_id           -- FK → ui_region
  member_kind        ENUM('ui','api')
  member_key                  -- ui: "PeopleAccess.SaveButton"；api: "PUT /api/auth/roles/..."
  UNIQUE(resource_group_id, member_kind, member_key)

auth_role_resource_group      -- 角色 ↔ 资源组（page 或 ui_region）
  id, role_id, resource_group_id
  UNIQUE(role_id, resource_group_id)
```

约束（DB 或应用层）：

- `ui_region.parent_id` 必须指向 `kind=page`
- `auth_resource_member` 禁止挂到 `kind=page`
- 删除 page 时级联子 region（系统种子禁止删）

**种子示例（访问管理页）：**

| group_key | kind | parent |
|-----------|------|--------|
| `people.access` | page | — |
| `people.access.subject_list` | ui_region | people.access |
| `people.access.region_matrix` | ui_region | people.access |
| `people.access.save_actions` | ui_region | people.access |

`subject_list` 成员：列表 UI + `GET .../members/`、`GET .../groups/` …  
`save_actions` 成员：保存按钮 UI + `PUT` 角色/绑定 API …

### 5.2 PDP 展开

```
角色绑定的 resource_group 集合
  → 若含 page：展开为其全部子 ui_region
  → 并集所有 ui_region
  → 注入:
       region:<region_key>
       page:<page_key>          -- 若该 page 下至少一 region 命中（侧栏用）
       （可选）api/ui 成员声明供反查；**不**展开旧粗码（决策 B2）
tenant_admin → 全部系统 resource_group（page+region）
```

### 5.3 Enforce（前后端）

| 层 | API |
|----|-----|
| BE | `RequireRegion("people.access.save_actions")`；或 `RequireAPI("PUT", path)` 经 registry 反查所属 region |
| FE | `hasRegion(key)` 控制区块/按钮；`hasPage(key)` 控制侧栏（page = 任一子 region） |
| 安全边界 | **无对应 region 则 API 403**；仅隐藏 UI 不算完成 |

### 5.4 访问管理 UI

1. 默认勾选粒度：**UI 组件区域**（可按页面折叠树展示）。
2. 「整页勾选」= 勾选该 page 下全部 region（写 `auth_role_resource_group` 可存 page 一行，由 PDP 展开）。
3. 展开 region 只读查看其 ui/api 成员（审计）。
4. 人/组：**多角色**分配。

### 5.5 分期

| 阶段 | 内容 |
|------|------|
| P1 | 表 + 种子 + PDP（仅 region/page）+ `RequireRegion` + 访问管理 Region 树 + 关键 API 挂 region |
| P2 | CI：新路由必须登记为某 region 的 api 成员 |
| P3 | 细到每个按钮 region；存量 `RequirePerm(粗码)` 逐步迁到 RequireRegion |

### 5.6 业务意图 → 事件

| 业务意图 | 事件 | 发布点 | 消费者 | 例外 |
|----------|------|--------|--------|------|
| 角色↔资源组变更 | `RoleResourceGroupsChanged`（或扩展 `RoleChanged`） | taskAuth | PDP 缓存失效 | — |
| 成员/组角色变更 | `TenantRoleChanged` | taskTenant | membership_rev | — |
| 查询目录 | — | — | — | 只读 |

---

## 6. 🕸️ Code Review Graph

- `CRG unavailable: empty graph (0 nodes)`
- Fallback：`rbac_pdp.go`、`shareLib/authz`、`rbac_tenant_roles.go`、既有 `PeopleAccess`/`tenantConsoleNav`
- 爆炸半径：taskAuth schema/PDP、shareLib、taskTenant 多角色、taskFE 树形访问管理、业务 handler Enforce

---

## 7. Value Stream 影响

- 新流：`tenant-logical-resource-access`（组织与成员）
- 步骤：配角色×Region（及 page 展开）→ 主体获角色 → PDP → FE 区块 / BE 403
- 字段例：`task-auth.auth_resource_group.group_key`、`task-auth.auth_resource_member.member_key`、`task-auth.auth_role_resource_group.role_id`

---

## 8. 🏛️ 架构变更影响

- **迭代版本**: v72 🎯 target
- **迭代名称**: rbac-logical-resource-group-v72
- **作者**: claude
- **设计日期**: 2026-08-11 11:36
- **用户决策**: Q1=A1，Q2=B2，Approve
- **新增文件**（application-integration，四类齐全）:
  - 🆕 `docs/architecture/v72-application-integration-20260811-1136-claude.puml`
  - 🆕 `docs/architecture/v72-application-integration-20260811-1136-claude.diff.archimate`（增量：v71→v72 + Plateau/Gap/WP）
  - 🆕 `docs/architecture/v72-application-integration-20260811-1136-claude.full.archimate`（全量拓扑 + 变迁视图）
  - 🆕 `docs/architecture/v72-application-integration-20260811-1136-claude.mermaid.md`
- **ADR**: `docs/adr/0003-logical-resource-group-page-region-rbac.md`
- **变更明细**:
  - 🟢 auth_resource_group / auth_resource_member / auth_role_resource_group
  - 🟡 taskAuth PDP、shareLib/authz、taskFE、业务 Enforce、taskTenant 多角色语义
- **已有文件（未修改）**: v71 及更早 target/current

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v71→Gap→WP→v72；Target 数据流 FE→GW→Auth/Biz + 三表 Access |
| **`.full.archimate`** | 同上拓扑 + auth DB Access；含变迁视图 |

### 🐍 Python 新增接口

- **not_applicable**（全 Go）

---

## 9. 与错误草稿 / 已写前端的关系

| 项 | 处置 |
|----|------|
| 「页面=资源组」表述 | **作废** |
| `PeopleAccess` 入口 | 保留，改为 **页面 → Region 树** 勾选 |
| 菜单→粗码单角色 | 非默认路径 |
| 仅 FE 隐藏 | 非完成标准 |

---

## 10. 成功标准

1. 模型与文档明确：**页面组是载体；UI 组件区域是资源组；ui/api 挂在区域上**。
2. 无某 region：对应 UI 不渲染 **且** 其登记 API 返回 403。
3. 无某 page 下任一 region：侧栏隐藏该页。
4. tenant_admin 全资源组；CI 新 API 必须归属某 region。
5. 不与 `tenant_resource_group_assignment` 混用。
6. **后续租户改动**遵守项目元规则：`.ai/01_project_constraints/45_tenant_logical_rbac_resource_groups.md`（约束索引第 40 条）。

---

## 11. 已确认决策

| 项 | 选择 |
|----|------|
| Q1 角色默认粒度 | **A1** — 默认 ui_region；整页勾选写 page，PDP 展开 |
| Q2 旧粗码 | **B2** — 本授权链路仅 `region:*` / `page:*` |
| Q3 | **Approve** — 2026-08-11 |

---

## 附录

- 无 traceId → 跳过 Loki。
