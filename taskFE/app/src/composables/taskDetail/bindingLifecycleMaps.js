/**
 * comment_container_bindings.status → 服务器生命周期标签 / 色点。
 * 与 serverLifecycleStatus.js 的标签体系对齐。
 */

const BINDING_STATUS_LIFECYCLE_MAP = {
  pending: '未启动',
  waiting_previous: '等待前序',
  starting: '启动中',
  running: '已启动',
  completed: '已完成',
  failed: '启动失败',
  released: '已停止',
  cancelled: '已终止',
}

export function mapBindingStatusToLifecycle(status) {
  const s = String(status || '').trim()
  return BINDING_STATUS_LIFECYCLE_MAP[s] || (s || '未知')
}

/** 生命周期标签 → 启动面板 serverStatus，释放/完成/终止后仍能渲染启动日志。 */
export function mapBindingLifecycleToServerStatus(lifecycle, running, starting) {
  if (lifecycle === '启动失败') return 'error'
  if (running) return 'success'
  if (starting) return 'processing'
  if (lifecycle === '已停止' || lifecycle === '已终止' || lifecycle === '已完成') return 'stopped'
  return ''
}

const LIFECYCLE_DOT_CLASS_MAP = {
  '已启动': 'bg-green-500',
  '启动中': 'bg-yellow-500',
  '等待前序': 'bg-yellow-400',
  '已完成': 'bg-green-400',
  '未启动': 'bg-gray-400',
  '启动失败': 'bg-red-500',
  '已停止': 'bg-gray-400',
  '已终止': 'bg-gray-400',
}

const LIFECYCLE_TEXT_CLASS_MAP = {
  '已启动': 'text-green-700',
  '启动中': 'text-yellow-700',
  '等待前序': 'text-yellow-600',
  '已完成': 'text-green-600',
  '未启动': 'text-gray-500',
  '启动失败': 'text-red-700',
  '已停止': 'text-gray-500',
  '已终止': 'text-gray-500',
}

export function bindingLifecycleDotClass(label) {
  return LIFECYCLE_DOT_CLASS_MAP[label] || 'bg-gray-400'
}

export function bindingLifecycleTextClass(label) {
  return LIFECYCLE_TEXT_CLASS_MAP[label] || 'text-gray-500'
}
