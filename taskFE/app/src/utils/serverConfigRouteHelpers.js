/**
 * ServerConfig 路由/上下文解析（纯函数，便于 composable 复用）。
 */
import { resolveTaskRouteIds } from './resolveTaskRouteIds.js'

/** 任务详情 compute API 的 taskId：props.taskId → task.id|pk → route.params.taskId */
export function resolveServerConfigTaskId({
  route,
  task,
  taskId,
  tenantId,
  workspaceId,
} = {}) {
  return resolveTaskRouteIds({
    taskId,
    task,
    tenantId,
    workspaceId,
    routeParams: route?.params,
  }).taskId
}

export function resolveTaskWorkspaceId({ route, task, workspaceId: workspaceIdProp }) {
  let workspaceId = workspaceIdProp
  if (typeof workspaceId === 'object' && workspaceId !== null) {
    workspaceId = workspaceId.id
  }
  if (!workspaceId) {
    workspaceId = route.params.workspace || route.params.workspaceId
  }
  if (!workspaceId && task?.workspace_id) {
    workspaceId = task.workspace_id
  }
  if (!workspaceId && task?.workspace) {
    workspaceId =
      typeof task.workspace === 'object' ? task.workspace.id : task.workspace
  }
  if (!workspaceId && typeof window !== 'undefined') {
    const pathSegments = window.location.pathname.split('/').filter((segment) => segment)
    if (pathSegments.length >= 4 && pathSegments[0] === 'tenant' && pathSegments[2] === 'workspace') {
      workspaceId = pathSegments[3]
    }
  }
  if (typeof workspaceId === 'object' && workspaceId !== null) {
    workspaceId = workspaceId.id
  }
  if (!workspaceId && typeof window !== 'undefined') {
    const pathSegments = window.location.pathname.split('/').filter((segment) => segment)
    if (pathSegments.length >= 5 && pathSegments[0] === 'tenant' && pathSegments[2] === 'workspace') {
      workspaceId = pathSegments[3]
    }
  }
  return String(workspaceId || '').trim()
}

export function resolveRelayContextIds({
  route,
  task,
  workspaceId: workspaceIdProp,
  taskId: taskIdProp,
  tenantId: tenantIdProp,
} = {}) {
  const ids = resolveTaskRouteIds({
    tenantId: tenantIdProp,
    workspaceId: workspaceIdProp,
    taskId: taskIdProp,
    task,
    routeParams: route?.params,
  })
  const workspaceId = ids.workspaceId || resolveTaskWorkspaceId({
    route,
    task,
    workspaceId: workspaceIdProp,
  })
  return {
    tenant_id: ids.tenantId,
    workspace_id: workspaceId,
    task_id: ids.taskId,
  }
}

export function relayToTraeApiUrl({ route, task, workspaceId, suffix, taskId, tenantId }) {
  const ctx = resolveRelayContextIds({ route, task, workspaceId, taskId, tenantId })
  const resolvedTenantId = ctx.tenant_id || route.params.tenant
  const wsId = ctx.workspace_id || workspaceId || route.params.workspace || route.params.workspaceId
  const resolvedTaskId = ctx.task_id || task?.id
  const path = String(suffix || '').replace(/^\/+/, '')
  return `/api/cloud/compute/relay-to-trae/tenant_id/${resolvedTenantId}/workspace_id/${wsId}/task_id/${resolvedTaskId}/${path}`
}
