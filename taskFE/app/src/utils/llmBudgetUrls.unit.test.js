// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  llmBudgetDefaultsUrl,
  taskModelBudgetsRaiseUrl,
  taskModelBudgetsUrl,
} from './llmBudgetUrls.js'

// 回归: e105bad 机械迁移曾把预算端点指向 /api/projects、/api/tasks（错误服务）
// 且缺失 workspace_id/ 键 → 线上 404。此处锁定 /api/cloud/ kv-last 约定。
describe('llmBudgetUrls — /api/cloud/ kv-last 约定（taskCloudService）', () => {
  it('工作区默认预算: 含 tenant_id 与 workspace_id 键', () => {
    expect(llmBudgetDefaultsUrl('c1', 'w1')).toBe(
      '/api/cloud/model-budget-defaults/tenant_id/c1/workspace_id/w1/'
    )
  })

  it('任务级预算: 含 task_id 键', () => {
    expect(taskModelBudgetsUrl('c1', 'w1', 'task1')).toBe(
      '/api/cloud/model-budgets/tenant_id/c1/workspace_id/w1/task_id/task1/'
    )
  })

  it('临时上调: model-budgets/raise 位于 kv 键值之前', () => {
    expect(taskModelBudgetsRaiseUrl('c1', 'w1', 'task1')).toBe(
      '/api/cloud/model-budgets/raise/tenant_id/c1/workspace_id/w1/task_id/task1/'
    )
  })
})
