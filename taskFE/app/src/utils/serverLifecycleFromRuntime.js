/**
 * 将云 Describe 的 runtime_status 映射为任务详情「服务器启动状态」可消费的生命周期意图。
 * 冷打开 / 轮询时用于对齐「服务器运行状态」与「服务器启动状态」。
 */
import { isServerRuntimeNotServingStatus } from './commentLayerZtreeUiState.js'

/** @typedef {'running' | 'starting' | 'not_serving'} RuntimeLifecycleKind */

/**
 * @param {unknown} runtimeStatus
 * @returns {{ kind: RuntimeLifecycleKind, serverStatusCode?: string } | null}
 *   null = 未知/空，调用方不得覆盖已有 SSE 状态
 */
export function resolveLifecycleFlagsFromRuntimeStatus(runtimeStatus) {
  const rs = String(runtimeStatus ?? '').trim()
  const lower = rs.toLowerCase()
  if (!lower) {
    return null
  }
  if (lower === 'running') {
    return { kind: 'running' }
  }
  if (lower === 'initializing' || lower === 'starting' || lower === 'pending') {
    return {
      kind: 'starting',
      serverStatusCode: lower === 'pending' ? 'processing' : lower,
    }
  }
  if (isServerRuntimeNotServingStatus(lower)) {
    return { kind: 'not_serving', serverStatusCode: 'stopped' }
  }
  return null
}
