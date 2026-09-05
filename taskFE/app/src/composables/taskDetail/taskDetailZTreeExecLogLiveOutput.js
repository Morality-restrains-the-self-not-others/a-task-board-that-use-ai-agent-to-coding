import { computed } from 'vue'
import {
  agentStepCardTitle,
  agentStepModelBadge,
  agentStepUsageBadge,
  agentStepCardSubtitle,
  agentStepToolResultDisplay,
  agentStepJsonPretty,
  agentStepCopyKey,
  agentStepBodyMode,
  agentStepRichHtml,
  agentStepMarkdownSource,
  agentStepPlainBodyForPre,
} from '../../utils/taskDetailExecLogFormatters.js'

const LIVE_OUTPUT_MAX = 100000
const JOB_OUTPUT_MAX = 120000

export function deriveLiveOutputDisplay(liveOutputMap, jobId) {
  const jid = String(jobId || '').trim()
  if (!jid) return ''
  const text = liveOutputMap && typeof liveOutputMap === 'object' ? liveOutputMap[jid] : ''
  if (typeof text !== 'string' || !text) return ''
  return text.length > LIVE_OUTPUT_MAX ? text.slice(-LIVE_OUTPUT_MAX) : text
}

export function deriveJobOutputDisplay(payload, liveDisplay) {
  const o = payload?.job?.output
  if (typeof o !== 'string' || !o) return ''
  const clipped = o.length > JOB_OUTPUT_MAX ? o.slice(0, JOB_OUTPUT_MAX) + '\n…(已截断)' : o
  const live = typeof liveDisplay === 'string' ? liveDisplay : ''
  if (!live) return clipped
  const normalize = (t) => String(t || '').replace(/\r\n/g, '\n').trim()
  const nClipped = normalize(clipped)
  const nLive = normalize(live)
  if (!nClipped || !nLive) return clipped
  if (nClipped === nLive || nClipped.includes(nLive) || nLive.includes(nClipped)) {
    return ''
  }
  return clipped
}

export function deriveAgentSteps(payload) {
  const steps = payload?.steps?.steps
  return Array.isArray(steps) && steps.length > 0 ? steps : []
}

export function deriveAgentStepCards(payload) {
  const steps = deriveAgentSteps(payload)
  return steps.map((step, stepIdx) => ({
    key: agentStepCopyKey(step, stepIdx),
    title: agentStepCardTitle(step),
    modelBadge: agentStepModelBadge(step),
    usageBadge: agentStepUsageBadge(step, stepIdx, steps),
    subtitle: agentStepCardSubtitle(step),
    bodyMode: agentStepBodyMode(step),
    richHtml: agentStepRichHtml(step),
    markdownSource: agentStepMarkdownSource(step),
    plainSubtitle: agentStepPlainBodyForPre(step),
    toolResultText: agentStepToolResultDisplay(step).join('\n\n'),
    timestamp: step?.timestamp ? String(step.timestamp).slice(0, 19) : '',
    error: step?.error ? String(step.error) : '',
    jsonPretty: agentStepJsonPretty(step),
    rawStep: step,
    stepIdx,
  }))
}

export function deriveJobCommandHead(payload) {
  const c = payload?.job?.command
  if (typeof c !== 'string' || !c) return ''
  return c.length > 56 ? c.slice(0, 55) + '…' : c
}

export function mergeLayerChangesIntoPayload(payload, layerChanges, selectedLayerId) {
  if (!layerChanges || typeof layerChanges !== 'object') return payload
  if (!selectedLayerId || selectedLayerId !== String(layerChanges.layer_id || '')) return payload
  if (!payload || typeof payload !== 'object') return payload
  return { ...payload, layer_changes: layerChanges }
}

/**
 * 执行日志「live output」派生（OPT-20260815-027 拆分）。
 * 从 SSE live output map 与 job payload 派生展示文本；纯 ref 输入输出。
 */
export function createLayerLiveOutputDerivations(ctx) {
  const { layerJobLiveOutputMap, zTreeLogTargets, layerJobExecutionPayload } = ctx

  function applyLiveLayerChangesToCurrentPayload(layerChanges) {
    const next = mergeLayerChangesIntoPayload(
      layerJobExecutionPayload.value,
      layerChanges,
      zTreeLogTargets.value?.layerId,
    )
    if (next !== layerJobExecutionPayload.value) {
      layerJobExecutionPayload.value = next
    }
  }

  const layerLiveOutputDisplay = computed(() =>
    deriveLiveOutputDisplay(layerJobLiveOutputMap.value, zTreeLogTargets.value.jobId),
  )

  const layerJobOutputDisplay = computed(() =>
    deriveJobOutputDisplay(layerJobExecutionPayload.value, layerLiveOutputDisplay.value),
  )

  const layerAgentSteps = computed(() => deriveAgentSteps(layerJobExecutionPayload.value))

  const layerAgentStepCards = computed(() => deriveAgentStepCards(layerJobExecutionPayload.value))

  const layerJobCommandHead = computed(() => deriveJobCommandHead(layerJobExecutionPayload.value))

  return {
    applyLiveLayerChangesToCurrentPayload,
    layerLiveOutputDisplay,
    layerJobOutputDisplay,
    layerAgentSteps,
    layerAgentStepCards,
    layerJobCommandHead,
  }
}
