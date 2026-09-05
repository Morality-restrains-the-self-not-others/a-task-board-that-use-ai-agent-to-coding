# 领域模型：任务关联项目查询

## Bounded Context

**任务协作 / 项目仓库** — 任务通过 TaskProject 引用 Project 及其 ProjectRepo。

## 聚合

- **TaskProject**（关联实体）：`(todo_id, project_id, repo_address)`
- **Project**（聚合根）：含 `project_repos: ProjectRepo[]`

## 应用服务

`collect_task_related_projects(scope)` — 输入 `TaskScope(tenant_id, workspace_id, task_id)`，输出 `Project[]`：

**不变量**：返回集合 = `{ tp.project | tp ∈ TaskProject WHERE todo_id = task_id }` ∩ `{ project ∈ tenant ∧ project ∈ workspace }`

## 领域事件

无（读路径语义修复）。
