// LLM 预算 API 端点（归属 taskCloudService，/api/cloud/ kv-last 约定）
// SSOT: 路由挂载见 taskCloudService src/main.go（/api/cloud/ 分派）+
// cloud_handlers.go（工作区级 model-budget-defaults / feature-params）+
// compute_handlers.go（任务级 model-budgets[/raise]）。
// 2026-08-05: 修复 e105bad 机械迁移遗留 — 此前指向 /api/projects、/api/tasks
// （错误服务，taskProjectService/taskTaskService 未实现）且缺 workspace_id/ 键。

export const llmBudgetDefaultsUrl = (tenantId, workspaceId) =>
  `/api/cloud/model-budget-defaults/tenant_id/${tenantId}/workspace_id/${workspaceId}/`

export const taskModelBudgetsUrl = (tenantId, workspaceId, taskId) =>
  `/api/cloud/model-budgets/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`

export const taskModelBudgetsRaiseUrl = (tenantId, workspaceId, taskId) =>
  `/api/cloud/model-budgets/raise/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`
