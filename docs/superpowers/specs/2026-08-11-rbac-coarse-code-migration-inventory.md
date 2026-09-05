# 租户粗码 RequirePerm 退役 P1 盘点与 region 映射表（OPT-20260811-074）

- **作者**: nightly-opt（2026-08-12 夜间执行）
- **关联设计**: `docs/superpowers/specs/2026-08-11-tenant-role-management-v75-design.md` §8
- **范围**: 仅盘点 + 拟定迁移顺序 PR 清单；**本 OPT 不删除任何 `RequirePerm` 代码**。
- **目标**: 让 P1「`rg RequirePerm.*Perm(Member|Project|…)` 租户调用归零」有可执行映射。

---

## 1. 现状：租户粗码门禁调用点

> 平台 `RequirePlatformPerm`（`platform:*`）不在本盘点范围，永久保留。

### 1.1 Go 侧（按服务）

| 文件 | 行 | 门禁 | 粗码 | 语义 |
|------|----|------|------|------|
| `taskAuth/src/rbac_resource_groups.go` | 38 | `HasPerm` | `member:manage` | 资源组列表仅 tenant 级成员管理可见 |
| `taskAuth/src/rbac_resource_groups.go` | 202 | `HasPerm` | `member:manage` | 资源组写权限 |
| `taskAuth/src/rbac_roles.go` | 87 / 139 / 176 | `RequirePerm` | `member:manage` | 角色列表 / 创建 / 更新删除 |
| `taskTenantService/src/policy.go` | 88 / 92 | `RequirePerm`/`HasPerm` | `member:manage` | 租户成员策略判定 |
| `taskTenantService/src/rbac_tenant_roles.go` | 126 / 183 | `RequirePerm` | `member:manage` | 主体角色绑定读写 |
| `taskTenantService/src/rbac_tenant_roles.go` | 217 / 273 | `RequirePerm` | `group:manage` | 组-角色绑定 |
| `taskTenantService/src/rbac_group_admin.go` | 17 / 45 | `RequirePerm` | `member:manage` | 组管理员 CRUD |
| `taskTenantService/src/rbac_resource_group.go` | 21 / 61 | `RequirePerm` | `group-resources:manage` | 组关联资源管理 |
| `taskTenantService/src/group_handlers.go` | 187 / 190 | `HasPerm` | `member:manage` 或 `group-members:manage` | 组内成员管理（含 group_admin 组范围校验） |
| `taskBill/src/order_comments_handlers.go` | 39 / 61 | `RequirePerm` | `billing:view` / `billing:manage` | 订单评论读 / 写 |
| `taskBill/src/refund_handlers.go` | 19 | `RequirePerm` | `billing:manage` | 退款申请 |
| `taskBill/src/tenant_member.go` | 68 | `RequirePerm` | `billing:manage` | 租户计费成员口径 |
| `taskCloudService/src/budget_user_handlers.go` | 261 | `HasPerm` | `cloud:manage` | 预算用户可见性 |
| `taskCloudService/src/budget_permission_handlers.go` | 127 / 235 | `HasPerm` | `cloud:manage` | 预算权限授予 |
| `taskTaskService/src/django_validate.go` | 96 | `HasPerm` | `task:manage` | 任务校验动作门禁 |

计数：taskAuth 5 · taskTenantService 12 · taskBill 4 · taskCloudService 3 · taskTaskService 1 · taskProjectService 0。

### 1.2 FE 侧

| 文件 | 粗码 | 用途 |
|------|------|------|
| `views/PeopleManage.vue` | `hasPerm(member:manage)` | 人员管理页门禁 |
| `views/PeopleGroups.vue` | `hasPerm(member:manage \| group:manage \| group-members:manage)` | 分组页门禁 |
| `views/PeopleAccess.vue` | `hasPerm(member:manage)` | 访问管理页门禁 |
| `views/PeopleInvite.vue` | `hasPerm(member:manage)` | 邀请页门禁 |
| `views/PeopleRoles.vue` | `hasPerm(member:manage)` | 角色管理页门禁 |
| `components/MemberList.vue` | `hasPerm(member:manage)` | 成员列表操作按钮 |
| `components/Sidebar.vue` | `anyOfPerms` 回退 | 侧栏过渡（无 `page:*` 时回退粗码） |
| `domain/auth/tenantConsoleNav.js` | `anyOfPerms` | 菜单可见性（见 §2 映射列） |

---

## 2. 粗码 → ui_region/page 映射表

> 原则：读路径 → `RequireRegionView`（或 `page:` view）；写路径 → `RequireRegionOperate`。列名 `<page>.<region>` 对齐 `dataMigrate/taskAuth/032_logical_resource_groups.sql` + `034_people_roles_page.sql` 种子。

| 粗码 | 建议 page（view） | 建议 ui_region（operate） | 备注 |
|------|-------------------|---------------------------|------|
| `member:manage` | `people.manage`、`people.invite`、`people.access`、`people.roles` | `people.manage.main`、`people.access.subject_list`、`people.access.save_actions`、`people.roles.save_actions` | tenant_admin 级；`people.access.region_matrix` 为勾选区（operate） |
| `member:view` | `people.manage` | `people.manage.main`（view） | 只读成员列表 |
| `group:manage` | `people.groups` | `people.groups.main` | 分组管理（含创建） |
| `group-members:manage` | `people.groups` | `people.groups.main`（operate，**组作用域**） | 需保留 `tenant_group_admin` 组范围校验，不能只换 region 就丢范围 |
| `group-resources:manage` | `people.groups` | `people.groups.main` / 新增 `people.groups.resources` | tenant_admin 级组资源 |
| `project:manage` / `project:view` | `nav.projects` | `nav.projects.main` | 项目列表 |
| `task:manage` / `task:view` | `nav.work_panel` | `nav.work_panel.main` | 工作面板 |
| `cloud:manage` / `cloud:view` | `settings.cloud`、`nav.image_market`、`settings.feature_params` | `settings.cloud.main`、`nav.image_market.main`、`settings.feature_params.main` | 云平台绑定/镜像市场/资源配置共用 |
| `billing:manage` / `billing:view` | `billing.overview`、`billing.orders`、`billing.transactions`、`billing.usage` | `billing.*.main` | view→读、manage→写 |
| `company:manage` / `company:view` | `settings.company`、`settings.gitlab`、`settings.deliverable`、`settings.status` | `settings.*.main` | 公司/交付物/进度设置 |
| `workspace:manage` | `settings.task_panel` | `settings.task_panel.main` | 工作空间管理 |

> 若某个能力无对应 page/region（如 `group-resources` 尚无独立区域），P1 应先补 `ui_region` 种子（`INSERT IGNORE INTO auth_resource_group …`，参照 034 格式）再切 handler。

---

## 3. 迁移顺序 PR 清单（P1）

> 每步独立 PR；完成判据 = 该服务 `rg "RequirePerm\(.*Perm(Member|Project|Task|Cloud|Billing|Group|Company|Workspace)"` 归零（读路径可降级为 `RequireRegionView`）。

| 序 | PR | 内容 | 风险 | 回归 |
|----|----|------|------|------|
| 1 | 种子补齐 | 确保 `people.groups.resources` 等缺失 ui_region 有种子；`dataMigrate` 新编号迁移 | 低 | 032/034 幂等复跑 |
| 2 | `taskBill` 读路径 | `order_comments_handlers.go:39` billing:view → `RequireRegionView(billing.overview.main/billing.orders.main)` | 低（只读） | 订单评论读写 403 单测 |
| 3 | `taskBill` 写路径 | `refund_handlers.go`、`tenant_member.go`、`order_comments_handlers.go:61` billing:manage → `RequireRegionOperate(billing.*.main)` | 中 | 退款/评论写 403 单测 |
| 4 | `taskCloudService` | `budget_user_handlers.go`、`budget_permission_handlers.go` cloud:manage → `RequireRegionOperate(settings.cloud.main / nav.image_market.main / settings.feature_params.main)` | 中 | 预算权限 403 单测 |
| 5 | `taskTaskService` | `django_validate.go:96` task:manage → `RequireRegionOperate(nav.work_panel.main)` | 低 | 任务校验 403 单测 |
| 6 | `taskTenantService` 分组族 | `rbac_resource_group.go` group-resources:manage → region；`rbac_tenant_roles.go` group:manage → `RequireRegionOperate(people.groups.main)`；`group_handlers.go` 保留组范围校验 + region | 中 | group 权限 403 单测 |
| 7 | `taskTenantService` 人员族 | `policy.go`、`rbac_tenant_roles.go`、`rbac_group_admin.go` member:manage → `RequireRegionOperate(people.manage.main / people.access.* / people.roles.*)` | 高（最宽） | people_api_test 全量 |
| 8 | `taskAuth` 角色族 | `rbac_roles.go`、`rbac_resource_groups.go` member:manage → `RequireRegionOperate(people.roles.save_actions / people.access.*)` | 高 | rbac_roles_test 全量 |
| 9 | FE 侧栏过渡 | `Sidebar.vue`/`tenantConsoleNav.js` `anyOfPerms` 粗码回退改 `hasPage`/`hasRegion` | 中 | tenantConsoleNav.test / 侧栏 E2E |
| 10 | 停双写 | 关 `saveSubjectResourceAccess` 粗码双写；PDP 只发 region/page | 高（最后） | 访问管理保存→PDP 一致单测 |

**顺序原则**：只读→写；窄能力（billing/cloud/task）→宽能力（member:manage）；FE 回退移除放 handler 全迁之后，避免「有 region 无 API」空洞。

---

## 4. 建议

- 每个 PR 落地时同步更新 §2 映射表（若某能力实际映射与拟定不同）。
- 迁移期间 `member:manage` 仍须覆盖 `people.access.*` 与 `people.roles.*`，否则管理员在「访问管理/角色管理」会先失去入口（v75 角色 UI 仍以粗码兜底）。
- 该盘点为静态快照；新增 handler 若继续写粗码门禁，P1 归零判据会回退，建议在 pre-commit 加 `rg` 扫描（参照 OPT-20260811-058 思路）。
