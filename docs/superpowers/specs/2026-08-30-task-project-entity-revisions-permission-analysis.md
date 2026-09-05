# 权限分析：任务与项目内容历史版本

- 日期：2026-08-30
- 设计：`docs/superpowers/specs/2026-08-30-task-project-entity-revisions-design.md`
- 结论：**绿灯** — 复用既有实体读/写门禁，不新增角色

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 创建任务写 v1 | 可创建任务的工作空间成员 | Workspace / Task | write | `hasWorkspaceAccess` | ✅ | 同事务副作用 |
| PATCH 任务写下一版 | `canMutateTaskFromRequest` | Task | write | 与 handleUpdateTask 相同 | ✅ | 仅字段变化时写 |
| GET task revisions 列表/详情 | 可 GET 该任务者 | Task | read | `hasWorkspaceAccess` + tenant 匹配 | ✅ | revision 须属于 path task_id |
| 创建/更新项目写 revision | 可改该项目者 | Tenant / Project | write | `rejectIfProjectNotInTenant` | ✅ | 同更新门禁 |
| GET project revisions | 可 GET 该项目者 | Project | read | 与 handleGetProject 相同 | ✅ | revision 须属于 path project_id |
| Kafka 消费 1_observe | taskEvents worker | Platform | observe | 无用户身份 | ✅ | 不写业务库 |
| FE 历史面板 | 已能打开详情页的用户 | UI | read | 页面可见性 + API 403 | ✅ | 登记 resource_member |

## 角色建模

不新增角色。登记 API 成员到既有 region：

- `nav.work_panel.main` ← 任务 revisions GET
- `nav.projects.main` ← 项目 revisions GET

tenant_admin 已绑定全部系统 page。

## 安全审查

- [x] **IDOR**：GET 校验 tenant + entity_id；错绑 revisionId → 404
- [x] **写权限**：无独立写 API；插入仅随已授权创建/更新
- [x] **跨租户**：WHERE tenant_id=?
- [x] **403 vs 404**：无工作空间访问已知任务 → 403；任务不存在或租户不符 → 404（与现网 GET 任务一致）
- [x] **user_id 注入**：actor 取网关 `X-User-Id`，禁止 body 指定
- [x] **敏感操作**：无删除 revision；无还原写
- [x] **PII**：事件与日志不含正文全文

## 权限测试清单

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 工作空间成员 | member | GET 本任务 revisions | 200 |
| 无工作空间权限 | outsider | GET | 403 |
| 其它租户 ID | other tenant | GET 路径 tid 不匹配 | 404 |
| 其它任务 revisionId | member | GET | 404 |
| 模拟登录 | impersonator | PATCH 改标题 | 快照 actor=被模拟用户；日志带 impersonation 字段 |

## 风险

- **中**：存量无历史 — UI 说明
- **低**：viewer 若不能 PATCH 则不能「制造」历史，但仍可读已有快照（与读任务一致）
