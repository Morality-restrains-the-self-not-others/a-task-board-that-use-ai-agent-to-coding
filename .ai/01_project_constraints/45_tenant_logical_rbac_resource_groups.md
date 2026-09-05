# 租户逻辑资源组 RBAC（页面组 ⊃ UI Region）强制参照（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-11
- 最后修改：2026-08-11
- 维护者：Trae AI 团队
- 适用范围：整个 monorepo（凡触及**租户控制台能力**、**租户域 API**、**租户侧栏/页面可见性**、**主体授权**的改动）
- 架构决策：**ADR-0003**（accepted）
- 权威设计：`docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md`

## 背景（为何是元规则）

租户控制台曾出现「仅前端按菜单/粗码隐藏」与「页面=资源组」的错误模型，导致 API 可绕过、页内分区无法表达。v72 已锁定：

```
页面组 (page) = 载体
  └── UI 组件区域 (ui_region) = 默认可授予的资源组
        ├── UI 组件
        └── API 端点
```

若后续租户功能仍只加粗码、只改 FE、或不登记 region 成员，将再次出现权限漂移。**本元规则要求：凡租户相关能力变更，必须按本模型做对应调整，不得绕开。**

与业务表 `tenant_resource_group_assignment`（实体→人员小组的数据范围）**正交**，禁止混名/混表。

## 核心原则

1. **页面组 ≠ 资源组**；默认可授予粒度是 **ui_region**（A1：整页勾选可写 page，PDP 展开子 region）。
2. **前后端同源 Enforce**：BE 用 `RequireRegion`（或 registry 反查 API→region）；FE 用 `hasRegion` / `hasPage`。仅隐藏 UI **不算完成**。
3. **PDP 注入** `region:<group_key>` / `page:<group_key>`；本授权链路**不**从 Region 展开旧粗码（B2）。存量 `RequirePerm(粗码)` 可过渡并存，但新能力不得以粗码作为唯一门禁。
4. **tenant_admin** 拥有全部系统 page/region（PDP 特判或种子绑定）。
5. **新租户页面 / 新区块 / 新租户 API** 必须登记到 `auth_resource_group` / `auth_resource_member`，并在访问管理树中可见。

## 触发条件（必须执行本规则检查清单）

以下任一改动触发：

| 触发 | 示例 |
|------|------|
| 租户控制台新页面或新侧栏项 | `/tenant/:id/settings/xxx`、`TENANT_CONSOLE_NAV` 增项 |
| 租户页内新功能区 / 危险操作区 | 只读列表区 vs 保存/删除区 |
| 新租户域 HTTP API（含 internal 被控制台调用） | `taskTenant*`、`taskAuth` 角色/成员、计费/云设置等租户路径 |
| 修改既有租户 API 的鉴权方式 | 新增/改 `RequirePerm`、去掉鉴权、改 company 范围 |
| 访问管理 / 角色 / 成员-角色绑定 | PeopleAccess、member-role、自定义角色 |
| 租户侧栏 / 路由门禁 / 无权限空态 | Sidebar、`canSeeMenuKey`、`usePermissions` |

**不触发（仍建议心智对齐）：** 纯平台超管控制台、与租户无关的基础设施、纯文档 typo。

## 强制检查清单（Agent / 开发者提交前）

对每次触发改动，**必须**完成（不适用项注明理由）：

### A. 目录与种子（taskAuth）

- [ ] 新页面 → `auth_resource_group`：`kind=page`，`group_key` 稳定（建议对齐 `TENANT_CONSOLE_NAV.key`）
- [ ] 页内能力区 → 至少一个 `kind=ui_region`，`parent_id` 指向该 page
- [ ] 相关 UI key / API `METHOD path` → `auth_resource_member`（**仅**挂在 ui_region）
- [ ] DDL/seed 放在 `dataMigrate/taskAuth/`（下一批次编号），经 9999 / migrate 生效，**禁止**业务启动路径建表

### B. 后端 Enforce

- [ ] 新/改租户 API：优先 `authz.RequireRegion(w, r, regionKey, companyID)`
- [ ] 过渡期若仍用 `RequirePerm`：须在 PR/提交说明中标注迁移计划，且 FE 不得假装已完成 region 门禁
- [ ] 校验公司归属 / 防 IDOR（与既有 RBAC 一致）

### C. 前端

- [ ] 侧栏：`canSeeMenuKey` / `hasPage(pageKey)`（允许无 `page:*` 时粗码回退）
- [ ] 页内区块/按钮：`hasRegion(regionKey)`；无权限空态（含 `data-traceId` 若请求失败）
- [ ] 访问管理树能展示并勾选新 region（依赖目录 API，无需硬编码全树亦可，但种子必须存在）

### D. 文档与意图

- [ ] 需求与 `docs/intents/` 中访问/权限相关意图不一致时，同步更新意图与测试意图
- [ ] 架构影响大时更新 ArchiMate / ADR（见 ADR 元规则）；本模型变更须改 ADR-0003 或 supersede

## 禁止事项

| 禁止 | 原因 | 正确做法 |
|------|------|----------|
| 仅 FE 隐藏菜单/按钮当作授权完成 | 可绕过 | BE `RequireRegion` |
| 把「页面」当作唯一资源组、叶子直接挂 page | 无法页内分区 | 叶子挂 ui_region |
| 复用 `tenant_resource_group_assignment` 做页面 ACL | 语义是业务数据范围 | 用 `auth_*` 逻辑资源组表 |
| 新租户 API 无任何鉴权 / 仅靠「知道 URL」 | 安全事故 | Region 或明确过渡粗码 + 计划 |
| 平行 ACL 绕过 taskAuth PDP | 与 v63/v72 双轨 | 扩展 PDP + `X-Tenant-Perms` |
| 从 Region 种子「自动展开」旧粗码当作完成标准 | 违反 B2 | 粗码双写仅作过渡桥接，Enforce 以 region/page 为准 |

## 关键落点（SSOT 路径）

| 层 | 路径 |
|----|------|
| ADR | `docs/adr/0003-logical-resource-group-page-region-rbac.md` |
| 设计 | `docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md` |
| DDL | `dataMigrate/taskAuth/032_logical_resource_groups.sql`（及后续增量） |
| PDP | `taskAuth/src/rbac_pdp.go`、`rbac_resource_groups.go` |
| Enforce | `shareLib/authz`（`RequireRegion` / `HasRegion` / `HasPage`） |
| FE 导航 | `taskFE/app/src/domain/auth/tenantConsoleNav.js` |
| 访问管理 | `taskFE/app/src/views/PeopleAccess.vue`、`saveSubjectResourceAccess.js` |
| 意图 | `docs/intents/frontend/tenant_logical_resource_group_access.*` |

## 与粗码（v63）的关系

- **并存过渡**：存量 handler 的 `RequirePerm(project:view)` 等可保留。
- **新租户能力**：必须挂 region；侧栏以 `page:*` 为准。
- **访问管理保存**：可双写粗码作桥接，但**不得**把「只写粗码、不绑 `auth_role_resource_group`」当作新默认路径。
- **授予效果（v73 / ADR-0004）**：`auth_role_resource_group.effect` 为 `view`（可访问）或 `operate`（可编辑执行，蕴含 view）。PDP 注入 `region:<key>:view|operate`；operate 另发遗留 `region:<key>`。写 API 用 `RequireRegionOperate`；读路径用 `RequireRegionView` / `HasRegionView`。

## Code Review 门禁（人工）

Reviewer 第一问：「本次租户改动的 page/region/api 成员登记了吗？对应 API 是否 `RequireRegion`（写路径是否 `RequireRegionOperate`）？」

PR 描述建议勾选本文件「强制检查清单」A–D。

## CI（演进）

- P2（跟踪 OPT-20260811-032）：静态检查新路由是否出现在某 region 的 `api` 成员中。
- 本条生效不依赖该 CI；**Agent 必须在无 CI 时仍执行清单**。

## Cursor / 加载

- Cursor 元规则：`.cursor/rules/tenant-logical-rbac-resource-groups.mdc`（alwaysApply）
- 约束索引：`00_project_constraints.md` 第 40 条
- 根摘要：仓库根 `.ai.md`
- Companion：`shareLib/authz/ai.md`
- ADR-0004：`docs/adr/0004-resource-group-grant-effect-view-operate.md`

## 验收（会话自检）

```bash
# 设计与 ADR 存在
test -f docs/adr/0003-logical-resource-group-page-region-rbac.md
test -f docs/adr/0004-resource-group-grant-effect-view-operate.md
test -f docs/superpowers/specs/2026-08-11-rbac-resource-grant-effect-v73-design.md

# Enforce API 存在
rg -n 'func RequireRegionOperate|func HasRegionView' shareLib/authz/middleware.go

# 种子表迁移存在
ls dataMigrate/taskAuth/*logical_resource* dataMigrate/taskAuth/*grant_effect* 2>/dev/null | head
```
