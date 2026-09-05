/**
 * 识别 server-runtime-status「无云实例绑定」成功响应（runtime_status 为空）。
 * 用于停机/释放后将 SSE「已启动」回落到非服务态，避免两面板不一致。
 */

const NO_INSTANCE_MESSAGE_SNIPPETS = Object.freeze([
  '该任务尚未创建云实例',
  '未找到服务器配置记录',
  '启动已发起，但尚未创建评论级服务器配置',
])

const START_FAILED_MESSAGE_SNIPPETS = Object.freeze([
  '启动失败且未创建评论级服务器配置',
  '调用镜像市场 API 失败',
  '镜像服务返回错误',
])

/** 启动中（评论 CSC 已建、云实例尚未回填）文案——不得当作「无实例」回落已停止 */
const PROVISIONING_MESSAGE_SNIPPETS = Object.freeze([
  '云实例创建中，等待分配',
])

/**
 * @param {unknown} message
 * @param {{ start_failed?: boolean }|null} [extra]
 * @returns {boolean}
 */
export function isServerRuntimeStartFailedMessage(message, extra) {
  if (extra && extra.start_failed === true) return true
  const msg = String(message ?? '').trim()
  if (!msg) return false
  return START_FAILED_MESSAGE_SNIPPETS.some((s) => msg.includes(s))
}

export function isServerRuntimeNoInstanceMessage(message, extra) {
  const msg = String(message ?? '').trim()
  if (!msg && !(extra && extra.start_failed === true)) return false
  if (PROVISIONING_MESSAGE_SNIPPETS.some((s) => msg.includes(s))) return false
  if (isServerRuntimeStartFailedMessage(msg, extra)) return true
  return NO_INSTANCE_MESSAGE_SNIPPETS.some((s) => msg.includes(s))
}

/**
 * @param {unknown} data server-runtime-status JSON body
 * @returns {boolean}
 */
export function isServerRuntimeAbsentPayload(data) {
  if (!data || typeof data !== 'object') return false
  if (data.status === 'error') return false
  const rs = data.runtime_status ?? data.instance_attribute?.body?.Status
  if (String(rs ?? '').trim()) return false
  const instanceID = data.instance_id
  if (instanceID != null && String(instanceID).trim()) return false
  return isServerRuntimeNoInstanceMessage(data.message, data)
}
