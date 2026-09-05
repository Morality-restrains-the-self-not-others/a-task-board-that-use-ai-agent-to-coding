# 权限分析：任务级只标记运行中机器数与容器数

> 设计：`docs/superpowers/specs/2026-08-14-task-level-running-comment-server-count-design.md`

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `workspace-runtime-indicators` | 工作区成员 | Workspace | read | 既有 cloud compute 工作区鉴权 + tenant/workspace 路径 | ✅ | 仅扩 JSON 字段；不改主体 |
| GET `server-runtime-status` | 工作区成员 | Workspace + Comment | read | 同上 + 强制 `comment_id` | ✅ | 禁止无 comment 回退任务级；跨评论不得读他人 instance |
| 任务详情壳读两计数 | 工作区成员 | Workspace + Task | read | 页面既有 workspace 门禁 | ✅ | 计数是投影，非新敏感面 |
| `recomputeTaskRunningCounts` | 系统（同服务写路径） | Task | write | 仅内部，随评论 CSC 写触发 | ✅ | 不暴露 HTTP；按 tenant+workspace+task 限定 |
| DDL 清空任务级运行态 | 运维 / 9999 migrate | System | write | 迁移入口，非公网 | ✅ | 不兼容存量；无用户 API |

租户 page/region：**不触发**。本能力属工作区任务详情 / 看板运行态，不是租户控制台新页面或新租户域 API。不新增 HTTP 路径。

## 角色与权限建模

无新角色、无新 region。复用既有工作区成员读云资源。

## 安全审查结论

- [x] IDOR：indicators 按路径 `tenant_id`+`workspace_id` 扫描；runtime-status 必须带 `comment_id`，禁止用任务级 instance 冒充评论
- [x] 权限提升：无写接口变更；计数列为服务端投影
- [x] 跨租户：SQL 必须带 tenant_id（+ workspace_id + task_id）
- [x] 403 vs 404：沿用既有 cloud compute 行为
- [x] user_id 注入：无
- [x] 敏感操作：清空任务级 instance 仅 DDL，不经公网

**评级：绿灯**

## 权限测试清单

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 工作区成员读 indicators | workspace member | GET indicators | 200，含两计数 |
| 未认证 | 匿名 | GET indicators | 401 |
| 跨工作区 | 他租户成员 | GET indicators | 403 |
| runtime-status 无 comment_id | 工作区成员 | GET runtime-status | 400 |
| runtime-status 他评论 comment_id | 工作区成员（同任务） | GET | 只返回该评论 CSC，不泄露任务级旧 instance |
