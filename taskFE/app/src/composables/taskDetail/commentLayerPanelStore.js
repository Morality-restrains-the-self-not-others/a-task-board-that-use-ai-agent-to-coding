import { ref } from 'vue'
import { trimCommentId } from '../../utils/cloudComputeCommentQuery.js'

export function emptyCommentLayerPanelSlot() {
  return {
    snapshot: null,
    selectedNode: null,
    containerEndpointRegistered: false,
    containerPageUrl: '',
    containerVscodeUrl: '',
    commandKind: 'trae',
    commandText: '',
    selectedModel: '',
    autoIterationCount: '',
    cmdError: '',
    cmdErrorTraceId: '',
    cmdSending: false,
    execLogLoading: false,
    execLogTopError: '',
    cloneLogText: '',
    cloneLogFetchError: '',
    cloneLogFetchErrorTraceId: '',
    jobLogFetchError: '',
    jobLogFetchErrorTraceId: '',
    jobExecutionPayload: null,
    liveOutputMap: {},
    layerChangesByLayerId: {},
    refreshing: false,
    hydrateInFlight: false,
    uiContextFetched: false,
  }
}

const REF_KEY_TO_SLOT = {
  layerGraphSnapshot: 'snapshot',
  selectedLayerGraphNode: 'selectedNode',
  containerEndpointRegistered: 'containerEndpointRegistered',
  containerPageUrl: 'containerPageUrl',
  containerVscodeUrl: 'containerVscodeUrl',
  layerGraphCommandKind: 'commandKind',
  layerGraphCommandText: 'commandText',
  layerGraphSelectedModel: 'selectedModel',
  layerGraphAutoIterationCount: 'autoIterationCount',
  layerGraphCmdError: 'cmdError',
  layerGraphCmdErrorTraceId: 'cmdErrorTraceId',
  layerGraphCmdSending: 'cmdSending',
  layerExecLogLoading: 'execLogLoading',
  layerExecLogTopError: 'execLogTopError',
  layerCloneLogText: 'cloneLogText',
  layerCloneLogFetchError: 'cloneLogFetchError',
  layerCloneLogFetchErrorTraceId: 'cloneLogFetchErrorTraceId',
  layerJobLogFetchError: 'jobLogFetchError',
  layerJobLogFetchErrorTraceId: 'jobLogFetchErrorTraceId',
  layerJobExecutionPayload: 'jobExecutionPayload',
  layerJobLiveOutputMap: 'liveOutputMap',
  layerChangesByLayerId: 'layerChangesByLayerId',
}

/**
 * 评论级层图 / 端点 / 执行日志 / 命令框。两评论并发启动不得共用页面级单例。
 */
export function createCommentLayerPanelStore() {
  const state = ref({})

  function get(commentId) {
    const cid = trimCommentId(commentId)
    if (!cid) return emptyCommentLayerPanelSlot()
    return state.value[cid] || emptyCommentLayerPanelSlot()
  }

  function patch(commentId, partial) {
    const cid = trimCommentId(commentId)
    if (!cid || !partial || typeof partial !== 'object') return
    state.value = {
      ...state.value,
      [cid]: { ...emptyCommentLayerPanelSlot(), ...state.value[cid], ...partial },
    }
  }

  function refsFor(commentId) {
    const cid = trimCommentId(commentId)
    const make = (slotKey) => ({
      get value() {
        return get(cid)[slotKey]
      },
      set value(v) {
        patch(cid, { [slotKey]: v })
      },
    })
    const out = {}
    for (const [refName, slotKey] of Object.entries(REF_KEY_TO_SLOT)) {
      out[refName] = make(slotKey)
    }
    return out
  }

  function commentIdOwningJob(jobId) {
    const jid = String(jobId || '').trim()
    if (!jid) return ''
    for (const [cid, slot] of Object.entries(state.value || {})) {
      const jobs = slot?.snapshot?.jobs
      if (!Array.isArray(jobs)) continue
      if (jobs.some((j) => j && String(j.id) === jid)) return cid
    }
    return ''
  }

  function commentIdOwningLayer(layerId) {
    const lid = String(layerId || '').trim()
    if (!lid) return ''
    for (const [cid, slot] of Object.entries(state.value || {})) {
      const layers = slot?.snapshot?.layers
      if (!Array.isArray(layers)) continue
      if (layers.some((l) => l && String(l.layer_id) === lid)) return cid
    }
    return ''
  }

  function clear() {
    state.value = {}
  }

  return { state, get, patch, refsFor, commentIdOwningJob, commentIdOwningLayer, clear }
}
