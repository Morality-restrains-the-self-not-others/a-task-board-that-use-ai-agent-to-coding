/**
 * 任务详情「服务器启动状态」生命周期文案与样式映射（与 SSE 连接态无关）。
 *
 * 扩展新生命周期态：只改本文件 + `serverLifecycleStatus.test.js`，勿改面板组件。
 * 步骤见同目录 `serverLifecycleStatus.js.ai.md`。
 */

import { resolveLifecycleFlagsFromRuntimeStatus } from './serverLifecycleFromRuntime.js'

/** @typedef {'已启动' | '启动中' | '等待容器' | '启动失败' | '已停止' | '未启动'} ServerLifecycleLabel */

/**
 * 展示文案 → 圆点 / 文字 Tailwind class。
 * 新增展示态时在此表追加一行即可。
 * @type {Readonly<Record<ServerLifecycleLabel, { dot: string, text: string }>>}
 */
export const SERVER_LIFECYCLE_STYLES = Object.freeze({
  已启动: { dot: 'bg-green-500', text: 'text-green-600' },
  启动中: { dot: 'bg-yellow-400 animate-pulse', text: 'text-yellow-700' },
  等待容器: { dot: 'bg-amber-500 animate-pulse', text: 'text-amber-800' },
  启动失败: { dot: 'bg-red-500', text: 'text-red-600' },
  已停止: { dot: 'bg-gray-300', text: 'text-gray-500' },
  未启动: { dot: 'bg-gray-300', text: 'text-gray-500' },
})

/**
 * serverStatus 码 → 展示文案（在 isServerRunning / isServerStarting 均未命中时使用）。
 * 新增后端状态码时在此表追加即可。
 * @type {Readonly<Record<string, ServerLifecycleLabel>>}
 */
export const SERVER_STATUS_CODE_TO_LABEL = Object.freeze({
  error: '启动失败',
  stopped: '已停止',
  // success 且未 running：多为停机成功；启动成功路径会先置 isServerRunning
  success: '已停止',
  processing: '启动中',
  initializing: '启动中',
  starting: '启动中',
  sdk_call: '启动中',
})

const DEFAULT_LABEL = /** @type {ServerLifecycleLabel} */ ('未启动')
const DEFAULT_STYLE = SERVER_LIFECYCLE_STYLES[DEFAULT_LABEL]

/**
 * @param {{
 *   isServerRunning?: boolean,
 *   isServerStarting?: boolean,
 *   serverStatus?: string | null,
 *   runtimeStatus?: string | null,
 * }} input
 * @returns {ServerLifecycleLabel}
 */
export function resolveServerLifecycleLabel({
  isServerRunning = false,
  isServerStarting = false,
  serverStatus = '',
  runtimeStatus = '',
} = {}) {
  if (isServerRunning) return '已启动'
  const s = String(serverStatus || '').trim()
  // 收口失败（含容器可达超时）优先于 VM Running：实例仍可能在跑，但面板必须显示启动失败。
  if (s === 'error' || SERVER_STATUS_CODE_TO_LABEL[s] === '启动失败') {
    return '启动失败'
  }
  // 云 Describe/CSC last_runtime_status=Running 表示 VM 已起来。
  // binding 仍 starting 只代表容器尚未 register-reachability，不得再显示「启动中」。
  const fromRuntime = resolveLifecycleFlagsFromRuntimeStatus(runtimeStatus)
  if (fromRuntime?.kind === 'running') {
    return isServerStarting ? '等待容器' : '已启动'
  }
  if (isServerStarting) return '启动中'
  if (s) {
    const mapped = SERVER_STATUS_CODE_TO_LABEL[s]
    if (mapped) return mapped
  }
  if (fromRuntime?.kind === 'starting') return '启动中'
  if (fromRuntime?.kind === 'not_serving') return '已停止'
  return DEFAULT_LABEL
}

/**
 * @param {ServerLifecycleLabel | string} label
 * @returns {string}
 */
export function serverLifecycleDotClass(label) {
  return (SERVER_LIFECYCLE_STYLES[label] || DEFAULT_STYLE).dot
}

/**
 * @param {ServerLifecycleLabel | string} label
 * @returns {string}
 */
export function serverLifecycleTextClass(label) {
  return (SERVER_LIFECYCLE_STYLES[label] || DEFAULT_STYLE).text
}
