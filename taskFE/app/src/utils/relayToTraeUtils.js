import { isLoopbackHttpUrl, replaceHttpUrlHostname } from './httpUrlHost.js'
import { TASK2APP_ACCESS_TOKEN_PLACEHOLDER } from './userdataContainerImageReplace.js'

export { TASK2APP_ACCESS_TOKEN_PLACEHOLDER }

export const RELAY_TO_TRAE_ENV_KEYS = Object.freeze([
  'TASK_API_ENDPOINT_ORIGIN',
  'BUSINESS_API_ENDPOINT_ORIGIN',
  'ACCESS_TOKEN',
  'DEBUG_AGENT',
])

/** relayToTrae 直接启动时 TASK_API 默认 origin（本地 Django） */
export const DEFAULT_TASK_API_ENDPOINT_ORIGIN = 'http://localhost:8001'

export function shouldApplyRelayToTraeStatusLogs(currentTaskId, statusPayload) {
  const current = String(currentTaskId || '').trim()
  const active = String(statusPayload?.active_task_id || '').trim()
  const logTask = String(statusPayload?.log_task_id || active || '').trim()
  if (!logTask) {
    return false
  }
  return logTask === current
}

/**
 * 解析 go-relay /v1/status 的 next_cursor（日志增量游标）。
 * @param {unknown} payload
 * @returns {number|null}
 */
export function parseRelayStatusNextCursor(payload) {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return null
  }
  const raw = payload.next_cursor ?? payload.nextCursor
  if (raw == null || raw === '') {
    return null
  }
  const n = Number(raw)
  if (!Number.isFinite(n) || n < 0) {
    return null
  }
  return Math.floor(n)
}

/**
 * 根据 status 响推进本地日志游标。
 * go-relay 会把过大的 cursor clamp 到 len(logs)；若请求游标大于 next_cursor，
 * 说明服务端缓冲已重置，应从 0 全量重拉一次。
 *
 * @param {unknown} currentCursor
 * @param {unknown} payload
 * @param {unknown} [requestedCursor]
 * @returns {{ cursor: number, shouldRefetchFromZero: boolean }}
 */
export function advanceRelayStatusLogCursor(currentCursor, payload, requestedCursor = currentCursor) {
  const current = Math.max(0, Math.floor(Number(currentCursor) || 0))
  const requested = Math.max(0, Math.floor(Number(requestedCursor) || 0))
  const next = parseRelayStatusNextCursor(payload)
  if (next == null) {
    return { cursor: current, shouldRefetchFromZero: false }
  }
  if (requested > next) {
    return { cursor: 0, shouldRefetchFromZero: true }
  }
  return { cursor: next, shouldRefetchFromZero: false }
}

/**
 * 构造 relay-to-trae/status 查询参数（task_id + 可选 cursor）。
 * cursor=0 时省略，与 go-relay「缺省从 0」一致，并减少无意义 query。
 *
 * @param {{ taskId?: string, cursor?: number }} [options]
 * @returns {URLSearchParams}
 */
export function buildRelayToTraeStatusSearchParams(options = {}) {
  const params = new URLSearchParams()
  const taskId = String(options.taskId || '').trim()
  if (taskId) {
    params.set('task_id', taskId)
  }
  const cursor = Math.max(0, Math.floor(Number(options.cursor) || 0))
  if (cursor > 0) {
    params.set('cursor', String(cursor))
  }
  return params
}

/**
 * onlineServiceJS 引导里程碑行（含 BOOTSTRAP_COMPLETE / BOOTSTRAP_FAILED）。
 * 供启动日志面板高亮，避免误判为卡在「拉取任务详情」。
 * @param {unknown} line
 * @returns {'complete'|'failed'|'phase'|null}
 */
export function classifyBootstrapMilestoneLogLine(line) {
  const text = String(line ?? '')
  if (!text.includes('[onlineServiceJS]')) return null
  if (text.includes('BOOTSTRAP_COMPLETE') || text.includes('任务引导完成')) return 'complete'
  if (text.includes('BOOTSTRAP_FAILED') || text.includes('bootstrap (post-listen) error')) return 'failed'
  if (text.includes('BOOTSTRAP_PHASE=')) return 'phase'
  return null
}

/**
 * 换票落盘失败日志：FAIL_PERSIST / TOKEN_PERSIST_FAILED / token-persist: FAIL。
 * 与 TOKEN_ACCESS_INVALID（令牌无效）区分，供启动日志高亮与状态条文案。
 * @param {unknown} line
 * @returns {boolean}
 */
export function isTokenPersistFailedLogLine(line) {
  const text = String(line ?? '')
  if (!text) return false
  if (/FAIL_PERSIST/i.test(text)) return true
  if (/TOKEN_PERSIST_FAILED/i.test(text)) return true
  if (/token-persist:\s*FAIL/i.test(text)) return true
  return false
}

/**
 * 将 relay status.error / 启动失败 message 映射为启动按钮旁中文提示。
 * 识别 TOKEN_PERSIST_FAILED，避免直接展示英文 state.Error。
 * @param {unknown} rawError
 * @param {string} [fallback]
 * @returns {string}
 */
export function mapRelayToTraeStatusErrorMessage(rawError, fallback = '') {
  const text = String(rawError ?? '').trim()
  if (!text) return String(fallback || '')
  if (/TOKEN_PERSIST_FAILED/i.test(text) || isTokenPersistFailedLogLine(text)) {
    return '换票落盘失败：请检查 ONLINE_PROJECT_STATE_ROOT 磁盘权限后重试'
  }
  return text
}

/**
 * 启动日志行分类（引导里程碑 + 换票落盘失败）。
 * @param {unknown} line
 * @returns {'complete'|'failed'|'phase'|'token_persist_failed'|null}
 */
export function classifyRelayStartupLogLine(line) {
  if (isTokenPersistFailedLogLine(line)) return 'token_persist_failed'
  return classifyBootstrapMilestoneLogLine(line)
}

const BOOTSTRAP_PHASE_LABELS = Object.freeze({
  task_detail_begin: '拉取任务详情',
  clone_begin: '项目克隆',
  feature_params_begin: '拉取智能体资源配置',
  post_listen: '启动后引导',
  task_detail_or_credentials: '任务详情/凭证',
  clone: '项目克隆',
  feature_params_env: '智能体资源配置环境',
  token_persist: '换票落盘',
})

/**
 * @param {string} text
 * @returns {string}
 */
function extractBootstrapPhaseKey(text) {
  const phaseEq = text.match(/BOOTSTRAP_PHASE=([a-z0-9_]+)/i)
  if (phaseEq) return phaseEq[1]
  const failedPhase = text.match(/BOOTSTRAP_FAILED\s+phase=([a-z0-9_]+)/i)
  if (failedPhase) return failedPhase[1]
  return ''
}

/**
 * @param {string} phaseKey
 * @returns {string}
 */
function bootstrapPhaseLabel(phaseKey) {
  const key = String(phaseKey || '').trim()
  if (!key) return ''
  return BOOTSTRAP_PHASE_LABELS[key] || key
}

/**
 * 从启动日志推导引导状态条（含失败独立徽章，不仅依赖红字行）。
 * 优先级：failed > complete > 最近 phase > idle。
 * @param {unknown[]} lines
 * @returns {{
 *   kind: 'idle'|'running'|'complete'|'failed',
 *   label: string,
 *   phaseKey: string,
 *   detail: string,
 * }}
 */
export function resolveBootstrapStatusFromLogs(lines) {
  const list = Array.isArray(lines) ? lines : []
  let kind = 'idle'
  let phaseKey = ''
  let detail = ''
  let label = ''

  for (const raw of list) {
    const text = String(raw ?? '')
    if (isTokenPersistFailedLogLine(text)) {
      kind = 'failed'
      phaseKey = 'token_persist'
      detail = text.replace(/^\[(onlineServiceJS|relayToTrae)\]\s*/, '').trim()
      label = '换票落盘失败'
      continue
    }
    const milestone = classifyBootstrapMilestoneLogLine(text)
    if (!milestone) continue
    if (milestone === 'failed') {
      kind = 'failed'
      phaseKey = extractBootstrapPhaseKey(text) || phaseKey
      detail = text.replace(/^\[onlineServiceJS\]\s*/, '').trim()
      label = phaseKey
        ? `引导失败（${bootstrapPhaseLabel(phaseKey)}）`
        : '引导失败'
      continue
    }
    if (kind === 'failed') continue
    if (milestone === 'complete') {
      kind = 'complete'
      phaseKey = ''
      detail = text.replace(/^\[onlineServiceJS\]\s*/, '').trim()
      label = '引导完成'
      continue
    }
    if (milestone === 'phase' && kind !== 'complete') {
      kind = 'running'
      phaseKey = extractBootstrapPhaseKey(text) || phaseKey
      detail = text.replace(/^\[onlineServiceJS\]\s*/, '').trim()
      label = phaseKey
        ? `引导中：${bootstrapPhaseLabel(phaseKey)}`
        : '引导中'
    }
  }

  return { kind, label, phaseKey, detail }
}

/**
 * 容器心跳探测 / 心跳上报相关日志：应从「启动日志」剥离，改在「容器连接状态」展示。
 * @param {unknown} line
 * @returns {boolean}
 */
export function isContainerHeartbeatLogLine(line) {
  const text = String(line ?? '')
  if (!text) {
    return false
  }
  if (text.includes('saas-heartbeat-probe')) {
    return true
  }
  if (text.includes('/server-container-token/heartbeat/')) {
    return true
  }
  try {
    const trimmed = text.trim()
    if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
      const obj = JSON.parse(trimmed)
      if (obj && typeof obj === 'object') {
        const path = String(obj.path || obj.url || '')
        if (path.includes('saas-heartbeat-probe') || path.includes('/server-container-token/heartbeat/')) {
          return true
        }
      }
    }
  } catch {
    // not JSON
  }
  return false
}

/**
 * 将日志行拆成「启动日志」与「心跳/连接状态」两类。
 * @param {unknown[]} lines
 * @returns {{ startupLines: string[], heartbeatLines: string[] }}
 */
export function partitionRelayToTraeLogLines(lines) {
  const startupLines = []
  const heartbeatLines = []
  if (!Array.isArray(lines)) {
    return { startupLines, heartbeatLines }
  }
  for (const line of lines) {
    const text = String(line)
    if (isContainerHeartbeatLogLine(text)) {
      heartbeatLines.push(text)
    } else {
      startupLines.push(text)
    }
  }
  return { startupLines, heartbeatLines }
}

function hasRelayLogPrefix(logs, prefix) {
  if (prefix.length > logs.length) {
    return false
  }
  for (let i = 0; i < prefix.length; i += 1) {
    if (logs[i] !== prefix[i]) {
      return false
    }
  }
  return true
}

/**
 * 若 needle 是 haystack 的连续子序列，返回起始下标；否则 -1。
 * @param {string[]} haystack
 * @param {string[]} needle
 * @returns {number}
 */
function indexOfContiguousSubsequence(haystack, needle) {
  if (!Array.isArray(needle) || needle.length === 0) {
    return 0
  }
  if (!Array.isArray(haystack) || needle.length > haystack.length) {
    return -1
  }
  for (let i = 0; i <= haystack.length - needle.length; i += 1) {
    let matched = true
    for (let j = 0; j < needle.length; j += 1) {
      if (haystack[i + j] !== needle[j]) {
        matched = false
        break
      }
    }
    if (matched) {
      return i
    }
  }
  return -1
}

/**
 * 判断 needle 是否为 haystack 的连续子序列（用于识别 SSE 增量回放已知历史行）。
 * @param {string[]} haystack
 * @param {string[]} needle
 * @returns {boolean}
 */
function isContiguousSubsequence(haystack, needle) {
  return indexOfContiguousSubsequence(haystack, needle) >= 0
}

/**
 * 合并 relay /status 或 SSE 推送的日志行到面板状态。
 * 快照保留服务端完整日志以便增量同步；展示用 logs 会剥离心跳探测行。
 * 清理日志后（suppressedAfterClear）：忽略历史全量/增量回放，仅展示快照真正增长后的新尾部。
 * @param {{ logs: string[], snapshot: string[], suppressedAfterClear: boolean, awaitingFirstStatusAfterStart?: boolean }} state
 * @param {unknown[]} lines
 * @returns {{ logs: string[], snapshot: string[], suppressedAfterClear: boolean, awaitingFirstStatusAfterStart: boolean, heartbeatLines: string[] }}
 */
export function mergeRelayToTraeLogLines(state, lines) {
  let logs = [...state.logs]
  let snapshot = [...state.snapshot]
  let suppressedAfterClear = state.suppressedAfterClear
  let awaitingFirstStatusAfterStart = Boolean(state.awaitingFirstStatusAfterStart)
  /** @type {string[]} */
  let heartbeatLines = []

  const takeStartupAndHeartbeat = (chunk) => {
    const { startupLines, heartbeatLines: hb } = partitionRelayToTraeLogLines(chunk)
    if (hb.length) {
      heartbeatLines = [...heartbeatLines, ...hb]
    }
    return startupLines
  }

  if (!Array.isArray(lines) || lines.length === 0) {
    // 空推送不得解除抑制，否则随后 /status 全量会再次灌入历史日志
    return { logs, snapshot, suppressedAfterClear, awaitingFirstStatusAfterStart, heartbeatLines }
  }

  const normalized = lines.map((line) => String(line))
  if (suppressedAfterClear) {
    if (awaitingFirstStatusAfterStart) {
      snapshot = [...normalized]
      logs = takeStartupAndHeartbeat(normalized)
      suppressedAfterClear = false
      awaitingFirstStatusAfterStart = false
      return { logs, snapshot, suppressedAfterClear, awaitingFirstStatusAfterStart, heartbeatLines }
    }
    if (snapshot.length === 0) {
      // 清理时尚未建立快照：吸收服务端缓冲但不回填展示
      snapshot = [...normalized]
      return { logs, snapshot, suppressedAfterClear: true, awaitingFirstStatusAfterStart: false, heartbeatLines }
    }
    if (hasRelayLogPrefix(normalized, snapshot)) {
      const newTail = normalized.slice(snapshot.length)
      snapshot = [...normalized]
      if (newTail.length > 0) {
        logs = [...logs, ...takeStartupAndHeartbeat(newTail)]
      }
      // 保持抑制：历史部分不再展示，仅追加清理后的新行
      return { logs, snapshot, suppressedAfterClear: true, awaitingFirstStatusAfterStart: false, heartbeatLines }
    }
    // 全量缓冲包含当前快照（SSE 先推了中段、随后 /status cursor=0 回放全量）：
    // 只吸收快照之后的新尾部，禁止把快照前的历史前缀回填到展示区。
    const containedAt = indexOfContiguousSubsequence(normalized, snapshot)
    if (containedAt >= 0) {
      const newTail = normalized.slice(containedAt + snapshot.length)
      snapshot = [...normalized]
      if (newTail.length > 0) {
        logs = [...logs, ...takeStartupAndHeartbeat(newTail)]
      }
      return { logs, snapshot, suppressedAfterClear: true, awaitingFirstStatusAfterStart: false, heartbeatLines }
    }
    // 更短的同源全量、或 SSE 回放的已知片段：忽略展示
    if (hasRelayLogPrefix(snapshot, normalized) || isContiguousSubsequence(snapshot, normalized)) {
      return { logs, snapshot, suppressedAfterClear: true, awaitingFirstStatusAfterStart: false, heartbeatLines }
    }
    // 与快照无关的增量（清理后新产生的行）：追加展示，仍保持抑制以免后续全量回填
    snapshot = [...snapshot, ...normalized]
    logs = [...logs, ...takeStartupAndHeartbeat(normalized)]
    return { logs, snapshot, suppressedAfterClear: true, awaitingFirstStatusAfterStart: false, heartbeatLines }
  }

  let toAppend = normalized
  if (snapshot.length === 0) {
    snapshot = [...normalized]
  } else if (
    normalized.length >= snapshot.length &&
    hasRelayLogPrefix(normalized, snapshot)
  ) {
    toAppend = normalized.slice(snapshot.length)
    snapshot = [...normalized]
  } else {
    // SSE 增量先到（无 relay 前缀）后 /status 全量到达：快照是全量的中段子序列。
    // 必须以服务端全量为准重建展示，否则会把 bootstrap 段再拼一遍。
    const containedAt = indexOfContiguousSubsequence(normalized, snapshot)
    if (containedAt >= 0) {
      snapshot = [...normalized]
      logs = takeStartupAndHeartbeat(normalized)
      return { logs, snapshot, suppressedAfterClear, awaitingFirstStatusAfterStart, heartbeatLines }
    }
    snapshot = [...snapshot, ...normalized]
  }
  if (toAppend.length > 0) {
    logs = [...logs, ...takeStartupAndHeartbeat(toAppend)]
  }
  return { logs, snapshot, suppressedAfterClear, awaitingFirstStatusAfterStart, heartbeatLines }
}

export function isRouteRelayToTraeQuery(query) {
  const raw = query?.relayToTrae
  if (raw == null) return false
  const v = Array.isArray(raw) ? String(raw[0] || '') : String(raw)
  return v.trim().toLowerCase() === 'true'
}

export function parseRelayToTraeEnv(envItems) {
  const out = {}
  for (const item of envItems) {
    const key = String(item?.key || '').trim()
    if (!key) {
      continue
    }
    out[key] = String(item?.value ?? '')
  }
  return out
}

/**
 * @param {object} query - route query object
 * @param {object} [defaults] - optional overrides for env defaults
 * @param {string} [defaults.taskApiOrigin]
 * @param {string} [defaults.businessApiOrigin]
 */
function defaultRelayBusinessApiOrigin() {
  if (typeof import.meta.env?.VITE_DEFAULT_RELAY_BUSINESS_API_ORIGIN === 'string') {
    const fromVite = import.meta.env.VITE_DEFAULT_RELAY_BUSINESS_API_ORIGIN.trim()
    if (fromVite) return fromVite
  }
  try {
    const host =
      typeof window !== 'undefined' ? String(window.location?.hostname || '').trim() : ''
    if (host && host !== 'localhost' && host !== '127.0.0.1') {
      return `http://${host}:8765`
    }
  } catch {
    /* ignore */
  }
  return 'http://127.0.0.1:8765'
}

export function buildDefaultRelayToTraeEnvItems(query, defaults) {
  const taskOrigin =
    defaults?.taskApiOrigin != null
      ? String(defaults.taskApiOrigin)
      : (typeof import.meta.env?.VITE_DEFAULT_MOCK_TASK_API_ENDPOINT === 'string'
          ? import.meta.env.VITE_DEFAULT_MOCK_TASK_API_ENDPOINT.trim()
          : '') || DEFAULT_TASK_API_ENDPOINT_ORIGIN
  const bizOrigin =
    defaults?.businessApiOrigin != null
      ? String(defaults.businessApiOrigin)
      : defaultRelayBusinessApiOrigin()
  return RELAY_TO_TRAE_ENV_KEYS.map((key) => {
    if (key === 'TASK_API_ENDPOINT_ORIGIN') {
      return { key, value: taskOrigin }
    }
    if (key === 'BUSINESS_API_ENDPOINT_ORIGIN') {
      return { key, value: bizOrigin }
    }
    if (key === 'DEBUG_AGENT') {
      return { key, value: 'True' }
    }
    return { key, value: TASK2APP_ACCESS_TOKEN_PLACEHOLDER }
  })
}

export function buildRelayToTraeStartEnvPayload(envItems, accessTokenPlaceholder) {
  const envPayload = parseRelayToTraeEnv(envItems)
  envPayload.ACCESS_TOKEN = accessTokenPlaceholder
  for (const key of ['TASK_API_ENDPOINT_ORIGIN', 'BUSINESS_API_ENDPOINT_ORIGIN']) {
    if (!String(envPayload[key] || '').trim()) {
      return { ok: false, missingKey: key }
    }
  }
  return { ok: true, env: envPayload }
}

export function buildRelayToTraeTokenInitPayload(envItems, accessTokenPlaceholder) {
  const built = buildRelayToTraeStartEnvPayload(envItems, accessTokenPlaceholder)
  if (!built.ok) {
    return built
  }
  return {
    ok: true,
    payload: {
      env: built.env,
    },
  }
}

/**
 * relay /status 的 ui_url 常为 127.0.0.1；对外链接应对齐 ServiceConfig 中的
 * BUSINESS_API_ENDPOINT_ORIGIN 或实例公网 IP（与 effectiveContainerVscodeUrl 一致）。
 *
 * @param {string} rawUiUrl
 * @param {{ businessApiOrigin?: string, taskApiOrigin?: string, publicIp?: string }} [options]
 * @returns {string}
 */
export function resolveRelayToTraePublicUiUrl(rawUiUrl, options = {}) {
  const raw = String(rawUiUrl || '').trim()
  if (!raw) {
    return ''
  }
  if (!isLoopbackHttpUrl(raw)) {
    return raw
  }

  const tryRewriteFromOrigin = (origin) => {
    const text = String(origin || '').trim()
    if (!text || isLoopbackHttpUrl(text)) {
      return ''
    }
    try {
      const withScheme =
        text.startsWith('http://') || text.startsWith('https://') ? text : `http://${text}`
      const originUrl = new URL(withScheme)
      const host = originUrl.hostname
      if (!host) {
        return ''
      }
      const rawWithScheme =
        raw.startsWith('http://') || raw.startsWith('https://') ? raw : `http://${raw}`
      const u = new URL(rawWithScheme)
      u.protocol = originUrl.protocol
      u.hostname = host
      // Domain-mode origins omit port — drop published loopback port
      if (!originUrl.port) {
        u.port = ''
      }
      const rewritten = u.toString()
      if (rewritten && !isLoopbackHttpUrl(rewritten)) {
        return rewritten
      }
    } catch {
      // ignore invalid origin
    }
    return ''
  }

  const fromBiz = tryRewriteFromOrigin(options.businessApiOrigin)
  if (fromBiz) {
    return fromBiz
  }

  const fromPublicIp = String(options.publicIp || '').trim()
  if (fromPublicIp) {
    // Full Origin (${scheme}://${domain}): adopt scheme/host; bare IP/host keeps published port
    if (fromPublicIp.includes('://')) {
      const fromOrigin = tryRewriteFromOrigin(fromPublicIp)
      if (fromOrigin) {
        return fromOrigin
      }
    } else {
      const rewritten = replaceHttpUrlHostname(raw, fromPublicIp)
      if (rewritten && !isLoopbackHttpUrl(rewritten)) {
        return rewritten
      }
    }
  }

  const fromTask = tryRewriteFromOrigin(options.taskApiOrigin)
  if (fromTask) {
    return fromTask
  }

  return raw
}

/**
 * 决定 relay「打开控制台」最终 href。
 * - selected_image：必须用容器 register-reachability 的 server_url / container_page_url，
 *   不得用侧车臆造的 /ui/{token}，也不得回退到 SaaS task-detail。
 * - 其它（host run.sh）：沿用侧车 ui_url（可改写 loopback）。
 *
 * @param {{
 *   mode?: string,
 *   relayUiUrl?: string,
 *   serverUrl?: string,
 *   containerPageUrl?: string,
 *   businessApiOrigin?: string,
 *   taskApiOrigin?: string,
 *   publicIp?: string,
 * }} input
 * @returns {string}
 */
export function resolveRelayConsoleOpenUrl(input = {}) {
  const mode = String(input.mode || '').trim()
  const rewriteOpts = {
    businessApiOrigin: input.businessApiOrigin,
    taskApiOrigin: input.taskApiOrigin,
    publicIp: input.publicIp,
  }

  if (mode === 'selected_image') {
    const registered = String(input.containerPageUrl || input.serverUrl || '').trim()
    if (!registered) {
      return ''
    }
    // 防御误把任务详情 SPA 当成容器地址
    if (/\/task-detail\//i.test(registered) || /:4000\b/.test(registered)) {
      return ''
    }
    return resolveRelayToTraePublicUiUrl(registered, rewriteOpts) || registered
  }

  return resolveRelayToTraePublicUiUrl(String(input.relayUiUrl || '').trim(), rewriteOpts)
}
