import { ref } from 'vue'
import {
  formatBindingLogClock,
  upsertStartupLogLine,
} from '../../utils/bindingServerStartupLogs.js'

/** binding 状态 → 启动日志中文消息 */
export const BINDING_STATUS_LOG_MESSAGES = {
  pending: '容器调度排队中',
  waiting_previous: '等待前序任务完成',
  starting: '正在启动容器实例',
  running: '容器已就绪，服务可用',
  completed: '容器执行完成',
  failed: '容器启动失败',
  released: '容器已释放',
  cancelled: '用户已终止等待',
}

/**
 * 启动过程中的阶段细粒度日志行（真实信号驱动，非伪造）：
 * - cscAllocated：binding 挂接评论级 CSC（实例已创建/分配，等待可达性注册）
 * - heartbeatEstablished：容器 HTTP 上报首个 container_heartbeat（容器已启动、agent 运行中）
 * - probeOk：SaaS → 容器健康探测通过
 * - bidirectionalOk：容器 ↔ SaaS 双向通道建立
 *
 * OPT-20260809-010: 修复「服务器启动日志只有一条，无法了解启动详细过程」——
 * 由 starting 单状态日志扩展为多阶段时间线。
 */
export const BINDING_STAGE_LOG_MESSAGES = {
  cscAllocated: '容器实例已分配，等待服务就绪',
  heartbeatEstablished: '容器心跳已建立，服务启动中…',
  probeOk: '服务健康探测通过',
  bidirectionalOk: '双向通信通道已建立',
}

/**
 * per-binding 启动日志状态机（容器阶段 + 服务器调度进度）。
 * 从 useCommentContainerBindings 抽出以控制文件行数。
 */
export function createBindingStatusLogs() {
  /** per-binding 启动日志行（时间戳消息），key = commentId */
  const perBindingStatusLogs = ref({})

  /**
   * 为指定 binding 追加一条启动日志行（带时间戳，精确去重：同消息只保留一次）。
   * @param {string} cid - commentId
   * @param {string} msg - 日志消息正文
   */
  function appendBindingLogLine(cid, msg) {
    const current = { ...(perBindingStatusLogs.value || {}) }
    const prev = Array.isArray(current[cid]) ? current[cid] : []
    const ts = formatBindingLogClock()
    const next = upsertStartupLogLine(prev, `[${ts}] ${msg}`)
    if (next.length === prev.length && next.every((line, i) => line === prev[i])) return
    perBindingStatusLogs.value = { ...current, [cid]: next }
  }

  function appendBindingStatusLog(commentId, status) {
    const cid = String(commentId || '').trim()
    if (!cid) return
    const msg = BINDING_STATUS_LOG_MESSAGES[status]
    if (!msg) return
    appendBindingLogLine(cid, msg)
  }

  /**
   * OPT-20260809-011: 合并服务端权威启动阶段日志（list API logs 字段）。
   * @param {string} cid - commentId
   * @param {Array<{stage:string,message:string,created_at:string}>} backendLogs
   */
  function mergeBackendBindingLogs(cid, backendLogs) {
    const cidStr = String(cid || '').trim()
    if (!cidStr || !Array.isArray(backendLogs) || !backendLogs.length) return
    const current = { ...(perBindingStatusLogs.value || {}) }
    const prev = Array.isArray(current[cidStr]) ? current[cidStr] : []
    let next = prev
    for (const log of backendLogs) {
      const msg = String(log?.message || '').trim()
      if (!msg) continue
      const ts = formatBindingLogClock(log?.created_at)
      next = upsertStartupLogLine(next, ts ? `[${ts}] ${msg}` : msg)
    }
    if (next.length === prev.length && next.every((line, i) => line === prev[i])) return
    perBindingStatusLogs.value = { ...current, [cidStr]: next }
  }

  function appendBindingStageLog(commentId, stageKey) {
    const cid = String(commentId || '').trim()
    if (!cid) return
    const msg = BINDING_STAGE_LOG_MESSAGES[stageKey]
    if (!msg) return
    appendBindingLogLine(cid, msg)
  }

  /**
   * 将服务器调度进度消息写入 per-binding 启动日志（幂等去重）。
   * @param {string} commentId
   * @param {string[]} messages
   */
  function appendServerSchedulingMessages(commentId, messages) {
    const cid = String(commentId || '').trim()
    if (!cid || !Array.isArray(messages) || !messages.length) return
    for (const raw of messages) {
      const msg = String(raw || '').trim()
      if (!msg) continue
      appendBindingLogLine(cid, msg)
    }
  }

  function clearBindingStatusLogs() {
    perBindingStatusLogs.value = {}
  }

  return {
    perBindingStatusLogs,
    appendBindingLogLine,
    appendBindingStatusLog,
    appendBindingStageLog,
    mergeBackendBindingLogs,
    appendServerSchedulingMessages,
    clearBindingStatusLogs,
  }
}
