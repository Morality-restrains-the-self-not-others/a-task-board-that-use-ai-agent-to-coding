/**
 * 任务详情 API 路径所需的 tenant / workspace / taskId 解析（SSOT）。
 * work-panel 弹窗等场景路由不完整，须优先使用父组件传入的 props。
 *
 * 优先级（TaskDetailContent.logic / useTaskDetail / 任务详情子面板统一走本函数）：
 * - tenant: props.tenantId → route.params.tenant → task.tenant_id
 * - workspace: props.workspaceId → route.params.workspace|workspaceId → task.workspace_id → task.workspace
 * - task: props.taskId → task.id|pk → route.params.taskId
 *
 * 禁止在任务详情链路内再手写 `props.x || route.params.x` 拼装 API 路径。
 */

function coerceId(value) {
  if (value == null || value === '') return ''
  if (typeof value === 'object') {
    const nested = value.id ?? value.pk
    return nested == null ? '' : String(nested).trim()
  }
  return String(value).trim()
}

/**
 * @param {object} input
 * @param {string|number|object|null|undefined} [input.tenantId]
 * @param {string|number|object|null|undefined} [input.workspaceId]
 * @param {string|number|null|undefined} [input.taskId]
 * @param {object|null|undefined} [input.task]
 * @param {Record<string, unknown>|null|undefined} [input.routeParams]
 * @returns {{ tenantId: string, workspaceId: string, taskId: string }}
 */
export function resolveTaskRouteIds(input = {}) {
  const routeParams = input.routeParams && typeof input.routeParams === 'object' ? input.routeParams : {}
  const task = input.task && typeof input.task === 'object' ? input.task : null

  const tenantId = coerceId(
    input.tenantId || routeParams.tenant || task?.tenant_id || '',
  )

  const workspaceId = coerceId(
    input.workspaceId ||
      routeParams.workspace ||
      routeParams.workspaceId ||
      task?.workspace_id ||
      task?.workspace ||
      '',
  )

  const taskId = coerceId(
    input.taskId || task?.id || task?.pk || routeParams.taskId || '',
  )

  return { tenantId, workspaceId, taskId }
}

export function resolveTaskRouteIdsComplete(input = {}) {
  const ids = resolveTaskRouteIds(input)
  return {
    ...ids,
    complete: Boolean(ids.tenantId && ids.workspaceId && ids.taskId),
  }
}
