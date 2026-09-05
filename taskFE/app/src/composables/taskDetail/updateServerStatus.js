/**
 * Pure function: handles all SSE event types for TaskDetail server status.
 * All state dependencies are passed explicitly via the `deps` object.
 */
import { isAsyncStartVmSubmissionAck } from '../../utils/startVmHttpResult.js'
import { applyContainerGitCloneProgress } from './applyContainerGitCloneProgress.js'
import { enrichStartupStatusUpdate } from '../../utils/serverStartupErrorDisplay.js'
import { resolveLifecycleFlagsFromRuntimeStatus } from '../../utils/serverLifecycleFromRuntime.js'
import { isCloudServerStopSuccessMessage } from '../../utils/stopVmSseMessage.js'
import { pushServerStatusLogLines } from '../../utils/serverStatusLogLines.js'
import {
  collectServerStartupLogMessages,
  resolveServerStartupBindingCommentId,
} from '../../utils/bindingServerStartupLogs.js'
import { latestServerStartupStatusForBinding } from './perContainerHeartbeatBus.js'
import { commentIdFromStatusEvent, runtimeStatusFromStatusEvent } from './runtimeStatusRefreshPolicy.js'
import { resolveContainerUiContextCommentId } from './resolveContainerUiContextCommentId.js'
import { mergeAgentStepsByNumber } from './fetchJobExecutionLogBySteps.js'
import { handleContainerBootstrapSse } from './containerBootstrapSse.js'
import { handleContainerLayerGraphSse } from './containerLayerGraphSse.js'
import {
  applyNormalizedLayerChangesToStore,
  resolveLayerChangesOwnerCommentId,
} from './commentLayerPanelApply.js'

/** 与 layerLiveOutputDisplay 展示上限对齐，避免 map 内字符串无限增长 */
export const LAYER_JOB_LIVE_OUTPUT_MAX_CHARS = 100000

function appendCappedLiveOutput(prev, chunk) {
  const next = `${prev || ''}${chunk || ''}`
  if (next.length <= LAYER_JOB_LIVE_OUTPUT_MAX_CHARS) return next
  return next.slice(-LAYER_JOB_LIVE_OUTPUT_MAX_CHARS)
}

/** job-stream 终态：gateway 发 done，onlineServiceJS events 发 completed/failed/interrupted */
const CONTAINER_JOB_STREAM_TERMINAL_PHASES = new Set([
  'done',
  'error',
  'completed',
  'failed',
  'interrupted',
])

/**
 * 从 job-stream phase / job_status 解析可写层展示用终态；error（轮询失败）不臆造 job.status。
 * @param {string} phase
 * @param {object} statusData
 * @returns {string} completed|failed|interrupted|'' 
 */
export function resolveContainerJobStreamTerminalStatus(phase, statusData) {
  const p = String(phase || '').trim().toLowerCase()
  if (p === 'completed' || p === 'failed' || p === 'interrupted') return p
  if (p === 'done') {
    const js = String(statusData?.job_status || '').trim().toLowerCase()
    if (js === 'completed' || js === 'failed' || js === 'interrupted') return js
    return 'completed'
  }
  return ''
}

/**
 * 乐观更新层图 jobs/layers，避免 GET job 超时导致 zTree 长期停在 running。
 * @param {{ value: object|null }} snapshotRef
 * @param {string} jobId
 * @param {string} status
 */
export function patchLayerGraphJobStatus(snapshotRef, jobId, status) {
  const jid = String(jobId || '').trim()
  const st = String(status || '').trim().toLowerCase()
  if (!jid || !st || !snapshotRef) return
  const snap = snapshotRef.value
  if (!snap || typeof snap !== 'object') return
  const prevJobs = Array.isArray(snap.jobs) ? snap.jobs : []
  let layerId = ''
  const jobs = prevJobs.map((j) => {
    if (!j || String(j.id) !== jid) return j
    layerId = j.layer_id != null ? String(j.layer_id) : ''
    return { ...j, status: st }
  })
  let layers = Array.isArray(snap.layers) ? snap.layers : []
  if (layerId) {
    const mind = st === 'running' || st === 'pending' ? 'running' : 'idle_done'
    layers = layers.map((L) => {
      if (!L || String(L.layer_id) !== layerId) return L
      return { ...L, job_status: st, mind_state: mind, job_id: L.job_id || jid }
    })
  }
  snapshotRef.value = { ...snap, jobs, layers }
}

/**
 * 把 job-stream phase=step 事件里的 step_number/delivery_summary 即时合并进
 * layerJobExecutionPayload.steps。persist 消费者是异步的，GET 可能读到缺最新一步
 * 的 DB；SSE 先并入本地 steps 后，迟到 GET 的空页/缺页经 mergeAgentStepsByNumber
 * 按 step_number 合并也不会把已完成的步骤闪掉（OPT-20260816-042）。
 * @param {{value: object|null}|null} payloadRef
 * @param {string} jobId
 * @param {{step_number?: unknown, delivery_summary?: unknown, state?: unknown}} statusData
 */
export function mergeJobStreamStepIntoExecutionPayload(payloadRef, jobId, statusData) {
  if (!payloadRef) return
  const sn = Number(statusData?.step_number)
  if (!Number.isFinite(sn) || sn <= 0) return
  const payload = payloadRef.value
  if (!payload || !payload.job) return // 尚未 hydrate 出 payload 时交给 GET 兜底
  if (payload.job.id != null && String(payload.job.id) !== String(jobId)) return
  const step = {
    step_number: Math.floor(sn),
    delivery_summary: typeof statusData.delivery_summary === 'string' ? statusData.delivery_summary : '',
    state: typeof statusData.state === 'string' ? statusData.state : '',
  }
  const prevSteps = payload.steps?.steps
  payloadRef.value = {
    ...payload,
    steps: {
      ...(payload.steps || {}),
      steps: mergeAgentStepsByNumber(Array.isArray(prevSteps) ? prevSteps : [], [step]),
    },
  }
}

export function updateServerStatus(statusData, deps) {
  const {
    // Refs — read/write .value
    serverConfigRef, activeAiInstructId, aiStreamBuffer, aiStreamBusy,
    activeContainerAgentId,
    containerEndpointRegistered, containerPageUrl, containerVscodeUrl,
    containerPageLinkPendingReveal, layerGraphSnapshot, layerChangesByLayerId,
    layerJobLiveOutputMap, layerJobExecutionPayload, isServerRunning, isServerStarting, serverUrl,
    serverStatus, statusMessage, statusTraceId, statusProgress, statusLogs,
    // Functions
    fetchTaskDetail, markContainerTransportOk,
    refreshLayerGraphFromServer,
    normalizeLayerChangesPayload,
    applyLiveLayerChangesToCurrentPayload, refreshZTreeExecutionLog,
    stopContainerHeartbeat, fetchContainerTaskUiContext,
    // Mutable plain vars (accessed as deps.xxx directly below)
  } = deps

  if (statusData.status === 'relay_to_trae_status') {
    serverConfigRef.value?.applyRelayToTraeStatusFromSse?.(statusData)
    return
  }
  if (statusData.status === 'ai_instruct_stream') {
    const iid = statusData.instruct_id != null ? String(statusData.instruct_id) : ''
    if (!iid) {
      return
    }
    if (activeAiInstructId.value && iid !== activeAiInstructId.value) {
      return
    }
    const phase = statusData.phase
    const msg = typeof statusData.message === 'string' ? statusData.message : ''
    if (phase === 'chunk') {
      aiStreamBuffer.value += msg
      return
    }
    if (phase === 'error') {
      aiStreamBuffer.value += msg
      aiStreamBusy.value = false
      activeAiInstructId.value = null
      void fetchTaskDetail()
      return
    }
    if (phase === 'done') {
      aiStreamBusy.value = false
      activeAiInstructId.value = null
      void fetchTaskDetail().then(() => {
        aiStreamBuffer.value = ''
      })
    }
    return
  }
  if (statusData.status === 'container_agent_stream') {
    // Reuse aiStream* display slot (TaskDetail Feed) while tracking agent id for filtering.
    const aid = statusData.agent_comment_id != null ? String(statusData.agent_comment_id) : ''
    if (!aid) {
      return
    }
    if (activeContainerAgentId.value && aid !== activeContainerAgentId.value) {
      return
    }
    if (!activeContainerAgentId.value) {
      activeContainerAgentId.value = aid
      aiStreamBuffer.value = ''
    }
    const phase = statusData.phase
    const msg = typeof statusData.message === 'string' ? statusData.message : ''
    if (phase === 'chunk') {
      aiStreamBusy.value = true
      aiStreamBuffer.value += msg
      return
    }
    if (phase === 'error') {
      aiStreamBuffer.value += msg
      aiStreamBusy.value = false
      activeContainerAgentId.value = null
      void fetchTaskDetail()
      return
    }
    if (phase === 'done') {
      aiStreamBusy.value = false
      activeContainerAgentId.value = null
      void fetchTaskDetail().then(() => {
        aiStreamBuffer.value = ''
      })
    }
    return
  }
  if (applyContainerGitCloneProgress(statusData, deps)) {
    return
  }
  if (handleContainerLayerGraphSse(statusData, deps)) {
    return
  }
  if (statusData.status === 'container_layer_changes') {
    markContainerTransportOk()
    const normalizedLayerChanges = normalizeLayerChangesPayload(statusData)
    if (!normalizedLayerChanges) {
      return
    }
    const store = deps.layerPanelStore
    const cid = resolveLayerChangesOwnerCommentId(store, statusData, normalizedLayerChanges.layer_id)
    if (cid && store) {
      applyNormalizedLayerChangesToStore(store, cid, normalizedLayerChanges)
    }
    const activeId = resolveContainerUiContextCommentId(deps)
    if (!cid || !activeId || cid === activeId || !store) {
      layerChangesByLayerId.value = {
        ...layerChangesByLayerId.value,
        [normalizedLayerChanges.layer_id]: normalizedLayerChanges,
      }
      applyLiveLayerChangesToCurrentPayload(normalizedLayerChanges)
    }
    return
  }
  if (statusData.status === 'container_job_stream') {
    markContainerTransportOk()
    const jid = statusData.job_id != null ? String(statusData.job_id) : ''
    if (!jid) return
    const phase = String(statusData.phase || '').trim().toLowerCase()
    const msg = typeof statusData.message === 'string' ? statusData.message : ''
    const evObj = statusData.event && typeof statusData.event === 'object' ? statusData.event : null
    const evLine = evObj ? `${JSON.stringify(evObj)}\n` : ''
    const ownerCid = deps.layerPanelStore?.commentIdOwningJob?.(jid) || commentIdFromStatusEvent(statusData)
    const liveMapRef = ownerCid && deps.layerPanelStore
      ? deps.layerPanelStore.refsFor(ownerCid).layerJobLiveOutputMap
      : layerJobLiveOutputMap
    if (phase === 'chunk' && msg) {
      const cur = liveMapRef.value[jid] || ''
      liveMapRef.value = {
        ...liveMapRef.value,
        [jid]: appendCappedLiveOutput(cur, evLine || msg),
      }
      return
    }
    if (phase === 'start') {
      const initText = evLine || (msg ? `${msg}\n` : '')
      liveMapRef.value = {
        ...liveMapRef.value,
        [jid]: appendCappedLiveOutput('', initText),
      }
      return
    }
    /** Agent 每完成一步推送 phase=step：立即拉取执行日志，避免任务结束才整批刷出 */
    if (phase === 'step') {
      if (msg) {
        const cur = liveMapRef.value[jid] || ''
        liveMapRef.value = {
          ...liveMapRef.value,
          [jid]: appendCappedLiveOutput(cur, `${msg}\n`),
        }
      }
      // OPT-20260816-042：SSE step 即时并入本地 steps，GET 只作 hydrate/对账，
      // 避免 persist 消费者滞后时 GET 缺页把最新一步闪掉。
      const stepPayloadRef = ownerCid && deps.layerPanelStore
        ? deps.layerPanelStore.refsFor(ownerCid).layerJobExecutionPayload
        : layerJobExecutionPayload
      mergeJobStreamStepIntoExecutionPayload(stepPayloadRef, jid, statusData)
      void refreshZTreeExecutionLog({ commentId: ownerCid })
      return
    }
    /**
     * 终态：gateway 在 GET /api/jobs/:id 成功时发 done；events 也会发 completed/failed/interrupted。
     * 旧容器 job JSON 含巨量 output 时 GET 易超时，仅靠 done 会长期停在 running——须同时认 events 终态。
     */
    if (CONTAINER_JOB_STREAM_TERMINAL_PHASES.has(phase)) {
      if (evLine || msg) {
        const cur = liveMapRef.value[jid] || ''
        liveMapRef.value = {
          ...liveMapRef.value,
          [jid]: appendCappedLiveOutput(cur, evLine || `${msg}\n`),
        }
      }
      const terminalStatus = resolveContainerJobStreamTerminalStatus(phase, statusData)
      if (terminalStatus) {
        if (ownerCid && deps.layerPanelStore) {
          patchLayerGraphJobStatus(deps.layerPanelStore.refsFor(ownerCid).layerGraphSnapshot, jid, terminalStatus)
        }
        patchLayerGraphJobStatus(layerGraphSnapshot, jid, terminalStatus)
      }
      void refreshZTreeExecutionLog({ commentId: ownerCid })
      void refreshLayerGraphFromServer(true, { commentId: ownerCid })
      return
    }
    // 未识别 phase（如 running）不落入下方启动状态分支
    return
  }
  if (handleContainerBootstrapSse(statusData, deps)) {
    return
  }
  if (statusData.status === 'container_task_ui_ready') {
    containerPageLinkPendingReveal.value = false
    markContainerTransportOk()
    const readyCid = commentIdFromStatusEvent(statusData)
    if (readyCid && typeof deps.fetchContainerTaskUiContext === 'function') {
      void deps.fetchContainerTaskUiContext(readyCid)
    }
    return
  }
  if (statusData.status === 'container_task_ui_context') {
    if (!statusData.container_endpoint_registered) {
      markContainerTransportOk()
    }
    const ctxCid = commentIdFromStatusEvent(statusData)
    const store = deps.layerPanelStore
    const endpoint = !!statusData.container_endpoint_registered
    const pageUrl = typeof statusData.container_page_url === 'string' ? statusData.container_page_url : ''
    const vscodeUrl = typeof statusData.container_vscode_url === 'string' ? statusData.container_vscode_url : ''
    if (ctxCid && store && typeof store.patch === 'function') {
      store.patch(ctxCid, {
        containerEndpointRegistered: endpoint,
        containerPageUrl: pageUrl,
        containerVscodeUrl: vscodeUrl,
        uiContextFetched: true,
      })
    }
    const activeId = resolveContainerUiContextCommentId(deps)
    if (!ctxCid || !activeId || ctxCid === activeId) {
      containerEndpointRegistered.value = endpoint
      containerPageUrl.value = pageUrl
      containerVscodeUrl.value = vscodeUrl
    }
    if (
      (containerEndpointRegistered.value || isServerRunning.value || isServerStarting.value) &&
      typeof deps.syncContainerHeartbeatAfterServingHint === 'function'
    ) {
      deps.syncContainerHeartbeatAfterServingHint()
    }
    const shouldPullLayerGraph = containerEndpointRegistered.value && !containerPageLinkPendingReveal.value
    if (shouldPullLayerGraph) {
      void refreshLayerGraphFromServer()
    }
    return
  }
  // 云 Describe 冷打开 / 轮询回填：不写启动日志，对齐「服务器启动状态」与运行态
  if (statusData.status === 'runtime_hydrate') {
    const runtimeStatus = statusData.runtime_status ?? statusData.runtimeStatus ?? ''
    if (deps.cloudRuntimeStatus) {
      deps.cloudRuntimeStatus.value = String(runtimeStatus ?? '')
    }
    const flags = resolveLifecycleFlagsFromRuntimeStatus(runtimeStatus)
    if (!flags) {
      return
    }
    if (flags.kind === 'running') {
      const wasRunning = isServerRunning.value === true
      isServerRunning.value = true
      isServerStarting.value = false
      if (typeof deps.markServerRuntimeServing === 'function') {
        deps.markServerRuntimeServing()
      } else if (deps.serverRuntimeNotServing) {
        deps.serverRuntimeNotServing.value = false
      }
      // 冷打开：runtime 确认服务后冲刷早到心跳，并离开 idle「等待连接」
      if (typeof deps.syncContainerHeartbeatAfterServingHint === 'function') {
        deps.syncContainerHeartbeatAfterServingHint()
      }
      if (!wasRunning) {
        void fetchContainerTaskUiContext()
      }
      return
    }
    if (flags.kind === 'starting') {
      if (!isServerRunning.value) {
        isServerStarting.value = true
        if (flags.serverStatusCode) {
          serverStatus.value = flags.serverStatusCode
        }
      }
      return
    }
    if (flags.kind === 'not_serving') {
      isServerRunning.value = false
      isServerStarting.value = false
      if (flags.serverStatusCode) {
        serverStatus.value = flags.serverStatusCode
      }
      if (deps.serverRuntimeNotServing) {
        deps.serverRuntimeNotServing.value = true
      }
      if (typeof deps.pauseContainerHeartbeatForRelayStop === 'function') {
        deps.pauseContainerHeartbeatForRelayStop()
      } else {
        stopContainerHeartbeat()
      }
      return
    }
    return
  }
  // 更新状态消息
  const enriched = enrichStartupStatusUpdate(statusData)
  statusMessage.value = enriched.message
  if (deps.statusEventCommentId) {
    deps.statusEventCommentId.value = commentIdFromStatusEvent(statusData)
  }
  if (deps.statusEventRuntimeStatus) {
    deps.statusEventRuntimeStatus.value = runtimeStatusFromStatusEvent(statusData)
  }
  serverStatus.value = enriched.status
  // 从 SSE statusData 提取 trace_id（由后端 publishSSEMessage 注入），
  // 透传至 statusTraceId → CommentsSection → CommentsPanel → data-traceId
  if (typeof statusData.trace_id === 'string' && statusData.trace_id.trim()) {
    statusTraceId.value = statusData.trace_id.trim()
  }

  // 更新进度
  statusProgress.value = statusData.progress || 0

  pushServerStatusLogLines(statusLogs.value, statusData, enriched.message)

  // 服务器调度进度同步写入对应评论的 per-binding 启动日志
  const bindingCommentId = resolveServerStartupBindingCommentId(statusData)
  if (bindingCommentId) {
    const messages = collectServerStartupLogMessages(statusData, enriched.message)
    const traceId =
      typeof statusData.trace_id === 'string' && statusData.trace_id.trim()
        ? statusData.trace_id.trim()
        : ''
    if (messages.length || traceId) {
      latestServerStartupStatusForBinding.value = {
        comment_id: bindingCommentId,
        messages,
        // OPT-20260812-013: 透传本条调度链路的 trace_id，供 per-binding 缓存启动 TraceId
        ...(traceId ? { trace_id: traceId } : {}),
      }
    }
  }

  // 更新服务器状态（停止虚拟机成功时后端仍用 success，需按文案区分；含 Mock）
  if (statusData.status === 'success') {
    const msg = typeof statusData.message === 'string' ? statusData.message : ''
    if (isCloudServerStopSuccessMessage(msg)) {
      statusProgress.value = 100
      isServerRunning.value = false
      isServerStarting.value = false
      serverUrl.value = ''
      deps.pauseContainerHeartbeatForRelayStop?.()
      void fetchContainerTaskUiContext()
    } else if (isAsyncStartVmSubmissionAck(statusData)) {
      isServerStarting.value = true
      isServerRunning.value = false
      statusProgress.value = Math.max(statusProgress.value || 0, statusData.progress || 5)
    } else {
      statusProgress.value = 100
      isServerRunning.value = true
      isServerStarting.value = false
      if (statusData.vm_info && statusData.vm_info.server_url) {
        serverUrl.value = statusData.vm_info.server_url
      }
      void fetchContainerTaskUiContext()
    }
  } else if (
    statusData.status === 'processing' ||
    statusData.status === 'initializing' ||
    statusData.status === 'starting'
  ) {
    isServerStarting.value = true
  } else if (statusData.status === 'stopped' || statusData.status === 'error') {
    isServerRunning.value = false
    isServerStarting.value = false
    serverUrl.value = ''
    // 与「停止虚拟机成功」对称：pause 心跳并清 endpoint，防止晚到 heartbeat 回写 connecting
    if (typeof deps.pauseContainerHeartbeatForRelayStop === 'function') {
      deps.pauseContainerHeartbeatForRelayStop()
    } else {
      stopContainerHeartbeat()
    }
    void fetchContainerTaskUiContext()
  } else if (statusData.status === 'sdk_call') {
    isServerStarting.value = true
  }
}
