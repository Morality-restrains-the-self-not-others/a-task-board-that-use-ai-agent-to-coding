/**
 * 创建任务：自动运行勾选后是否展示「加入自动调度队列」。
 */

export function shouldShowCreateTaskQueuedAutoRun({
  canEnableAutoRun = false,
  autoRun = false,
  workspaceScheduleEnabled = false,
} = {}) {
  return Boolean(canEnableAutoRun && autoRun && workspaceScheduleEnabled)
}

export function resolveQueuedAutoRunHint() {
  return '工作空间已启用自动调度。勾选后任务进入排队，按调度时段逐个启动，创建后不立即启服。不勾选则仍立即按运行模版启动。'
}

/**
 * 自动运行开启时把 queued_auto_run 写入创建/保存 payload。
 * @param {Record<string, unknown>} payload
 * @param {{ auto_run?: unknown, queued_auto_run?: unknown }} task
 */
export function appendQueuedAutoRunToCreatePayload(payload, task) {
  const out = payload && typeof payload === 'object' ? payload : {}
  if (task?.auto_run !== true) return out
  out.queued_auto_run = task.queued_auto_run === true
  return out
}
