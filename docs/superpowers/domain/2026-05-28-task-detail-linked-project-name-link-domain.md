# Domain Model: 任务详情关联项目名称链接

## 结论

**跳过领域层实现** — 增量为前端 SPA 导航，无新业务概念、无持久化、无跨上下文事件。

## 轻量映射（文档用）

| 概念 | 角色 |
|------|------|
| Task（任务） | 展示关联 Project 列表 |
| Project（项目） | 导航目标实体 |
| ProjectDetailRoute | 值对象：`/tenant/{tenantId}/projects/{projectId}/` |

## 契约

- 输入：`tenantId`, `project_id`, `project_name`（已有）
- 输出：用户导航至项目详情页
