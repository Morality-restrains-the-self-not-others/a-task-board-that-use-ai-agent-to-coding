import { isCloudServerStopLogLine } from './stopVmSseMessage.js'

/**
 * 将任务级「服务器调度启动进度」路由到评论 per-binding 启动日志。
 *
 * 背景：评论面板有 binding 时仅展示容器阶段日志（排队/启动实例/分配…），
 * 云服务器调度 SSE（前置资源、RunInstances、成功/失败）原本只写入任务级 statusLogs，
 * 导致详情内「启动日志」看不到服务器调度过程。
 */

/**
 * 是否应将本条 SSE/轮询状态路由到指定评论的 binding 启动日志。
 * @param {object|null|undefined} statusData
 * @returns {string} commentId；不可路由时返回 ''
 */
export function resolveServerStartupBindingCommentId(statusData) {
  if (!statusData || typeof statusData !== 'object') return ''
  const cid = String(statusData.comment_id || statusData.commentId || '').trim()
  if (!cid) return ''
  // 容器心跳 / 层图 / relay 等非服务器调度事件不写入「启动日志」
  // userdata_boot（UserData / init_from_task2app.sh boot-progress）须保留
  const status = String(statusData.status || '').trim().toLowerCase()
  const phase = String(statusData.phase || '').trim().toLowerCase()
  if (phase === 'userdata_boot') {
    return cid
  }
  if (
    status === 'container_heartbeat' ||
    status === 'binding_advanced' ||
    status === 'relay_to_trae_status' ||
    status === 'ai_instruct_stream' ||
    status === 'runtime_hydrate' ||
    status.startsWith('container_')
  ) {
    return ''
  }
  return cid
}

/**
 * 从 statusData 提取应写入 binding 启动日志的消息正文（不含时间戳前缀）。
 * 与 pushServerStatusLogLines 展示内容对齐（主消息 + sdk_call 明细）。
 * @param {object|null|undefined} statusData
 * @param {string} [displayMessage]
 * @returns {string[]}
 */
export function collectServerStartupLogMessages(statusData, displayMessage) {
  const messages = []
  const msg =
    typeof displayMessage === 'string' && displayMessage.trim()
      ? displayMessage.trim()
      : String(statusData?.message || '').trim()
  if (msg) messages.push(msg)

  if (statusData?.status === 'sdk_call' && statusData.sdk_call) {
    const sdkCall = statusData.sdk_call
    const logLabel =
      typeof statusData.log_label === 'string' && statusData.log_label.trim()
        ? statusData.log_label.trim()
        : ''
    const sdkPrefix = logLabel ? `${logLabel} ` : ''
    if (sdkCall.method) {
      messages.push(`${sdkPrefix}SDK调用: ${sdkCall.method}`)
    }
    if (Array.isArray(sdkCall.request_ids) && sdkCall.request_ids.length) {
      messages.push(`${sdkPrefix}Request IDs: ${sdkCall.request_ids.join(', ')}`)
    }
  }
  return messages
}

/**
 * 过滤任务级 statusLogs 中属于某评论的服务器调度行（按容器名 / commentId / instanceId 匹配）。
 * 用作 live 总线之外的展示兜底（例如轮询未带 comment_id 但消息含容器名）。
 *
 * 兼容 UserData 脚本硬编码 CONTAINER_NAME=task2app-container、以及 token 落到任务级 CSC
 * 导致 SSE 无 comment_id 时，日志标签形如 [i-xxx、task2app-container]。
 *
 * @param {string[]} taskLogs
 * @param {{ commentId?: string, containerName?: string, instanceId?: string, soleActiveBinding?: boolean }} opts
 * @returns {string[]}
 */
export function filterTaskStatusLogsForBinding(taskLogs, opts = {}) {
  const list = Array.isArray(taskLogs) ? taskLogs : []
  const cid = String(opts.commentId || '').trim()
  const cname = String(opts.containerName || '').trim()
  const instanceId = String(opts.instanceId || '').trim()
  const sole = opts.soleActiveBinding === true
  if (!cid && !cname && !instanceId && !sole) return []
  return list.filter((line) => {
    if (typeof line !== 'string') return false
    if (cname && line.includes(cname)) return true
    // UserData / init_from_task2app.sh 默认 docker 名（与 mock task_<task>_<comment> 不同）
    if (line.includes('task2app-container')) return true
    if (instanceId && line.includes(instanceId)) return true
    // 容器名约定 task_<taskId>_<commentId>；亦匹配 label 中的 commentId 片段
    if (cid && (line.includes(`_${cid}`) || line.includes(`、${cid}`) || line.includes(`/${cid}`))) {
      return true
    }
    // 单活跃 binding：仅吸入「无其他评论容器标签」的云调度/UserData 行，避免串台
    if (sole) {
      const labelMatch = line.match(/\[([^\]、]+)、([^\]]+)\]/)
      if (labelMatch) {
        const labelName = String(labelMatch[2] || '').trim()
        // 明确属于其他评论的 task_*_<otherCid> 标签跳过
        if (labelName.startsWith('task_') && cid && !labelName.includes(cid) && labelName !== 'task2app-container') {
          return false
        }
        if (labelName === '-' || labelName === 'task2app-container' || (cname && labelName === cname)) {
          return true
        }
        return false
      }
      if (isCloudServerStopLogLine(line)) {
        return false
      }
      if (/(?:安装基础依赖|容器运行时|拉取容器镜像|开始容器初始化|启动 docker|aliyun|服务器启动|准备启动)/.test(line)) {
        return true
      }
    }
    return false
  })
}

/**
 * 合并 binding 日志与任务级服务器调度日志，按规范化正文去重（忽略时钟与 trace_id= 后缀）。
 * 冲突时保留带 trace_id= 的行。
 * @param {string[]} bindingLogs
 * @param {string[]} serverLogs
 * @returns {string[]}
 */
export function mergeBindingAndServerStartupLogs(bindingLogs, serverLogs) {
  let out = Array.isArray(bindingLogs) ? [...bindingLogs] : []
  const extra = Array.isArray(serverLogs) ? serverLogs : []
  for (const line of extra) {
    out = upsertStartupLogLine(out, line)
  }
  return out
}

/** 去掉 `[时:分:秒]` 前缀与末尾 `trace_id=`，用于同一 UserData 步骤去重。 */
export function startupLogDedupeKey(line) {
  if (typeof line !== 'string') return ''
  return line
    .replace(/^\[[^\]]+\]\s*/, '')
    .replace(/\s+trace_id=\S+\s*$/i, '')
    .trim()
}

/**
 * MySQL naive UTC DATETIME（无时区）与 RFC3339 Z 视为同一瞬间。
 * @param {string} rawTs
 * @returns {Date}
 */
export function parseBindingLogDate(rawTs) {
  const s = String(rawTs || '').trim()
  if (!s) return new Date(NaN)
  if (/^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}$/.test(s)) {
    return new Date(`${s.replace(' ', 'T')}Z`)
  }
  return new Date(s)
}

export function formatBindingLogClock(rawTs, now = new Date()) {
  const d = rawTs ? parseBindingLogDate(rawTs) : now
  const src = d && !Number.isNaN(d.getTime()) ? d : now
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(src.getHours())}:${pad(src.getMinutes())}:${pad(src.getSeconds())}`
}

export function upsertStartupLogLine(lines, line) {
  const list = Array.isArray(lines) ? [...lines] : []
  if (typeof line !== 'string' || !line.trim()) return list
  const key = startupLogDedupeKey(line)
  if (!key) return list
  const idx = list.findIndex((l) => startupLogDedupeKey(l) === key)
  if (idx < 0) {
    list.push(line)
    return list
  }
  if (line.includes('trace_id=') && !list[idx].includes('trace_id=')) {
    list[idx] = line
  }
  return list
}
