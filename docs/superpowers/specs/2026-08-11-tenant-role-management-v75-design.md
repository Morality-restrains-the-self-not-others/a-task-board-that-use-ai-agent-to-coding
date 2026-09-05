# 租户角色管理（可复用角色 + 多角色并集）— v75 设计

- **Status:** accepted（2026-08-11 总体审批通过；选项 1）
- **Date:** 2026-08-11
- **Iteration:** tenant-role-management-v75
- **Author:** cursor
- **Based on:** ADR-0003 / ADR-0004；v72–v74 application-integration targets；用户锁定 Q1=C / Q2=A / Q3=B / Q4=A
- **Architecture impact:** **是** — 已写入 ArchiMate **v75** 四件套（`.puml` + `.diff.archimate` + `.full.archimate` + `.mermaid.md`）
- **python_api_approval:** not_applicable（全 Go；扩展 taskAuth / taskTenant / taskFE）
- **ADR:** No-ADR: covered by existing ADR-0003/0004（模型不变；本迭代补齐角色生命周期 UX + 多角色绑定 API）；若后续「移除租户粗码 RequirePerm」落地为架构约束，另开 ADR supersede 过渡说明
- **design_approval:** approved 2026-08-11

---

## 0. 架构现状理解（设计前提）

- Enterprise Landscape **v13 current**；Application Integration 积压 **v74 target**（邀请预授）
- Identity 链路：APISIX forward-auth → taskAuth PDP → `X-Tenant-Perms`（粗码 ∪ `region:*`/`page:*`）→ `shareLib/authz`
- 访问管理现状（`/tenant/:id/people/access/`）：**主体中心**勾选 page/region → 隐式创建 `访问·某人` 自定义角色并绑定；**无**可复用角色管理 UI
- 后端已具备：taskAuth 角色 CRUD + resource-groups；taskTenant member/group 角色 PUT；PDP **已按多角色名并集**计算权限
- 缺口：角色一等公民 UI、多角色增删 API、访问页改为「选角色」、淘汰一对一隐式角色

> 本次在既有 RBAC 模型上补齐 **角色定义页 + 主体多角色分配**，不改变 page⊃region 授权语义。

---

## 1. 决策锁定（用户确认）

| # | 议题 | 决策 |
|---|------|------|
| Q1 | 入口布局 | **C** — 侧栏独立页 `people.roles` 管角色定义；访问管理改为主体↔角色（见 §3） |
| Q2 | 与 `访问·…` 关系 | **A 角色优先** — 先建可复用角色并配 page/region；主体只选角色；淘汰一对一隐式角色 |
| Q3 | 一员角色数 | **B 多角色并集** — 直接角色 ∪ 组继承角色；PDP 已并集，补齐绑定 API/UI |
| Q4 | 角色可配内容 | **A** — 仅 page/region + view/operate（ADR-0003/0004）；**不**在角色 UI 暴露粗码勾选 |

附加评估：**旧租户粗码 `RequirePerm` 可否移除** → 见 **§8**（结论：**现阶段不可移除**；可分阶段退役）。

---

## 2. 问题陈述

| 现状 | 痛点 |
|------|------|
| 无角色管理页 | 无法创建「财务只读」等可复用角色并批量赋给多人 |
| 访问管理隐式建角 | `访问·Alice` 膨胀、难命名、难共享、孤儿角色需手工清理 |
| PUT member-role 仅「加/触碰一行」 | 无 DELETE；FE 当单角色展示；多角色心智与 UI 不一致 |
| 邀请预授 (v74) 按 grants 建角 | 与角色优先模型短期并存，需对齐路径 |

---

## 3. 目标产品模型

```
角色管理页 (people.roles)          访问管理页 (people.access)
─────────────────────────          ─────────────────────────
• 列表：系统角色（只读）             • Tab：成员 / 小组
  + 租户自定义角色                   • 选主体 → 多选可分配角色
• 创建/改名/删自定义角色               • 只读预览：并集后的 page/region
• 编辑角色的 page/region             • 禁止再「直接勾选 region 并隐式建角」
  view|operate 矩阵
```

**系统角色**（`tenant_admin` / `member` 等 `is_system=1`）：不可删、不可改 resource-groups；可被分配（`member`）；`tenant_admin` 仍特判全量 region operate。

**自定义角色**：`display_name` 租户内唯一；权限真源 = `auth_role_resource_group`（effect）；粗码双写仅作过渡桥接（见 §8），**不对管理员暴露**。

---

## 4. 领域概念（供 /6-ddd）

| 概念 | 说明 |
|------|------|
| **ReusableRole** | 租户自定义 `auth_role`（`company_id` 非空，`is_system=0`） |
| **SystemTenantRole** | 内置租户角色；锁定 |
| **RoleGrant** | 角色↔page/region + effect（view\|operate） |
| **SubjectRoleBinding** | 成员/组 ↔ 多个 `role_name`；权限 = 并集 |
| **EffectiveAccess** | PDP 展开后的 `page:*`/`region:*`（+ 过渡粗码） |
| **LegacyAccessRole** | 历史 `访问·…`；迁移期可清理/合并，新路径不再创建 |

### Bounded Contexts

- **taskAuth**：角色定义、resource-groups、PDP、RoleChanged
- **taskTenant**：成员/组多角色绑定、TenantRoleChanged
- **taskFE**：角色管理页 + 访问管理改版

---

## 5. 方案对比

| 方案 | 说明 | 决策 |
|------|------|------|
| 仅在 Access 加「角色」Tab | 角色定义与主体分配挤同一页 | ❌ 拒（用户选 C） |
| 继续主体勾选隐式建角 | 无法复用 | ❌ 拒（用户选 A） |
| **独立角色页 + Access 改分配多角色** | 对齐角色优先 + 多角色 | ✅ **采用** |
| 角色 UI 继续勾粗码 | 与 Q4/ADR-0003 冲突 | ❌ 拒 |

---

## 6. Decision（实现要点）

### 6.1 前端（taskFE）

1. **导航**：`TENANT_CONSOLE_NAV` 增 `people.roles`（label「角色管理」，`pathSuffix: /people/roles/`）；种子登记 `auth_resource_group` page + 至少一 ui_region（如 `people.roles.main` / `people.roles.save_actions`）；侧栏 `hasPage('people.roles')`；过渡仍可用 `member:manage` 回退。
2. **新页 `PeopleRoles.vue`**（行数超限即拆）：角色列表、创建/改名/删除、选中后 page/region 矩阵（复用 Access 目录组件）、保存 → PUT resource-groups（**不**要求管理员填粗码）。
3. **改 `PeopleAccess.vue`**：去掉「勾选 region → 隐式建角」保存路径；改为多选角色（checkbox 列表）+ 有效权限只读预览；保存 → 成员/组角色绑定 API（§6.2）。
4. **邀请页 (v74 对齐，可同迭代或紧随)**：优先「选择已有自定义角色」；`pending_grants` 保留为兼容，文档标记 deprecated-to-roles。
5. **清理**：保留「清理未绑定访问角色」；新增可选「将 `访问·…` 标记为遗留」提示。

### 6.2 后端 API（全 Go）

#### taskAuth（扩展既有，零 Python）

| 方法 | 路径 | 说明 |
|------|------|------|
| 既有 | `POST/PUT/DELETE/GET …/roles…` | 继续；创建时允许 `permissions: []` 或省略，**真源靠 resource-groups** |
| 既有 | `GET/PUT …/resource-groups/` | 角色矩阵保存 |
| 可选增强 | 列表响应带 `bound_subject_count` | 删除前提示 |

Enforce：优先 `RequireRegionOperate('people.roles.save_actions')`，过渡并存 `member:manage`。

#### taskTenant（多角色绑定 — 新增/调整）

| 方法 | 路径 | 说明 |
|------|------|------|
| **PUT** | `/api/tenant/member-role/company_id/{cid}/member_id/{mid}/` | **语义改为 replace-all**：body `{ "role_names": ["…"] }`（空数组 = 仅保留默认策略见下）；兼容旧 `{ "role_name": "…" }` → 单元素数组 |
| **DELETE** | `/api/tenant/member-role/company_id/{cid}/member_id/{mid}/role_name/{name}/` | 移除单个角色 |
| **PUT** | `/api/tenant/group-role/…` | 同上 replace-all `role_names` |
| **DELETE** | `/api/tenant/group-role/…/role_name/{name}/` | 移除单个 |
| GET | 既有 list | 响应按 subject 聚合为 `role_names[]`（或保留多行，由 FE 聚合） |

规则：

- 禁止移除最后一个「有效身份」导致无法登录租户？→ **允许**清空自定义角色，但若无任何角色则 PDP 仍回落 `member` 基线（既有逻辑）；UI 提示。
- 禁止给主体绑定他租户角色；`tenant_admin` 可分配。
- 变更发既有 `TenantRoleChanged` / rev++。

Swagger：同步更新 taskTenant OpenAPI。

### 6.3 PDP

- **无需改并集算法**（已 `DirectRoles ∪ GroupRoles` → 粗码 ∪ region/page）。
- 列表/预览：FE 调角色 resource-groups 并集，或后续加 internal `effective-grants`（非本迭代必须）。

### 6.4 数据 / 迁移

- **无新表**（多角色已由 `UNIQUE(member_id, role_name)` 支撑）。
- `dataMigrate/taskAuth/`：种子 page `people.roles` + regions + api/ui members。
- 可选数据迁移脚本（非阻断）：扫描 `display_name LIKE '访问·%'` 列入遗留报告；**不**自动删除。

### 6.5 废弃路径

| 废弃 | 替代 |
|------|------|
| Access 保存隐式 `访问·…` | 角色页创建 + Access 分配 |
| 单 `role_name` 作为唯一心智 | `role_names[]` |
| 角色 UI 勾粗码 | 仅 page/region |

---

## 7. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 创建/更新/删除自定义角色 | RoleChanged | taskAuth | PDP 缓存 rev / 失效 | — |
| 更新角色 resource-groups | RoleChanged | taskAuth | 同上 | — |
| 替换成员多角色 | TenantMemberRoleChanged（既有 TenantRoleChanged 族） | taskTenant | membership rev++ | — |
| 替换组多角色 | TenantGroupRoleChanged | taskTenant | 组内成员权限级联失效 | — |
| 查询角色列表 / 有效权限预览 | — | — | — | 纯查询，无对应事件 |

意图文档：`docs/intents/frontend/tenant_role_management.*`（本迭代同步）。

---

## 8. 评估：旧租户 `RequirePerm`（粗码）是否可移除？

### 8.1 结论（摘要）

| 问题 | 结论 |
|------|------|
| **现在能否删除 `RequirePerm` API / 停发粗码？** | **否** |
| **能否作为 v75 范围移除？** | **否**（本迭代 Q4=A 只停止在角色 UI 暴露粗码） |
| **能否中长期退役租户粗码门禁？** | **能**，需独立退役计划（建议 v76+ / 专 ADR） |
| **平台 `RequirePlatformPerm`？** | **永久保留**（与租户 page/region 正交） |

### 8.2 证据（为何现在不能删）

1. **存量 Enforce 仍吃粗码**（非测试）：`taskAuth` 角色 CRUD、`taskTenant` policy/member-role/group-role/group-admin/业务资源组、`taskBill` 退款/账单管理、`taskCloudService`/`taskTaskService` 多处 `HasPerm(…, PermCloudManage|PermTaskManage|…)`。
2. **系统角色种子**仍通过 `auth_role_permission` 展开粗码；`member` / `tenant_admin` 基线依赖此路径。
3. **FE 侧栏过渡**：`useTenantPageAccess` / `TENANT_CONSOLE_NAV.anyOfPerms` 在无 `page:*` 时回退 `hasPerm(粗码)`；人员页门禁大量 `member:manage`。
4. **双写桥接仍在写**：`saveSubjectResourceAccess` / `coarsePermsFromGrantKeys` 把 region 映射成粗码，注释写明「以便存量 RequirePerm」。
5. **`RequireRegion*` 实现上仍走同一 `HasPerm` 集合**（查的是 `region:…` 伪码，不是删掉 RequirePerm 函数本身）。真正要退役的是 **租户级 `company:manage` / `member:manage` / `project:view` 等词汇表 + `auth_role_permission` 租户用法**，不是删掉 O(1) 集合 API。

### 8.3 建议退役路线（非本迭代交付）

| 阶段 | 动作 | 完成判据 |
|------|------|----------|
| **P0（本 v75）** | 角色 UI 只配 region；新角色允许空粗码；Access 不再隐式建角 | 新路径不依赖管理员理解粗码 |
| **P1** | 每个粗码门禁映射到明确 `ui_region`（或 capability-region）；handler 改 `RequireRegion*` | `rg RequirePerm.*Perm(Member\|Project\|…)` 租户调用归零 |
| **P2** | FE 去掉 `anyOfPerms` 粗码回退；仅 `hasPage`/`hasRegion` | 侧栏/按钮 E2E 全绿 |
| **P3** | 停止双写 `auth_role_permission`；PDP 对自定义角色只发 region/page | 自定义角色粗码列为空 |
| **P4** | 内置 `member`/`tenant_admin` 改为「全量/基线 region 种子」或保留最小粗码集并文档化例外 | ADR 记录；可删租户粗码常量 |

**风险**：过早删除会导致「只授了 region、账单/云 API 仍 RequirePerm(billing/cloud)」的**权限空洞**（有 UI 无 API 或反之）。

### 8.4 与本设计的边界

- v75 **不**删除 `RequirePerm`、**不**删 `auth_permission` 种子。
- v75 **不**在角色管理 UI 提供粗码勾选（Q4=A）。
- 创建/更新角色时：resource-groups 为真源；粗码可继续由服务端按映射**静默双写**（兼容 P0→P1），或逐步改为空数组 + 依赖已迁 Region 的 API——实现阶段选「静默双写」更安全，并在 OPT 跟踪停写。

---

## 9. 🕸️ Code Review Graph 分析

```
CRG unavailable: graph.db 存在但未索引 taskFE PeopleAccess / taskAuth rbac_roles 主业务符号（communities 以 skills 样例为主）
```

静态对照（替代）：

- 爆炸半径：`taskFE` 导航 + PeopleAccess/PeopleInvite；`taskTenant` rbac_tenant_roles；`taskAuth` rbac_roles / resource_groups / PDP（只读并集）
- 测试触及：`PeoplePageAccess.test.js`、`saveSubjectResourceAccess*.test.js`、`tenantConsoleNav.test.js`、taskTenant people/rbac tests

---

## 10. Value Stream 影响

- 仓库根无 `value-stream.yaml`（`NO_VS`）；影响域归入 **组织与成员 / 访问管理**。
- 预期：新 stream 步骤「角色管理 CRUD」「主体多角色分配」；Access 步骤从「主体勾选 region」改为「主体选角色」。
- 字段：无新表列强制；绑定语义 `tenant_member_role.role_name` 多行；种子 `auth_resource_group.group_key=people.roles*`。
- 完整切片交 `/4-value-stream`。

---

## 11. 🏛️ 架构变更影响

- **迭代版本:** v75 🎯 target（基于 v74）
- **迭代名称:** tenant-role-management-v75
- **作者:** cursor
- **设计日期:** 2026-08-11 21:35
- **视图:** `application-integration`（enterprise-landscape 无新服务组件，未改）
- **新增文件**（四类伴生，缺一不可）:
  - 🆕 `docs/architecture/v75-application-integration-20260811-2135-cursor.puml`
  - 🆕 `docs/architecture/v75-application-integration-20260811-2135-cursor.diff.archimate`（增量变迁 v74→v75）
  - 🆕 `docs/architecture/v75-application-integration-20260811-2135-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v75-application-integration-20260811-2135-cursor.mermaid.md`
- **已有文件（未修改）:** v74 及更早 current/target
- **变更明细:**
  - 🟢 [NEW] taskFE `PeopleRoles` + nav `people.roles` + 种子
  - 🟢 [NEW] taskTenant DELETE 单角色 + replace-all `role_names`
  - 🟡 [MODIFIED] taskFE PeopleAccess；taskAuth 种子；member/group_role 多行语义
  - 🔴 [DEPRECATED] Access 隐式 `访问·…` 建角路径
  - （文档）粗码 RequirePerm 退役路线 — 非本版删代码

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v74→Gap→WP→Plateau v75；目标拓扑：PeopleRoles→taskAuth、PeopleAccess→taskTenant 多角色 |
| **`.full.archimate`** | taskFE 组合 Roles/Access/Invite + Auth/Tenant/authz + 数据对象全量连线 |

---

## 12. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 多角色 replace-all 误清空 | UI 确认；服务端审计日志；可快速从角色列表重绑 |
| 旧书签依赖隐式角色 | 保留孤儿清理；迁移说明 |
| 邀请仍写 pending_grants | 兼容保留 + 文档引导改选角色 |
| 空粗码导致未迁 API 403 | P0 继续静默双写粗码桥接 |

---

## 13. 验收大纲（实现阶段细化）

1. 管理员可在「角色管理」创建角色并保存 region view/operate；列表/改名/删除（有绑定时 422 或强制解绑策略 — 实现选 **有引用则 422**）。
2. 访问管理可为同一成员勾选多个自定义角色；保存后 PDP 权限为并集；可移除单个角色。
3. 不再通过访问管理保存创建新的 `访问·…` 角色。
4. 无 `people.roles` page 权限者侧栏不可见；直链无权限空态（含 `data-traceId` 若请求失败）。
5. 单元：replace-all / DELETE / FE 多选；E2E：角色页 + 分配 + 受控 API region 门禁。

---

## 14. 实现分期建议

| 期 | 内容 |
|----|------|
| **M1** | 种子 + PeopleRoles CRUD/矩阵 + API Enforce region |
| **M2** | taskTenant 多角色 API + PeopleAccess 改版 |
| **M3** | 邀请选角色对齐；遗留 `访问·` 清理体验 |
| **后续** | §8 RequirePerm 退役 P1–P4 |

---

## 附录 A — 与既有文档关系

- 不取代 ADR-0003/0004；补充产品壳与绑定基数。
- 相对 `2026-08-05-rbac-custom-role-design.md`：后端角色 API 已落地，本设计补齐**当时未做的角色管理前端**，并切换授权词汇到 region（Q4）。
- 相对 v72 Access：主体矩阵上移到角色页；Access 变为分配器。
