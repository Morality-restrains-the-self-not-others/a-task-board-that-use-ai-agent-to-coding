# 权限分析：工作空间任务帖人读序号

> 设计：`docs/superpowers/specs/2026-08-14-workspace-task-display-seq-design.md`

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST 创建任务（分配 `workspace_seq`） | 工作区成员（可写） | Workspace + Task | write | `verifyWorkspace` + `hasWorkspaceAccess` | ✅ | 发号在同一 tenant+workspace 事务内；不新增 path |
| GET 列表/详情（读 `workspace_seq`） | 工作区成员 | Workspace + Task | read | 既有 workspace 门禁 | ✅ | 序号非敏感；随任务 JSON |
| 搜索 `#N` / `N` | 工作区成员 | Workspace | read | 必须带 `tenant_id + workspace_id` | ⚠️ 实施时强制 | 禁止跨工作空间用序号撞库（IDOR） |
| `TASK_CREATED` 增补 seq | 系统（既有发布） | Task | write | 既有创建路径 | ✅ | 不新开消费权限 |
| DDL 清空任务帖域 | 运维 / 9999 migrate | System | write | 迁移入口，非公网 | ✅ | 不清 git 身份；不级联 CSC |

租户 page/region：**不触发**。本能力属工作区任务看板/详情，不是租户控制台新页面或新租户域 API。不新增 HTTP 路径。

## 角色与权限建模

无新角色、无新 region。复用既有工作区成员读写任务。

## 安全审查结论

- [x] IDOR：按序号搜索必须 `tenant_id + workspace_id`；不得只凭 `workspace_seq` 全局查
- [x] 权限提升：无新写接口；发号不可由客户端指定
- [x] 跨租户：SQL 必须带 tenant_id（+ workspace_id）
- [x] 403 vs 404：沿用既有任务 API（跨租户 404）
- [x] user_id 注入：无
- [x] 敏感操作：清空仅 DDL，不经公网

**评级：绿灯**（黄灯项在实施计划中关闭：搜索强制工作空间作用域）

## 权限测试清单

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 工作区成员建帖 | workspace member | POST create | 200，`workspace_seq` 递增 |
| 未认证 | 匿名 | POST create | 401 |
| 跨工作区建帖 | 他租户成员 | POST create | 403/404 |
| 搜索 `#2` 当前工作空间 | workspace member | GET search | 命中本空间 seq=2 |
| 搜索 `#2` 他空间同号 | workspace member | GET search | 不命中他空间任务 |
| 客户端传入 `workspace_seq` | workspace member | POST body 带 seq | 忽略，服务端发号 |

## 7. 权限影响分析（回写设计）

见上表。客户端不得指定 `workspace_seq`。

## 8. 角色与权限建模

无新角色。

## 9. 安全审查结论

绿灯。实施时搜索必须带工作空间，防止序号 IDOR。
