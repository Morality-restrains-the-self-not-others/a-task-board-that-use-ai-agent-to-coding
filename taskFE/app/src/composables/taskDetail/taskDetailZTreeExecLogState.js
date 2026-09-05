import { ref, computed, nextTick, unref } from 'vue'
import { isCommentExecutionReleased } from './useCommentExecutionContext.js'
import { layerChangesPayloadImpliesWorktreeDirty } from '../../utils/layerChangesDirty.js'
import { refreshZTreeExecutionLog as _refreshZTreeExecLog } from './taskDetailExecLog.js'
import {
  createActiveJobExecLogPoller,
  isActiveJobStatus,
} from './activeJobExecLogPoller.js'
import {
  agentStepCardTitle,
  agentStepJsonPretty,
  agentStepCopyKey,
} from '../../utils/taskDetailExecLogFormatters.js'
import { getCookie } from '../../utils/cookieUtils.js'
import { resolveContainerUiContextCommentId } from './resolveContainerUiContextCommentId.js'
import { resolveZTreeLogTargets } from './taskDetailZTreeExecLogDerived.js'
import { createLayerChangesController } from './taskDetailZTreeExecLogLayerChanges.js'
import { createLayerLiveOutputDerivations } from './taskDetailZTreeExecLogLiveOutput.js'

export { layerChangesPayloadImpliesWorktreeDirty }
export {
  ztreeLogCreatedAtMs,
  resolveZTreeLogTargets,
  layerChangePathIsGitInternal,
  normalizeLayerChangesPayload,
  mergeLayerChangesPage,
  layerChangesContentFingerprint,
  pickNewestAddedLayerIdFromSnapshot,
} from './taskDetailZTreeExecLogDerived.js'
export { installTaskDetailZTreeExecLogWatchers } from './taskDetailZTreeExecLogWatchers.js'

export function createTaskDetailZTreeExecLogState(deps) {
  const {
    selectedLayerGraphNode,
    layerGraphSnapshot,
    layerChangesByLayerId,
    layerGraphCommandText,
    taskLayerAssociationPanelRef,
    effectiveTenantId,
    effectiveWorkspaceId,
    effectiveTaskId,
    containerEndpointRegistered,
    containerHttpUnreachable,
    taskRepoRows,
    repoCloneIdentityIdForUrl,
    commentComposerChips,
    bumpProjectFileTreeRefresh,
    markContainerTransportUnreachableIfForwardingFailed,
    markContainerTransportOk,
  } = deps

  let layerLogAbortController = null

  const containerReleased = computed(() => {
    if (unref(deps.containerReleased) || unref(deps.serverRuntimeNotServing)) return true
    const cid = String(unref(deps.commentId) || '').trim()
    const binding = typeof deps.bindingStatusFor === 'function' ? deps.bindingStatusFor(cid) : ''
    return isCommentExecutionReleased(binding, '')
  })

  let layerExecLogCopyFeedbackTimer = null
  let layerAgentStepCopyFeedbackTimer = null

  const layerExecLogLoading = ref(false)
  const layerExecLogTopError = ref('')
  const layerCloneLogText = ref('')
  const layerCloneLogFetchError = ref('')
  const layerCloneLogFetchErrorTraceId = ref('')
  const layerJobLogFetchError = ref('')
  const layerJobLogFetchErrorTraceId = ref('')
  const layerJobExecutionPayload = ref(null)
  const layerJobLiveOutputMap = ref({})
  const layerChangesRefreshBusy = ref(false)
  const layerChangesRefreshError = ref('')
  const layerChangesRefreshErrorTraceId = ref('')
  const layerChangesLoadMoreBusy = ref(false)

  const selectedLayerGraphFileTreeLayerId = computed(() => {
    const n = selectedLayerGraphNode.value
    const jobs = layerGraphSnapshot.value?.jobs
    const lid = resolveZTreeLogTargets(n, jobs).layerId
    return lid ? String(lid).trim() : ''
  })

  const zTreeLogTargets = computed(() =>
    resolveZTreeLogTargets(selectedLayerGraphNode.value, layerGraphSnapshot.value?.jobs),
  )

  const selectedZTreeLayerChangesPanel = computed(() => {
    const lid = selectedLayerGraphFileTreeLayerId.value
    if (!lid) return null
    const src = layerChangesByLayerId.value[lid]
    if (!src || typeof src !== 'object') return null
    const changes = Array.isArray(src.changes) ? src.changes : []
    const rawCount = Number(src.change_count)
    const count = Number.isFinite(rawCount) && rawCount >= 0 ? Math.floor(rawCount) : changes.length
    const hasMore = Boolean(src.has_more) || changes.length < count
    return {
      layer_id: lid,
      changes,
      displayCount: count,
      truncated: Boolean(src.truncated),
      has_more: hasMore,
      next_offset:
        src.next_offset != null ? Number(src.next_offset) : changes.length,
      detail: typeof src.detail === 'string' ? src.detail : '',
      loadMoreBusy: layerChangesLoadMoreBusy.value,
      truncatedTraceId: String(src.fetch_trace_id || '').trim(),
    }
  })

  const layerChangesRefreshEnabled = computed(() =>
    Boolean(
      containerEndpointRegistered.value &&
        zTreeLogTargets.value.jobId &&
        effectiveTenantId.value &&
        effectiveWorkspaceId.value &&
        effectiveTaskId.value &&
        resolveContainerUiContextCommentId(deps),
    ),
  )

  const selectedJobRow = computed(() => {
    const jobId = zTreeLogTargets.value.jobId
    if (!jobId) return null
    const jobs = layerGraphSnapshot.value?.jobs
    if (!Array.isArray(jobs)) return null
    return jobs.find((j) => j && String(j.id) === String(jobId)) || null
  })

  /** 选中任务仍在执行时禁止暂存区操作 */
  const layerChangesGitStagedActionsBlocked = computed(() => {
    if (!containerEndpointRegistered.value || containerHttpUnreachable?.value) {
      return true
    }
    const job = selectedJobRow.value
    if (!job) return false
    return isActiveJobStatus(job.status)
  })

  /** 关联仓库未全部选择克隆身份时禁止提交到仓库 */
  const layerChangesGitCommitIdentityBlocked = computed(() => {
    const rows = taskRepoRows?.value
    if (!Array.isArray(rows) || rows.length === 0) return false
    return rows.some((r) => !String(repoCloneIdentityIdForUrl?.(r.url) || '').trim())
  })

  const selectedLayerGraphNodeLogKey = computed(() => {
    const n = selectedLayerGraphNode.value
    if (!n || n.nodeKind === 'virtual' || n.nodeKind === 'cycle') return ''
    return `${String(n.nodeKind || '')}:${String(n.id ?? '')}`
  })

  const layerChangesController = createLayerChangesController({
    deps,
    layerChangesByLayerId,
    layerChangesRefreshBusy,
    layerChangesRefreshError,
    layerChangesRefreshErrorTraceId,
    layerChangesLoadMoreBusy,
    zTreeLogTargets,
    selectedLayerGraphFileTreeLayerId,
    layerGraphSnapshot,
    effectiveTenantId,
    effectiveWorkspaceId,
    effectiveTaskId,
    containerEndpointRegistered,
    bumpProjectFileTreeRefresh,
    resolveContainerUiContextCommentId,
  })
  const {
    layerChangesPrefetchInFlight,
    ingestLayerChangesFromExecutionPayload,
    refreshSelectedLayerChanges,
    loadMoreSelectedLayerChanges,
    prefetchLayerChangeSummariesForDirtyLayers,
  } = layerChangesController

  const liveOutput = createLayerLiveOutputDerivations({
    layerJobLiveOutputMap,
    zTreeLogTargets,
    layerJobExecutionPayload,
  })
  const {
    applyLiveLayerChangesToCurrentPayload,
    layerLiveOutputDisplay,
    layerJobOutputDisplay,
    layerAgentSteps,
    layerAgentStepCards,
    layerJobCommandHead,
  } = liveOutput

  const layerExecLogCopyText = computed(() => {
    const lines = []
    if (layerExecLogTopError.value) {
      lines.push(`[错误] ${layerExecLogTopError.value}`)
    }
    const targets = zTreeLogTargets.value
    if (targets.layerId) {
      if (layerCloneLogFetchError.value) {
        lines.push(`[错误] ${layerCloneLogFetchError.value}`)
      } else if (layerCloneLogText.value) {
        lines.push(layerCloneLogText.value)
      }
    }
    if (targets.jobId) {
      lines.push('=== 任务执行 ===')
      const live = layerJobLiveOutputMap.value[targets.jobId]
      if (typeof live === 'string' && live) {
        lines.push(`--- 实时输出（SSE） ---\n${live}`)
      }
      if (layerJobLogFetchError.value) {
        lines.push(`[错误] ${layerJobLogFetchError.value}`)
      } else if (layerJobExecutionPayload.value?.job) {
        const j = layerJobExecutionPayload.value.job
        const cmd = typeof j.command === 'string' ? j.command : ''
        lines.push(`id: ${j.id}\nstatus: ${j.status || '—'}\ncommand: ${cmd}`)
        const sn = layerJobExecutionPayload.value.steps?.note
        if (sn) {
          lines.push(`steps: ${sn}`)
        }
        const rawOut = typeof j.output === 'string' ? j.output : ''
        const out =
          rawOut.length > 500000
            ? rawOut.slice(0, 500000) + '\n…(已截断)'
            : rawOut
        if (out) {
          lines.push(`--- 控制台输出 ---\n${out}`)
        } else {
          lines.push('（暂无控制台输出）')
        }
        const steps = layerJobExecutionPayload.value.steps?.steps
        if (Array.isArray(steps) && steps.length) {
          lines.push('--- 代理步骤（按步） ---')
          for (let i = 0; i < steps.length; i++) {
            const one = steps[i]
            const title = agentStepCardTitle(one)
            try {
              let sj = JSON.stringify(one, null, 2)
              if (sj.length > 200000) {
                sj = sj.slice(0, 200000) + '\n…(已截断)'
              }
              lines.push(`--- ${title} ---\n${sj}`)
            } catch {
              lines.push(`--- ${title} ---\n(无法序列化)`)
            }
          }
        }
      } else {
        lines.push('（暂无任务日志）')
      }
    }
    if (!targets.layerId && !targets.jobId && !layerExecLogTopError.value) {
      lines.push('当前节点无关联可写层或任务')
    }
    return lines.join('\n\n')
  })

  const layerExecLogCopyable = computed(
    () => !layerExecLogLoading.value && layerExecLogCopyText.value.trim().length > 0,
  )

  const layerExecLogClearable = computed(() => layerExecLogCopyable.value)

  const profileGitIdentitiesApiPath = computed(() => {
    const userId = String(getCookie('userId') || '').trim()
    if (!userId) {
      return ''
    }
    return `/api/git-identities/user/${encodeURIComponent(userId)}/`
  })

  const layerExecLogCopyFeedback = ref(false)
  const layerAgentStepCopyFeedbackKey = ref('')

  function formatRichInteractPayload(p) {
    if (!p || typeof p !== 'object') return ''
    if (p.kind === 'radio') return `[选择] ${p.label || p.value}${p.value ? ` (${p.value})` : ''}`
    if (p.kind === 'checkbox') return `[${p.selected ? '勾选' : '取消'}] ${p.label || p.value || ''}`
    if (p.kind === 'action') return `[${p.action || 'action'}]${p.param ? ` ${p.param}` : ''}`
    return ''
  }

  function onAgentStepRichInteract(payload) {
    const line = formatRichInteractPayload(payload)
    if (!line) return
    const cur = layerGraphCommandText.value || ''
    layerGraphCommandText.value = cur ? `${cur}\n${line}` : line
    if (commentComposerChips) {
      commentComposerChips.value = [...commentComposerChips.value, line].slice(-12)
    }
    nextTick(() => {
      taskLayerAssociationPanelRef.value?.focusLayerGraphCommandInput?.()
    })
  }

  async function copyAgentStepJson(step, stepIdx) {
    const text = agentStepJsonPretty(step)
    if (!text) return
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
      } else {
        const ta = document.createElement('textarea')
        ta.value = text
        ta.setAttribute('readonly', '')
        ta.style.position = 'fixed'
        ta.style.left = '-9999px'
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
      }
      layerAgentStepCopyFeedbackKey.value = agentStepCopyKey(step, stepIdx)
      if (layerAgentStepCopyFeedbackTimer) {
        clearTimeout(layerAgentStepCopyFeedbackTimer)
      }
      layerAgentStepCopyFeedbackTimer = setTimeout(() => {
        layerAgentStepCopyFeedbackKey.value = ''
        layerAgentStepCopyFeedbackTimer = null
      }, 2000)
    } catch (e) {
      console.error('复制步骤 JSON 失败', e)
    }
  }

  const copyLayerExecLog = async () => {
    const text = layerExecLogCopyText.value
    if (!text.trim()) return
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
      } else {
        const ta = document.createElement('textarea')
        ta.value = text
        ta.setAttribute('readonly', '')
        ta.style.position = 'fixed'
        ta.style.left = '-9999px'
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
      }
      layerExecLogCopyFeedback.value = true
      if (layerExecLogCopyFeedbackTimer) {
        clearTimeout(layerExecLogCopyFeedbackTimer)
      }
      layerExecLogCopyFeedbackTimer = setTimeout(() => {
        layerExecLogCopyFeedback.value = false
        layerExecLogCopyFeedbackTimer = null
      }, 2000)
    } catch (e) {
      console.error('复制日志失败', e)
    }
  }

  const clearLayerExecLog = (commentId) => {
    const cid = String(commentId || '').trim()
    if (cid && deps.layerPanelStore) {
      deps.layerPanelStore.patch(cid, {
        execLogLoading: false,
        execLogTopError: '',
        cloneLogText: '',
        cloneLogFetchError: '',
        cloneLogFetchErrorTraceId: '',
        jobLogFetchError: '',
        jobLogFetchErrorTraceId: '',
        jobExecutionPayload: null,
      })
      const activeId = resolveContainerUiContextCommentId(deps)
      if (activeId && cid !== activeId) return
    }
    if (layerLogAbortController) {
      layerLogAbortController.abort()
      layerLogAbortController = null
    }
    layerExecLogLoading.value = false
    layerExecLogTopError.value = ''
    layerCloneLogText.value = ''
    layerCloneLogFetchError.value = ''
    layerCloneLogFetchErrorTraceId.value = ''
    layerJobLogFetchError.value = ''
    layerJobLogFetchErrorTraceId.value = ''
    layerJobExecutionPayload.value = null
    layerChangesByLayerId.value = {}
    if (layerExecLogCopyFeedbackTimer) {
      clearTimeout(layerExecLogCopyFeedbackTimer)
      layerExecLogCopyFeedbackTimer = null
    }
    layerExecLogCopyFeedback.value = false
    if (layerAgentStepCopyFeedbackTimer) {
      clearTimeout(layerAgentStepCopyFeedbackTimer)
      layerAgentStepCopyFeedbackTimer = null
    }
    layerAgentStepCopyFeedbackKey.value = ''
  }

  const refreshZTreeExecutionLog = (opts) => {
    const cid = String(opts?.commentId || '').trim() || resolveContainerUiContextCommentId(deps)
    const slice = cid && deps.layerPanelStore ? deps.layerPanelStore.refsFor(cid) : null
    return _refreshZTreeExecLog(opts, {
      selectedLayerGraphNode: slice?.selectedLayerGraphNode || selectedLayerGraphNode,
      layerGraphSnapshot: slice?.layerGraphSnapshot || layerGraphSnapshot,
      containerEndpointRegistered: slice?.containerEndpointRegistered || containerEndpointRegistered,
      effectiveTenantId,
      effectiveWorkspaceId,
      effectiveTaskId,
      layerExecLogLoading: slice?.layerExecLogLoading || layerExecLogLoading,
      layerExecLogTopError: slice?.layerExecLogTopError || layerExecLogTopError,
      layerCloneLogText: slice?.layerCloneLogText || layerCloneLogText,
      layerCloneLogFetchError: slice?.layerCloneLogFetchError || layerCloneLogFetchError,
      layerCloneLogFetchErrorTraceId: slice?.layerCloneLogFetchErrorTraceId || layerCloneLogFetchErrorTraceId,
      layerJobLogFetchError: slice?.layerJobLogFetchError || layerJobLogFetchError,
      layerJobLogFetchErrorTraceId: slice?.layerJobLogFetchErrorTraceId || layerJobLogFetchErrorTraceId,
      layerJobExecutionPayload: slice?.layerJobExecutionPayload || layerJobExecutionPayload,
      layerChangesByLayerId: slice?.layerChangesByLayerId || layerChangesByLayerId,
      ingestLayerChangesFromExecutionPayload,
      markContainerTransportUnreachableIfForwardingFailed,
      markContainerTransportOk,
      resolveZTreeLogTargets,
      containerReleased,
      serverRuntimeNotServing: deps.serverRuntimeNotServing,
      commentId: cid || deps.commentId,
      effectiveCommentId: deps.effectiveCommentId,
      bindingStatusFor: deps.bindingStatusFor,
      bindingCscIdFor: deps.bindingCscIdFor,
      runtimeStatusFor: deps.runtimeStatusFor,
      displayComments: deps.displayComments,
      activeContainerAgentId: deps.activeContainerAgentId,
      get layerLogAbortController() {
        return layerLogAbortController
      },
      set layerLogAbortController(v) {
        layerLogAbortController = v
      },
    })
  }

  const activeJobExecLogPoller = createActiveJobExecLogPoller({
    isEndpointReady: () =>
      Boolean(containerEndpointRegistered.value && !containerHttpUnreachable?.value),
    getSelectedJobStatus: () => {
      const jobId = zTreeLogTargets.value.jobId
      if (!jobId) return null
      const job = selectedJobRow.value
      return job ? { jobId, status: String(job.status || '') } : { jobId, status: '' }
    },
    refresh: () => refreshZTreeExecutionLog(),
  })

  return {
    layerExecLogLoading,
    layerExecLogTopError,
    layerCloneLogText,
    layerCloneLogFetchError,
    layerCloneLogFetchErrorTraceId,
    layerJobLogFetchError,
    layerJobLogFetchErrorTraceId,
    layerJobExecutionPayload,
    layerJobLiveOutputMap,
    layerChangesRefreshBusy,
    layerChangesRefreshError,
    layerChangesRefreshErrorTraceId,
    layerChangesLoadMoreBusy,
    selectedLayerGraphFileTreeLayerId,
    zTreeLogTargets,
    selectedZTreeLayerChangesPanel,
    layerChangesRefreshEnabled,
    layerChangesGitStagedActionsBlocked,
    layerChangesGitCommitIdentityBlocked,
    selectedLayerGraphNodeLogKey,
    ingestLayerChangesFromExecutionPayload,
    refreshSelectedLayerChanges,
    loadMoreSelectedLayerChanges,
    prefetchLayerChangeSummariesForDirtyLayers,
    applyLiveLayerChangesToCurrentPayload,
    layerLiveOutputDisplay,
    layerJobOutputDisplay,
    layerAgentSteps,
    layerAgentStepCards,
    layerJobCommandHead,
    layerExecLogCopyText,
    layerExecLogCopyable,
    layerExecLogClearable,
    profileGitIdentitiesApiPath,
    layerExecLogCopyFeedback,
    layerAgentStepCopyFeedbackKey,
    onAgentStepRichInteract,
    copyAgentStepJson,
    copyLayerExecLog,
    clearLayerExecLog,
    refreshZTreeExecutionLog,
    containerReleased,
    activeJobExecLogPoller,
    layerChangesPrefetchInFlight,
    get layerLogAbortController() { return layerLogAbortController },
    set layerLogAbortController(v) { layerLogAbortController = v },
  }
}
