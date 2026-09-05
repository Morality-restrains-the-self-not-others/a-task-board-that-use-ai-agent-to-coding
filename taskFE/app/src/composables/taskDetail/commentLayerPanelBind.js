import { snapshotHasRealWritableLayer, bootstrapFailureMessageFromLayers, renameBootstrapAnchorPendingNameOnFailure } from '../../utils/layerZtreeBootstrapAnchor.js'
import {
  buildZTreeNodesSerialFromLayers,
  normalizeLayerGitDirty,
} from '../../utils/layerZtreeNodes.js'
import {
  formatLayerSubmitFileChangesCaption,
  layerChangesDisplayCount,
  layerChangesPayloadImpliesWorktreeDirty,
  submitTitleForParentDiffOnly,
} from '../../utils/layerChangesDirty.js'
import { resolveZTreeLogTargets } from './taskDetailZTreeExecLogDerived.js'
import {
  deriveLiveOutputDisplay,
  deriveJobOutputDisplay,
  deriveAgentStepCards,
  deriveJobCommandHead,
} from './taskDetailZTreeExecLogLiveOutput.js'

export function buildTaskDetailLayerGraphZNodes(
  snapshot,
  layerChangesByLayerId,
  mergeTargetBranch,
  containerReleased = false,
) {
  const s = snapshot
  if (!s || !Array.isArray(s.layers) || s.layers.length === 0) {
    return []
  }
  const base = buildZTreeNodesSerialFromLayers(s.layers, Array.isArray(s.jobs) ? s.jobs : [], {
    bootstrapLayerId: s.bootstrap_layer_id ? String(s.bootstrap_layer_id) : '',
    mergeTargetBranch: mergeTargetBranch || '',
    containerReleased: Boolean(containerReleased),
  })
  const byId = layerChangesByLayerId || {}
  return base.map((n) => {
    const layerId = n.layerId != null ? String(n.layerId).trim() : ''
    const payload = layerId ? byId[layerId] : null
    if (!layerId || !payload || typeof payload !== 'object') {
      return { ...n, submitFileChangesLabel: '', submitFileChangesTitle: '' }
    }
    const layerRow = (s.layers || []).find(
      (l) => String(l?.layer_id || '').trim() === layerId,
    )
    const snapDirty = layerRow ? normalizeLayerGitDirty(layerRow.git_worktree_dirty) : null
    const impliesDirty =
      layerChangesPayloadImpliesWorktreeDirty(payload) && snapDirty !== false
    // OPT-20260822-001(3)：容器已释放时禁止「提交并创建PR」被 impliesDirty 重新启用
    if (!n.canSubmit && !impliesDirty) {
      return { ...n, submitFileChangesLabel: '', submitFileChangesTitle: '' }
    }
    const caption = formatLayerSubmitFileChangesCaption(payload, { impliesDirty })
    let out = {
      ...n,
      submitFileChangesLabel: caption.label,
      submitFileChangesTitle: caption.title,
    }
    if (impliesDirty && !containerReleased) {
      const wasPushAllowed = !n.pushDisabled
      out = {
        ...out,
        canSubmit: true,
        submitDisabled: false,
        submitTitle: '',
        pushDisabled: true,
        pushTitle: wasPushAllowed ? '请先提交' : n.pushTitle,
        ztStyle:
          n.nodeKind === 'layer' && n.ztStyle === 'clean' ? 'dirty' : n.ztStyle,
      }
    } else if (n.canSubmit && n.submitDisabled && caption.parentDiffOnly) {
      out = {
        ...out,
        submitTitle: submitTitleForParentDiffOnly(layerChangesDisplayCount(payload)),
      }
    }
    return out
  })
}

function instructionJobStatusIsActive(raw) {
  const v = String(raw || '').trim().toLowerCase()
  return v === 'pending' || v === 'running'
}

function formatInstructionFinishedAt(iso) {
  const ms = Date.parse(iso || '')
  if (!Number.isFinite(ms)) return ''
  return new Date(ms).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

/** 最近一条非 clone 指令：进行中为 pending，否则为 finished_at 本地时间。 */
export function latestInstructionFinishCaption(jobs) {
  const list = (Array.isArray(jobs) ? jobs : []).filter(
    (j) => j && String(j.command_kind || '').toLowerCase() !== 'clone',
  )
  if (!list.length) return ''
  const sorted = [...list].sort((a, b) => {
    const da = Date.parse(a.created_at || '')
    const db = Date.parse(b.created_at || '')
    const ma = Number.isFinite(da) ? da : 0
    const mb = Number.isFinite(db) ? db : 0
    if (ma !== mb) return ma - mb
    return String(a.id || '').localeCompare(String(b.id || ''))
  })
  const last = sorted[sorted.length - 1]
  if (instructionJobStatusIsActive(last.status)) return '最近指令完成 pending'
  const formatted = formatInstructionFinishedAt(last.finished_at)
  return formatted ? `最近指令完成 ${formatted}` : '最近指令完成 —'
}

export function layerGraphMetaLineFromSnapshot(snapshot) {
  const s = snapshot
  if (!s) return ''
  const parts = []
  const nL = Array.isArray(s.layers) ? s.layers.length : 0
  const nJ = Array.isArray(s.jobs) ? s.jobs.length : 0
  if (nL) parts.push(`可写层 ${nL} 个（串行 · 按创建时间旧→新）`)
  if (nJ) parts.push(`任务 ${nJ} 个`)
  const finish = latestInstructionFinishCaption(s.jobs)
  if (finish) parts.push(finish)
  if (s.layers_root) parts.push(`服务扫描 ${s.layers_root}`)
  return parts.join(' · ')
}

export function fileTreeLayerIdFromSlot(slot) {
  const jobs = slot?.snapshot?.jobs
  const lid = resolveZTreeLogTargets(slot?.selectedNode, jobs).layerId
  return lid ? String(lid).trim() : ''
}

export function layerChangesPanelFromSlot(slot) {
  const lid = fileTreeLayerIdFromSlot(slot)
  if (!lid) return null
  const src = slot?.layerChangesByLayerId?.[lid]
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
    next_offset: src.next_offset != null ? Number(src.next_offset) : changes.length,
    detail: typeof src.detail === 'string' ? src.detail : '',
    loadMoreBusy: false,
    truncatedTraceId: String(src.fetch_trace_id || '').trim(),
  }
}

export function layerPanelViewFromSlot(slot, extras = {}) {
  const s = slot || {}
  const zNodes = renameBootstrapAnchorPendingNameOnFailure(
    buildTaskDetailLayerGraphZNodes(
      s.snapshot,
      s.layerChangesByLayerId,
      extras.mergeTargetBranch || '',
      extras.containerReleased,
    ),
    bootstrapFailureMessageFromLayers(s.snapshot?.layers) ||
      s.cloneLogText ||
      extras.bootstrapFailureRaw ||
      '',
  )
  const zTreeLogTargets = resolveZTreeLogTargets(s.selectedNode, s.snapshot?.jobs)
  const live = deriveLiveOutputDisplay(s.liveOutputMap, zTreeLogTargets.jobId)
  const payload = s.jobExecutionPayload
  const hasLogBody = Boolean(live || s.cloneLogText || payload?.job)
  return {
    layerGraphZNodes: zNodes,
    layerGraphMetaLine: layerGraphMetaLineFromSnapshot(s.snapshot),
    selectedLayerGraphNode: s.selectedNode,
    selectedLayerGraphFileTreeLayerId: fileTreeLayerIdFromSlot(s),
    zTreeLogTargets,
    containerEndpointRegistered: Boolean(s.containerEndpointRegistered),
    containerPageUrl: s.containerPageUrl || '',
    displayContainerVscodeUrl: extras.displayContainerVscodeUrl || s.containerVscodeUrl || '',
    layerGraphRefreshing: Boolean(s.refreshing),
    layerExecLogLoading: Boolean(s.execLogLoading),
    layerExecLogTopError: s.execLogTopError || '',
    layerCloneLogText: s.cloneLogText || '',
    layerCloneLogFetchError: s.cloneLogFetchError || '',
    layerCloneLogFetchErrorTraceId: s.cloneLogFetchErrorTraceId || '',
    layerJobLogFetchError: s.jobLogFetchError || '',
    layerJobLogFetchErrorTraceId: s.jobLogFetchErrorTraceId || '',
    layerJobExecutionPayload: s.jobExecutionPayload,
    layerGraphCmdError: s.cmdError || '',
    layerGraphCmdErrorTraceId: s.cmdErrorTraceId || '',
    layerGraphCmdSending: Boolean(s.cmdSending),
    layerLiveOutputDisplay: live,
    layerJobOutputDisplay: deriveJobOutputDisplay(payload, live),
    layerAgentStepCards: deriveAgentStepCards(payload),
    layerJobCommandHead: deriveJobCommandHead(payload),
    selectedZTreeLayerChangesPanel: layerChangesPanelFromSlot(s),
    layerExecLogCopyable: hasLogBody && !s.execLogLoading,
    layerExecLogClearable: hasLogBody && !s.execLogLoading,
    containerReleased: Boolean(extras.containerReleased),
    layerGraphHasRealWritableLayer: snapshotHasRealWritableLayer(s.snapshot?.layers),
    layerGraphBootstrapFailureMessage: bootstrapFailureMessageFromLayers(s.snapshot?.layers),
  }
}
