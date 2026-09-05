import { computed, nextTick, ref, watch } from 'vue'
import {
  advanceCommentContainerBindings,
  ensureCommentContainerBinding,
  fetchCommentContainerBindings,
  normalizeExecutionMode,
  patchAICommentExecutionMode,
  patchHumanCommentExecutionMode,
} from '../../utils/commentExecutionApi.js'
import {
  resolveCommentDependsOnIds,
  resolveCommentExecutionMode,
} from './useCommentExecutionContext.js'
import {
  bindingRefreshRequested,
  latestPerContainerHeartbeat,
  latestServerStartupStatusForBinding,
} from './perContainerHeartbeatBus.js'
import {
  BINDING_STATUS_LOG_MESSAGES,
  BINDING_STAGE_LOG_MESSAGES,
  createBindingStatusLogs,
} from './createBindingStatusLogs.js'
import { createPerBindingStartTraceIdApi } from './perBindingStartTraceId.js'
import {
  mapBindingStatusToLifecycle,
  mapBindingLifecycleToServerStatus,
  bindingLifecycleDotClass,
  bindingLifecycleTextClass,
} from './bindingLifecycleMaps.js'
import {
  buildCommentContainerName,
  normalizeCommentContainerName,
} from '../../utils/commentContainerName.js'
import { useBindingAdvancePolling } from './useBindingAdvancePolling.js'
import { createCancelWaitingPreviousBinding } from './cancelWaitingPreviousBinding.js'
import { assemblePerBindingServerStatusProps } from './assemblePerBindingServerStatusProps.js'

export {
  BINDING_STATUS_LOG_MESSAGES,
  BINDING_STAGE_LOG_MESSAGES,
  mapBindingStatusToLifecycle,
  mapBindingLifecycleToServerStatus,
  bindingLifecycleDotClass,
  bindingLifecycleTextClass,
}

/**
 * 评论容器绑定拉取/调度 + 依赖模式 PATCH。
 *
 * OPT-20260724-022: 扩展 per-binding 服务器生命周期状态，使每个评论面板展示
 * 各自容器的独立状态而非共享 task 级别数据。
 */
function unwrap(v) {
  return v && typeof v === 'object' && 'value' in v ? v.value : v
}

export function useCommentContainerBindings(opts) {
  const bindings = ref([])
  const bindingsLoading = ref(false)
  const modeSavingByCommentId = ref({})
  const modeErrorByCommentId = ref({})
  let syncSeq = 0

  /** OPT-20260724-022: per-binding 独立心跳状态，key = commentId */
  const perBindingHeartbeat = ref({})

  /** OPT-20260812-013: per-binding 启动 TraceId（key = commentId；并行独立 CSC 启动互不覆盖） */
  const {
    perBindingStartTraceId,
    recordBindingStartTraceId,
    bindingStartTraceIdFor,
    hydrateFromBinding,
  } = createPerBindingStartTraceIdApi(() => String(unwrap(opts.taskId) || '').trim())

  const {
    perBindingStatusLogs,
    appendBindingStatusLog,
    appendBindingStageLog,
    mergeBackendBindingLogs,
    appendServerSchedulingMessages,
    clearBindingStatusLogs,
  } = createBindingStatusLogs()

  const bindingByCommentId = computed(() => {
    const map = {}
    for (const b of bindings.value || []) {
      const id = String(b?.comment_id || b?.commentId || '')
      if (id) map[id] = b
    }
    return map
  })

  function bindingStatusFor(commentId) {
    const b = bindingByCommentId.value[String(commentId || '')]
    return String(b?.status || '')
  }

  function bindingCscIdFor(commentId) {
    const b = bindingByCommentId.value[String(commentId || '')]
    return String(b?.csc_id || b?.cscId || '').trim()
  }

  /** 评论绑定容器名：API container_name / mock_container_name，否则规范推导 */
  function bindingContainerNameFor(commentId) {
    const cid = String(commentId || '').trim()
    const b = bindingByCommentId.value[cid]
    const fromApi = String(b?.container_name || b?.containerName || b?.mock_container_name || b?.mockContainerName || '').trim()
    const { tk } = ids()
    if (fromApi) return normalizeCommentContainerName(tk, cid, fromApi)
    return buildCommentContainerName(tk, cid)
  }

  function commentHasLiveBinding(commentId) {
    const s = bindingStatusFor(commentId)
    return s === 'running' || s === 'starting'
  }

  /** 是否已挂接独立 CSC（非空 csc_id）；并行评论可各自持有不同实例。 */
  function commentOwnsSharedContainer(commentId) {
    return Boolean(bindingCscIdFor(commentId))
  }

  function ids() {
    return {
      tid: String(unwrap(opts.tenantId) || '').trim(),
      wid: String(unwrap(opts.workspaceId) || '').trim(),
      tk: String(unwrap(opts.taskId) || '').trim(),
    }
  }

  function commentList() {
    const list = unwrap(opts.displayComments)
    return Array.isArray(list) ? list : []
  }

  async function refreshBindings() {
    const { tid, wid, tk } = ids()
    if (!tid || !wid || !tk) return
    bindingsLoading.value = true
    try {
      const prevMap = bindingByCommentId.value
      const fresh = await fetchCommentContainerBindings({
        tenantId: tid,
        workspaceId: wid,
        taskId: tk,
      })
      bindings.value = fresh
      // OPT-20260809-011: 优先合并服务端权威启动阶段日志（冷打开完整历史），
      // 其后状态播种/SSE 派生行按消息文本去重，避免重复并保持后端时间线在前。
      for (const b of fresh || []) {
        const cid = String(b?.comment_id || b?.commentId || '')
        if (!cid) continue
        if (Array.isArray(b?.logs)) {
          mergeBackendBindingLogs(cid, b.logs)
        }
        hydrateFromBinding(b)
      }
      // 为新绑定或状态变更的绑定播种启动日志
      for (const b of fresh || []) {
        const cid = String(b?.comment_id || b?.commentId || '')
        if (!cid) continue
        const newStatus = String(b?.status || '')
        if (!newStatus) continue
        const prevStatus = String(prevMap[cid]?.status || '')
        if (newStatus !== prevStatus) {
          appendBindingStatusLog(cid, newStatus)
        }
        // OPT-20260809-010: starting + 已挂接评论级 CSC → 回填实例分配阶段日志
        // （幂等：精确去重保证同阶段只出现一次；同时覆盖冷打开回填与轮询期间
        //   csc 挂接的实时捕捉——starting 全程均可追加，重复观察自然去重）
        if (newStatus === 'starting' && String(b?.csc_id || b?.cscId || '').trim()) {
          appendBindingStageLog(cid, 'cscAllocated')
        }
      }
    } catch (e) {
      console.warn('fetchCommentContainerBindings failed', e)
    } finally {
      bindingsLoading.value = false
    }
  }

  async function syncBindingsForComments() {
    const { tid, wid, tk } = ids()
    const list = commentList()
    if (!tid || !wid || !tk || !list.length) return
    const seq = ++syncSeq

    for (const c of list) {
      if (seq !== syncSeq) return
      if (!c?.id || c.commentKind === 'container_agent') continue
      const mode = resolveCommentExecutionMode(c)
      const dependsIds = mode === 'wait_previous' ? resolveCommentDependsOnIds(c) : []
      try {
        await ensureCommentContainerBinding({
          tenantId: tid,
          workspaceId: wid,
          taskId: tk,
          commentId: c.id,
          executionMode: mode,
          dependsOnCommentId: dependsIds.join(','),
          dependsOnCommentIds: dependsIds,
        })
      } catch (e) {
        console.warn('ensureCommentContainerBinding failed', c.id, e)
      }
    }
    if (seq !== syncSeq) return
    try {
      await advanceCommentContainerBindings({ tenantId: tid, workspaceId: wid, taskId: tk })
    } catch (e) {
      console.warn('advanceCommentContainerBindings failed', e)
    }
    if (seq !== syncSeq) return
    await refreshBindings()
  }

  // changeDependencyMode is currently unused in production UI (comment execution
  // mode is read-only after posting per OPT-20260722-052).  Retained for potential
  // admin / debug use; remove when the dead path is confirmed obsolete.
  async function changeDependencyMode({ commentId, executionMode, commentKind }) {
    const { tid, wid, tk } = ids()
    const cid = String(commentId || '')
    const mode = normalizeExecutionMode(executionMode)
    if (!tid || !tk || !cid) return
    modeSavingByCommentId.value = { ...modeSavingByCommentId.value, [cid]: true }
    modeErrorByCommentId.value = { ...modeErrorByCommentId.value, [cid]: '' }
    try {
      if (commentKind === 'ai') {
        await patchAICommentExecutionMode({
          tenantId: tid,
          workspaceId: wid,
          taskId: tk,
          commentId: cid,
          executionMode: mode,
        })
      } else {
        await patchHumanCommentExecutionMode({
          tenantId: tid,
          taskId: tk,
          commentId: cid,
          executionMode: mode,
        })
      }
      const hit = commentList().find((c) => String(c?.id) === cid)
      if (hit) hit.execution_mode = mode
      await ensureCommentContainerBinding({
        tenantId: tid,
        workspaceId: wid,
        taskId: tk,
        commentId: cid,
        executionMode: mode,
      })
      await advanceCommentContainerBindings({ tenantId: tid, workspaceId: wid, taskId: tk })
      await refreshBindings()
    } catch (e) {
      modeErrorByCommentId.value = {
        ...modeErrorByCommentId.value,
        [cid]: e?.message || '保存失败',
      }
    } finally {
      modeSavingByCommentId.value = { ...modeSavingByCommentId.value, [cid]: false }
    }
  }

  /** OPT-20260724-022: 从 binding status 推导服务器生命周期标签 */
  function bindingServerLifecycle(commentId) {
    return mapBindingStatusToLifecycle(bindingStatusFor(commentId))
  }

  /** OPT-20260724-022: binding 是否处于 "运行中" */
  function bindingIsRunning(commentId) {
    return bindingStatusFor(commentId) === 'running'
  }

  /** OPT-20260724-022: binding 是否处于 "启动中" */
  function bindingIsStarting(commentId) {
    const s = bindingStatusFor(commentId)
    return s === 'starting' || s === 'pending'
  }

  /** OPT-20260724-022: 获取 per-binding 心跳状态（含 seq/ack） */
  function bindingHeartbeatStateFor(commentId) {
    const cid = String(commentId || '').trim()
    const map = perBindingHeartbeat.value || {}
    return map[cid] || {
      status: 'idle',
      lastSuccess: null,
      seqInfo: {},
      error: '',
      logLines: [],
    }
  }

  /** OPT-20260724-022: 从 SSE container_heartbeat 更新 per-binding 心跳 */
  function applyPerBindingHeartbeatFromSse(heartbeatData) {
    if (!heartbeatData || typeof heartbeatData !== 'object') return false
    const cid = String(heartbeatData.comment_id || '').trim()
    if (!cid) return false
    const current = { ...(perBindingHeartbeat.value || {}) }
    const prev = current[cid] || { status: 'idle', lastSuccess: null, seqInfo: {}, error: '', logLines: [] }
    const seqInfo = {
      containerSeq: heartbeatData.container_seq ?? null,
      containerAck: heartbeatData.container_ack ?? null,
      saasSeq: heartbeatData.saas_seq ?? null,
      saasAck: heartbeatData.saas_ack ?? null,
      uplinkOk: heartbeatData.uplink_ok ?? null,
      downlinkOk: heartbeatData.downlink_ok ?? null,
      probeOk: heartbeatData.probe_ok ?? null,
      bidirectionalOk: heartbeatData.bidirectional_ok ?? null,
    }
    const isOk = heartbeatData.status === 'ok' && heartbeatData.bidirectional_ok === true
    current[cid] = {
      ...prev,
      status: isOk ? 'connected' : (heartbeatData.bidirectional_ok === true ? 'connected' : 'connecting'),
      lastSuccess: isOk ? new Date() : prev.lastSuccess,
      seqInfo,
      error: isOk ? '' : (heartbeatData.message || prev.error || ''),
    }
    perBindingHeartbeat.value = current

    // OPT-20260809-010: 心跳信号 → 启动阶段细粒度日志（仅 live binding；幂等去重）
    if (commentHasLiveBinding(cid)) {
      // 首个携带 seq/ack 或 status=ok 的心跳 = 容器已启动、agent 已运行
      const hasContact =
        heartbeatData.status === 'ok' || Object.values(seqInfo).some((v) => v !== null && v !== undefined)
      if (hasContact) appendBindingStageLog(cid, 'heartbeatEstablished')
      if (heartbeatData.probe_ok === true) appendBindingStageLog(cid, 'probeOk')
      if (heartbeatData.bidirectional_ok === true) appendBindingStageLog(cid, 'bidirectionalOk')
    }
    return true
  }

  function buildPerBindingServerStatusProps(commentId) {
    const cid = String(commentId || '').trim()
    const b = bindingByCommentId.value[cid] || {}
    return assemblePerBindingServerStatusProps({
      commentId: cid,
      running: bindingIsRunning(cid),
      starting: bindingIsStarting(cid),
      lifecycle: bindingServerLifecycle(cid),
      heartbeat: bindingHeartbeatStateFor(cid),
      bindingLogs: (perBindingStatusLogs.value || {})[cid] || [],
      bindings: bindings.value,
      taskStatusLogs: opts.taskStatusLogs,
      containerName: bindingContainerNameFor(cid),
      startTraceId: bindingStartTraceIdFor(cid),
      runtimeStatus: String(b.last_runtime_status || b.lastRuntimeStatus || '').trim(),
      hasServerUrl: Boolean(b.has_server_url || b.hasServerUrl),
      errorReason: String(b.error_reason || b.errorReason || '').trim(),
    })
  }

  watch(
    () => commentList().map((c) => `${c?.id}:${resolveCommentExecutionMode(c)}`).join(','),
    () => {
      syncBindingsForComments()
    },
    { immediate: true },
  )

  /** OPT-20260724-022: 监听 per-container heartbeat 总线，自动更新 per-binding 状态 */
  watch(latestPerContainerHeartbeat, (data) => {
    if (data && typeof data === 'object') {
      applyPerBindingHeartbeatFromSse(data)
    }
  })

  /** 服务器调度进度 → 写入本评论启动日志（与容器阶段日志同面板） */
  watch(latestServerStartupStatusForBinding, (data) => {
    if (!data || typeof data !== 'object') return
    const cid = String(data.comment_id || data.commentId || '').trim()
    if (!cid) return
    // OPT-20260812-013: 并行评论各自缓存启动 TraceId，避免任务级单槽被后一次启动覆盖
    if (data.trace_id) recordBindingStartTraceId(cid, data.trace_id)
    appendServerSchedulingMessages(cid, data.messages)
  })

  /** OPT-20260822-062: 云实例 Released/Terminated → 重拉绑定列表对齐服务端终态 */
  watch(bindingRefreshRequested, (req) => {
    if (!req || typeof req !== 'object') return
    void refreshBindings()
  })

  const {
    applyBindingAdvancedFromSse,
    autoAdvanceOnContainerRunning,
    startPolling,
    stopPolling,
  } = useBindingAdvancePolling({
    bindings,
    ids,
    bindingStatusFor,
    refreshBindings,
    appendBindingStatusLog,
    appendBindingStageLog,
  })

  const {
    cancelWaitingBusyByCommentId,
    isCancelWaitingBusy,
    cancelWaitingPreviousBinding,
  } = createCancelWaitingPreviousBinding({
    ids,
    refreshBindings,
    appendBindingStatusLog,
  })

  // OPT-20260724-023: 任务切换时清理 bindings / 心跳 / 终止中状态
  watch(
    () => unwrap(opts.taskId),
    (newTk, oldTk) => {
      const n = String(newTk || '').trim()
      const o = String(oldTk || '').trim()
      if (n && n !== o) {
        bindings.value = []
        perBindingHeartbeat.value = {}
        perBindingStartTraceId.value = {}
        clearBindingStatusLogs()
        cancelWaitingBusyByCommentId.value = {}
        syncSeq = 0
        stopPolling()
        void nextTick(() => startPolling())
      }
    },
  )

  function markBindingReconnecting(commentId, triggerTaskSSEReconnect) {
    const cid = String(commentId || '').trim()
    if (!cid) return
    const current = { ...(perBindingHeartbeat.value || {}) }
    const prev = current[cid] || { status: 'idle', lastSuccess: null, seqInfo: {}, error: '', logLines: [] }
    current[cid] = {
      ...prev,
      status: 'connecting',
      error: '正在重新连接容器 SSE…',
    }
    perBindingHeartbeat.value = current
    if (typeof triggerTaskSSEReconnect === 'function') {
      triggerTaskSSEReconnect()
    }
  }

  return {
    bindings,
    bindingsLoading,
    bindingByCommentId,
    bindingStatusFor,
    bindingCscIdFor,
    bindingContainerNameFor,
    commentHasLiveBinding,
    commentOwnsSharedContainer,
    refreshBindings,
    syncBindingsForComments,
    changeDependencyMode,
    modeSavingByCommentId,
    modeErrorByCommentId,
    perBindingHeartbeat,
    perBindingStartTraceId,
    recordBindingStartTraceId,
    bindingStartTraceIdFor,
    bindingServerLifecycle,
    bindingIsRunning,
    bindingIsStarting,
    bindingHeartbeatStateFor,
    applyPerBindingHeartbeatFromSse,
    buildPerBindingServerStatusProps,
    markBindingReconnecting,
    applyBindingAdvancedFromSse,
    autoAdvanceOnContainerRunning,
    stopPolling,
    cancelWaitingPreviousBinding,
    isCancelWaitingBusy,
  }
}
