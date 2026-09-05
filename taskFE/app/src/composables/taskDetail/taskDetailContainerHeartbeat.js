import { ref } from 'vue'
import { appendHeartbeatLogRing } from '../../utils/containerHeartbeatLogRing.js'
import { snapshotHasRealWritableLayer } from '../../utils/layerZtreeBootstrapAnchor.js'
import { probeContainerReachability as _probeContainerReachability } from './taskDetailContainerFns.js'

export const CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS = 8000
export const CONTAINER_UNREACHABLE_PROBE_BACKOFF_MAX_MS = 120000

/**
 * Factory: container heartbeat status, unreachable probe, and transport helpers.
 */
export function createContainerHeartbeatState(deps) {
  /** DB 中已有 server_url，但 SaaS 转发容器 HTTP 连续失败（如容器停机、网络不可达） */
  const containerHttpUnreachable = ref(false)
  /** 容器 access token 失效时暂停自动层图拉取，避免持续刷 502。 */
  const containerLayerGraphAuthInvalid = ref(false)
  /** 容器不可达时用 setTimeout 链探测（指数退避），避免固定间隔刷爆 upstream */
  let containerUnreachableProbeTimer = null
  let containerUnreachableProbeBackoffMs = CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS

  function clearContainerUnreachableProbeTimer() {
    if (containerUnreachableProbeTimer != null) {
      clearTimeout(containerUnreachableProbeTimer)
      containerUnreachableProbeTimer = null
    }
  }

  function scheduleContainerUnreachableProbe(delayMs) {
    clearContainerUnreachableProbeTimer()
    const jitter = Math.round(delayMs * 0.12 * (Math.random() * 2 - 1))
    const delay = Math.max(500, delayMs + jitter)
    containerUnreachableProbeTimer = setTimeout(() => {
      containerUnreachableProbeTimer = null
      void probeContainerReachability()
    }, delay)
  }

  /** 容器连接心跳状态 */
  const containerHeartbeatStatus = ref('idle') // idle, connecting, connected, disconnected
  /** relayToTrae 停止时暂停容器心跳 SSE 事件处理，避免状态被后续事件覆盖 */
  const containerHeartbeatPaused = ref(false)
  /** server-runtime-status 判定为 released/stopped 等（冷打开已释放任务） */
  const serverRuntimeNotServing = ref(false)
  const containerHeartbeatLastSuccess = ref(null)
  const containerHeartbeatAttempts = ref(0)
  const containerHeartbeatError = ref('')
  /** 最近一次 SSE container_heartbeat 的 seq/ack（供状态面板展示） */
  const containerHeartbeatSeqInfo = ref(emptyContainerHeartbeatSeqInfo())
  /** 从启动/容器日志剥离的心跳探测行（环形缓冲，展示在「容器连接状态」） */
  const containerHeartbeatLogLines = ref([])
  let containerHeartbeatTimer = null

  function emptyContainerHeartbeatSeqInfo() {
    return {
      containerSeq: null,
      containerAck: null,
      saasSeq: null,
      saasAck: null,
      uplinkOk: null,
      downlinkOk: null,
      probeOk: null,
      bidirectionalOk: null,
    }
  }

  const appendContainerHeartbeatLogLines = (lines) => {
    containerHeartbeatLogLines.value = appendHeartbeatLogRing(
      containerHeartbeatLogLines.value,
      lines,
    )
  }

  const parseHeartbeatSeqField = (value) => {
    if (value === null || value === undefined || value === '') {
      return null
    }
    const n = Number(value)
    return Number.isFinite(n) && n >= 0 ? Math.floor(n) : null
  }

  const applyContainerHeartbeatSeqFromSse = (statusData) => {
    if (!statusData || typeof statusData !== 'object') {
      return
    }
    const tri = (v) => (v === true ? true : v === false ? false : null)
    containerHeartbeatSeqInfo.value = {
      containerSeq: parseHeartbeatSeqField(statusData.container_seq),
      containerAck: parseHeartbeatSeqField(statusData.container_ack),
      saasSeq: parseHeartbeatSeqField(statusData.saas_seq),
      saasAck: parseHeartbeatSeqField(statusData.saas_ack),
      uplinkOk: tri(statusData.uplink_ok),
      downlinkOk: tri(statusData.downlink_ok),
      probeOk: tri(statusData.probe_ok),
      bidirectionalOk: tri(statusData.bidirectional_ok),
    }
  }

  const markContainerTransportOk = () => {
    containerHttpUnreachable.value = false
    containerLayerGraphAuthInvalid.value = false
    clearContainerUnreachableProbeTimer()
    containerUnreachableProbeBackoffMs = CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS
  }

  /** 启动容器心跳：依赖 SSE 上由容器 HTTP 上报触发的 container_heartbeat 事件（服务端不伪造） */
  const startContainerHeartbeat = () => {
    const taskId = deps.effectiveTaskId.value
    const tenantId = deps.effectiveTenantId.value
    const workspaceId = deps.effectiveWorkspaceId.value

    if (containerHeartbeatTimer != null) {
      clearInterval(containerHeartbeatTimer)
      containerHeartbeatTimer = null
    }

    if (!taskId || !tenantId || !workspaceId) {
      return
    }

    deps.establishSSEConnection(taskId)
  }

  /** 停止容器心跳检测 */
  const stopContainerHeartbeat = () => {
    if (containerHeartbeatTimer != null) {
      clearInterval(containerHeartbeatTimer)
      containerHeartbeatTimer = null
    }
    containerHeartbeatStatus.value = 'idle'
    containerHeartbeatAttempts.value = 0
    containerHeartbeatLastSuccess.value = null
    containerHeartbeatError.value = ''
    containerHeartbeatSeqInfo.value = emptyContainerHeartbeatSeqInfo()
    containerHeartbeatLogLines.value = []
  }

  const pauseContainerHeartbeatForRelayStop = () => {
    containerHeartbeatPaused.value = true
    deps.containerEndpointRegistered.value = false
    deps.containerPageUrl.value = ''
    deps.containerVscodeUrl.value = ''
    deps.containerPageLinkPendingReveal.value = true
    deps.layerGraphSnapshot.value = null
    deps.resetLayerGraphFetchBackoff()
    markContainerTransportOk()
    stopContainerHeartbeat()
  }

  const resumeContainerHeartbeatForRelayStart = () => {
    containerHeartbeatPaused.value = false
    serverRuntimeNotServing.value = false
    deps.containerPageUrl.value = ''
    deps.containerVscodeUrl.value = ''
    deps.containerEndpointRegistered.value = false
    deps.serverUrl.value = ''
  }

  /** 云 runtime 确认已在服务：仅清 notServing 标志，不抹 URL/endpoint */
  const markServerRuntimeServing = () => {
    serverRuntimeNotServing.value = false
  }

  /** 冷打开 / runtime 查询确认机器已非服务时进入与停机对称的 UI 态 */
  const enterServerNotServingUiState = (opts = {}) => {
    const fromRuntime = opts.fromRuntime === true
    if (fromRuntime) {
      serverRuntimeNotServing.value = true
    }
    if (deps.isServerRunning.value || deps.isServerStarting.value) {
      return
    }
    pauseContainerHeartbeatForRelayStop()
  }

  const isContainerAccessTokenInvalidDetail = (detailMaybe) => {
    const d = typeof detailMaybe === 'string' ? detailMaybe : ''
    if (!d) {
      return false
    }
    return /invalid or missing access token/i.test(d) || /access token/i.test(d)
  }

  /** 将层级图「发送给 AI」接口返回的技术性错误转为用户可读文案（不暴露本机路径） */
  const formatLayerGraphCommandErrorForUser = (httpStatus, rawDetail) => {
    const d = typeof rawDetail === 'string' ? rawDetail : ''
    if (isContainerAccessTokenInvalidDetail(d)) {
      return '容器访问凭据无效或已过期，请重新授权后重试。'
    }
    if (/Config missing/i.test(d) && /service_config\.ya?ml/i.test(d)) {
      return '智能体执行服务缺少必要配置（service_config），请检查 Trae/运行环境配置后重试。'
    }
    if (/Config missing/i.test(d)) {
      return '智能体执行环境配置不完整，请检查相关服务配置后重试。'
    }
    if (httpStatus === 502 || /容器接口返回错误/.test(d)) {
      if (
        /127\.0\.0\.1|localhost|upstream|\/api\/jobs/i.test(d) ||
        /HTTP 400|HTTP 500|HTTP 502/.test(d)
      ) {
        return '无法连接或调用智能体执行服务，请确认本地 Trae/容器服务已启动且配置正确后重试。'
      }
      return '智能体服务暂时不可用，请稍后重试。'
    }
    if (httpStatus === 503) {
      return '服务繁忙，请稍后重试。'
    }
    if (httpStatus === 504) {
      return '执行服务响应超时，请稍后重试。'
    }
    if (httpStatus === 403) {
      if (/forbidden scope/i.test(d) || /容器配置不存在或尚未就绪/.test(d)) {
        return '当前评论的容器配置不可用（可能尚未启动或已释放）。请等待容器就绪后刷新重试。'
      }
      const generic =
        !d.trim() ||
        /^HTTP\s*403$/i.test(d.trim()) ||
        /403\s*Forbidden/i.test(d)
      if (generic) {
        return '没有权限执行该操作，或登录态/容器授权已失效。请刷新页面后重试。'
      }
    }
    let display = d.replace(/\/Users\/[^\s"'<>]+/g, '…')
    display = display.replace(/\/home\/[^\s"'<>]+/g, '…')
    if (display.length > 240) {
      display = `${display.slice(0, 240)}…`
    }
    if (!display.trim()) {
      return httpStatus ? `请求失败（HTTP ${httpStatus}）` : '请求失败，请稍后重试'
    }
    return display
  }

  const markContainerTransportUnreachableIfForwardingFailed = (httpStatus, detailMaybe) => {
    if (!deps.containerEndpointRegistered.value) {
      return
    }
    const d = typeof detailMaybe === 'string' ? detailMaybe : ''
    if (isContainerAccessTokenInvalidDetail(d)) {
      containerLayerGraphAuthInvalid.value = true
      return
    }
    if (
      httpStatus === 502 ||
      httpStatus === 503 ||
      d.includes('转发容器失败') ||
      d.includes('容器内在线服务未启动')
    ) {
      containerHttpUnreachable.value = true
      if (containerUnreachableProbeTimer === null) {
        containerUnreachableProbeBackoffMs = CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS
        void probeContainerReachability()
      }
    }
  }

  const probeContainerReachability = () => _probeContainerReachability({
    containerEndpointRegistered: deps.containerEndpointRegistered,
    containerHttpUnreachable,
    effectiveTenantId: deps.effectiveTenantId,
    effectiveWorkspaceId: deps.effectiveWorkspaceId,
    effectiveTaskId: deps.effectiveTaskId,
    displayComments: deps.displayComments,
    activeContainerAgentId: deps.activeContainerAgentId,
    ensureServerRuntimeAllowsContainerLayerGraph: deps.ensureServerRuntimeAllowsContainerLayerGraph,
    refreshLayerGraphFromServer: deps.refreshLayerGraphFromServer,
    applyLayerGraphFromPayload: deps.applyLayerGraphFromPayload,
    resetLayerGraphFetchBackoff: deps.resetLayerGraphFetchBackoff,
    markContainerTransportOk,
    clearContainerUnreachableProbeTimer,
    scheduleContainerUnreachableProbe,
    get containerUnreachableProbeBackoffMs() { return containerUnreachableProbeBackoffMs },
    set containerUnreachableProbeBackoffMs(v) { containerUnreachableProbeBackoffMs = v },
    CONTAINER_UNREACHABLE_PROBE_BACKOFF_MAX_MS,
    CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS,
  })

  return {
    containerHttpUnreachable,
    containerLayerGraphAuthInvalid,
    containerHeartbeatStatus,
    containerHeartbeatPaused,
    serverRuntimeNotServing,
    containerHeartbeatLastSuccess,
    containerHeartbeatAttempts,
    containerHeartbeatError,
    containerHeartbeatSeqInfo,
    containerHeartbeatLogLines,
    appendContainerHeartbeatLogLines,
    applyContainerHeartbeatSeqFromSse,
    markContainerTransportOk,
    startContainerHeartbeat,
    stopContainerHeartbeat,
    pauseContainerHeartbeatForRelayStop,
    resumeContainerHeartbeatForRelayStart,
    markServerRuntimeServing,
    enterServerNotServingUiState,
    isContainerAccessTokenInvalidDetail,
    formatLayerGraphCommandErrorForUser,
    markContainerTransportUnreachableIfForwardingFailed,
    probeContainerReachability,
    scheduleContainerUnreachableProbe,
    clearContainerUnreachableProbeTimer,
    get containerUnreachableProbeBackoffMs() { return containerUnreachableProbeBackoffMs },
    set containerUnreachableProbeBackoffMs(v) { containerUnreachableProbeBackoffMs = v },
    get containerHeartbeatTimer() { return containerHeartbeatTimer },
    set containerHeartbeatTimer(v) { containerHeartbeatTimer = v },
  }
}

export const LAYER_GRAPH_FETCH_BACKOFF_BASE_MS = 2000
export const LAYER_GRAPH_FETCH_BACKOFF_MAX_MS = 120000

export function createLayerGraphFetchBackoffState() {
  let layerGraphFetchBackoffUntil = 0
  let layerGraphFetchConsecutiveFailures = 0

  function resetLayerGraphFetchBackoff() {
    layerGraphFetchBackoffUntil = 0
    layerGraphFetchConsecutiveFailures = 0
  }

  function registerLayerGraphFetchFailure() {
    layerGraphFetchConsecutiveFailures = Math.min(layerGraphFetchConsecutiveFailures + 1, 14)
    const raw =
      LAYER_GRAPH_FETCH_BACKOFF_BASE_MS * 2 ** Math.max(0, layerGraphFetchConsecutiveFailures - 1)
    const capped = Math.min(LAYER_GRAPH_FETCH_BACKOFF_MAX_MS, raw)
    const jitter = Math.round(capped * 0.15 * (Math.random() * 2 - 1))
    layerGraphFetchBackoffUntil = Date.now() + Math.max(LAYER_GRAPH_FETCH_BACKOFF_BASE_MS, capped + jitter)
  }

  return {
    resetLayerGraphFetchBackoff,
    registerLayerGraphFetchFailure,
    get layerGraphFetchBackoffUntil() { return layerGraphFetchBackoffUntil },
    set layerGraphFetchBackoffUntil(v) { layerGraphFetchBackoffUntil = v },
  }
}

/**
 * 层级树是否已有真实可写层（引导 empty/pending 锚点不算就绪，心跳须继续拉 clone-log）。
 */
export function layerGraphSnapshotHasContent(snap) {
  if (!snap || typeof snap !== 'object') {
    return false
  }
  return snapshotHasRealWritableLayer(snap.layers)
}

/**
 * 与容器 /api/jobs 一致：任一并行任务为 queued|pending|running 时，主页面须继续向容器拉取快照。
 */
export function layerGraphSnapshotHasActiveJob(snap) {
  if (!snap || typeof snap !== 'object') {
    return false
  }
  const jobs = Array.isArray(snap.jobs) ? snap.jobs : []
  return jobs.some((j) => {
    const s = String(j?.status || '')
      .trim()
      .toLowerCase()
    return s === 'queued' || s === 'pending' || s === 'running'
  })
}

export const SERVER_RUNTIME_LAYER_GRAPH_GATE_MS = 2500

export function createServerRuntimeLayerGraphGateState(deps) {
  let serverRuntimeLayerGraphGateCheckedAt = 0
  let serverRuntimeLayerGraphGateAllowed = true

  function resetServerRuntimeLayerGraphGateCache() {
    serverRuntimeLayerGraphGateCheckedAt = 0
    serverRuntimeLayerGraphGateAllowed = true
  }

  function shouldBypassServerRuntimeLayerGraphGate() {
    return (
      deps.containerEndpointRegistered.value &&
      deps.containerHeartbeatStatus.value === 'connected' &&
      !deps.containerHttpUnreachable.value
    )
  }

  function serverRuntimeStatusAllowsContainerLayerGraphFetch(data) {
    if (!data || typeof data !== 'object' || data.status !== 'success') {
      return true
    }
    if (data.mock === true) {
      return true
    }
    const rs = String(data.runtime_status ?? '')
      .trim()
      .toLowerCase()
    if (!rs) {
      return true
    }
    if (rs === 'running') {
      return true
    }
    const transitional = new Set(['initializing', 'starting', 'pending'])
    if (transitional.has(rs)) {
      return false
    }
    const notServing = new Set(['released', 'stopped', 'stopping', 'terminated', 'shutting-down', 'shuttingdown'])
    if (notServing.has(rs)) {
      return false
    }
    return true
  }

  return {
    resetServerRuntimeLayerGraphGateCache,
    shouldBypassServerRuntimeLayerGraphGate,
    serverRuntimeStatusAllowsContainerLayerGraphFetch,
    get serverRuntimeLayerGraphGateCheckedAt() { return serverRuntimeLayerGraphGateCheckedAt },
    set serverRuntimeLayerGraphGateCheckedAt(v) { serverRuntimeLayerGraphGateCheckedAt = v },
    get serverRuntimeLayerGraphGateAllowed() { return serverRuntimeLayerGraphGateAllowed },
    set serverRuntimeLayerGraphGateAllowed(v) { serverRuntimeLayerGraphGateAllowed = v },
  }
}

export function createLayerGraphHeartbeatRefresh(deps) {
  let lastLayerGraphHeartbeatRefreshAt = 0
  const LAYER_GRAPH_REFRESH_ON_HEARTBEAT_MIN_MS = 4000

  function maybeRefreshLayerGraphOnContainerHeartbeatOk() {
    if (!deps.containerEndpointRegistered.value) return
    if (deps.containerHttpUnreachable.value) return
    if (deps.containerLayerGraphAuthInvalid.value) return
    if (deps.containerPageLinkPendingReveal.value) return
    const snap = deps.layerGraphSnapshot.value
    const need =
      snap === null ||
      !layerGraphSnapshotHasContent(snap) ||
      layerGraphSnapshotHasActiveJob(snap)
    if (!need) return
    const now = Date.now()
    if (now - lastLayerGraphHeartbeatRefreshAt < LAYER_GRAPH_REFRESH_ON_HEARTBEAT_MIN_MS) return
    lastLayerGraphHeartbeatRefreshAt = now
    void deps.refreshLayerGraphFromServer()
    if (typeof deps.fetchContainerBootstrapCloneLog === 'function') {
      void deps.fetchContainerBootstrapCloneLog()
    }
  }

  return { maybeRefreshLayerGraphOnContainerHeartbeatOk }
}
