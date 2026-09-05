/**
 * 云 Describe runtime_status → 父组件「服务器启动状态」冷打开对齐（不写启动日志）。
 */
import { resolveLifecycleFlagsFromRuntimeStatus } from '../../utils/serverLifecycleFromRuntime.js'

export const RUNTIME_ABSENT_HYDRATE_STATUS = 'Stopped'

/** @param {Function|null|undefined} updateServerStatus */
export function notifyRuntimeHydrate(updateServerStatus, runtimeStatus) {
  if (typeof updateServerStatus !== 'function') return
  const rs = String(runtimeStatus ?? '').trim()
  if (!rs) return
  updateServerStatus({
    status: 'runtime_hydrate',
    runtime_status: rs,
  })
}

/**
 * API 明确「尚未创建云实例」且本地仍显示已启动时，用 Stopped hydrate 回落生命周期。
 * 仅在 isServerRunning 时调用，避免启动中短暂无 instance_id 误清 isServerStarting。
 * @param {Function|null|undefined} updateServerStatus
 */
export function notifyRuntimeAbsentHydrate(updateServerStatus) {
  notifyRuntimeHydrate(updateServerStatus, RUNTIME_ABSENT_HYDRATE_STATUS)
}

/**
 * 本地标志与云 Describe 失配时，应重新 hydrate：
 * - 本地未 running 但云 Running → 回填已启动
 * - 本地仍 running 但云已非服务（Stopped/Released/…）→ 清除误显示的「已在运行」
 * - 本地仍 running 但 API 明确无实例绑定（runtimeAbsent）→ 同上
 * @param {boolean} isServerRunning
 * @param {unknown} runtimeStatus
 * @param {{ runtimeAbsent?: boolean }} [opts]
 * @returns {boolean}
 */
export function shouldRematchRuntimeHydrate(isServerRunning, runtimeStatus, opts = {}) {
  if (isServerRunning && opts.runtimeAbsent) return true
  const flags = resolveLifecycleFlagsFromRuntimeStatus(runtimeStatus)
  if (!flags) return false
  if (!isServerRunning && flags.kind === 'running') return true
  if (isServerRunning && flags.kind === 'not_serving') return true
  return false
}
