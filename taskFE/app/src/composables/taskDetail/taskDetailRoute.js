/** 任务详情路由需保留的 query 键（Fork / 新标签打开等） */
export const TASK_DETAIL_PRESERVED_QUERY_KEYS = Object.freeze([
  'accessCode',
  'relayToTrae',
  'github',
])

/** 从当前 route.query 提取任务详情应携带的 query */
export function buildTaskDetailRouteQuery(routeQuery = {}) {
  const q = {}
  for (const key of TASK_DETAIL_PRESERVED_QUERY_KEYS) {
    const raw = routeQuery[key]
    if (raw == null || raw === '') continue
    q[key] = Array.isArray(raw) ? String(raw[0]) : String(raw)
  }
  return q
}

/** 在新标签页打开任务详情；返回 window.open 结果（被拦截时为 null） */
export function openTaskDetailInNewTab(router, { tenantId, workspaceId, taskId, query = {} }) {
  if (!router?.resolve || typeof window === 'undefined' || typeof window.open !== 'function') {
    return null
  }
  const resolved = router.resolve({
    name: 'task_detail',
    params: {
      tenant: String(tenantId),
      workspaceId: String(workspaceId),
      taskId: String(taskId),
    },
    query: query || {},
  })
  const path = resolved.href || resolved.fullPath || ''
  const href = /^https?:\/\//i.test(path) ? path : `${window.location.origin}${path}`
  return window.open(href, '_blank', 'noopener,noreferrer')
}
