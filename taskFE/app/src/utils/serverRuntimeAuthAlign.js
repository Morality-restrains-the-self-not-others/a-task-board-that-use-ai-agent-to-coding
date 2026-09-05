/**
 * 启动状态已「已启动」但 server-runtime-status 因云平台授权缺失返回 error 时，
 * 对齐展示为 Running，避免面板显示「未知」与启动态矛盾。
 *
 * @param {{ isServerRunning?: boolean, errMsg?: unknown }} input
 * @returns {{ runtimeStatus: string, message: string } | null}
 */
export function resolveRuntimeStatusOnAuthError({ isServerRunning = false, errMsg } = {}) {
  const msg = String(errMsg ?? '').trim()
  if (!isServerRunning || !msg) {
    return null
  }
  if (!/未找到云平台授权|未配置云平台授权/.test(msg)) {
    return null
  }
  return {
    runtimeStatus: 'Running',
    message: `${msg}（启动状态显示已运行；请重新配置云平台授权以刷新实例详情）`,
  }
}
