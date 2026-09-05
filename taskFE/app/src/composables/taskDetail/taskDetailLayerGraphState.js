import { ref, computed, watch, nextTick } from 'vue'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import { buildTaskDetailLayerGraphZNodes, layerGraphMetaLineFromSnapshot } from './commentLayerPanelBind.js'
import { callLayerGraphJobAction as _callLayerGraphJobAction, callLayerGraphLayerDelete as _callLayerGraphLayerDelete } from './taskDetailJobActions.js'
import { pickNewestAddedLayerIdFromSnapshot } from './taskDetailZTreeExecLogState.js'
import {
  createLayerGraphModelOptionsState,
  normalizeSelectedAgentModels,
} from './taskDetailLayerGraphModelOptions.js'
import { createLayerGraphGitIdentityState } from './taskDetailLayerGraphGitIdentity.js'

export { normalizeSelectedAgentModels }

export function createTaskDetailLayerGraphState(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    effectiveTaskId,
    localTask,
    layerGraphCommandText,
    taskLayerAssociationPanelRef,
    layerGraphSnapshot,
    layerChangesByLayerId,
    selectedLayerGraphNode,
    layerGraphMergeTargetBranch,
    containerEndpointRegistered,
    containerHttpUnreachable,
    refreshLayerGraphFromServer,
    refreshZTreeExecutionLog,
    markContainerTransportUnreachableIfForwardingFailed,
    markContainerTransportOk,
  } = deps

  const layerGraphSeenLayerIdSet = ref(/** @type {Set<string>} */ (new Set()))
  const layerGraphRefreshing = ref(false)
  const layerGraphCommandKind = ref('trae')
  const layerGraphAutoIterationCount = ref('')
  const layerGraphCmdError = ref('')
  const layerGraphCmdErrorTraceId = ref('')
  const layerGraphCmdSending = ref(false)
  const layerGraphEditRunTargetJobId = ref('')
  const layerGraphBusyActionKey = ref('')
  /** 用户显式取消选中后，禁止 refresh 自动重选把指令面板再次打开 */
  const layerGraphSelectionDismissedByUser = ref(false)
  const {
    layerGraphModelProvider,
    layerGraphDefaultModel,
    layerGraphModelOptions,
    layerGraphSelectedModel,
    layerGraphModelLoading,
    layerGraphModelLoadError,
    layerGraphModelLoadErrorTraceId,
    layerGraphModelSelectDisabled,
    fetchLayerGraphModelOptions,
  } = createLayerGraphModelOptionsState({
    effectiveTenantId,
    effectiveWorkspaceId,
    localTask,
    layerGraphCmdSending,
    layerGraphCommandKind,
  })
  const {
    layerGitIdentityOptions,
    layerGitIdentityLoading,
    layerGitGithubAppOauthConnected,
    layerRepoGitIdentityLoading,
    layerRepoGitIdentityFetchError,
    layerRepoGitIdentityRows,
    perRepoGitIdentitySyncing,
    perRepoGitIdentitySyncError,
    fetchLayerGitIdentityOptions,
    containerGitIdentityRowForRepoUrl,
    containerGitIdentityLineForRepoUrl,
  } = createLayerGraphGitIdentityState({ effectiveTenantId })

  const refreshLayerGraph = async (commentId) => {
    const cid = String(commentId || '').trim()
    if (!cid && layerGraphRefreshing.value) {
      return
    }
    if (!cid) layerGraphRefreshing.value = true
    try {
      await refreshLayerGraphFromServer(true, { bypassBackoff: true, commentId: cid })
    } finally {
      if (!cid) layerGraphRefreshing.value = false
    }
  }

  const layerGraphZNodes = computed(() =>
    buildTaskDetailLayerGraphZNodes(
      layerGraphSnapshot.value,
      layerChangesByLayerId.value,
      layerGraphMergeTargetBranch.value || '',
    ),
  )

  const layerGraphMetaLine = computed(() => layerGraphMetaLineFromSnapshot(layerGraphSnapshot.value))

  const layerGraphLayerIdsKey = computed(() => {
    const s = layerGraphSnapshot.value
    if (!s || !Array.isArray(s.layers)) {
      return ''
    }
    return s.layers
      .map((l) => (l && l.layer_id != null ? String(l.layer_id).trim() : ''))
      .filter(Boolean)
      .sort()
      .join('|')
  })

  const onLayerGraphNodeSelect = (node, commentId) => {
    const store = deps.layerPanelStore
    const cid = String(commentId || '').trim()
    const clearish = !node || node.nodeKind === 'virtual' || node.nodeKind === 'cycle'
    const nextNode = clearish ? null : node
    if (store && cid) {
      store.patch(cid, { selectedNode: nextNode })
      if (!clearish && typeof refreshZTreeExecutionLog === 'function') {
        void refreshZTreeExecutionLog({ reset: true, commentId: cid })
      }
    }
    const activeId = typeof deps.resolveActiveLayerCommentId === 'function'
      ? String(deps.resolveActiveLayerCommentId() || '').trim()
      : ''
    if (cid && activeId && cid !== activeId) {
      if (clearish && node == null) {
        layerGraphSelectionDismissedByUser.value = true
      }
      return
    }
    if (clearish) {
      selectedLayerGraphNode.value = null
      layerGraphCmdError.value = ''
      layerGraphCmdErrorTraceId.value = ''
      layerGraphEditRunTargetJobId.value = ''
      if (node == null) {
        layerGraphSelectionDismissedByUser.value = true
      }
      return
    }
    layerGraphSelectionDismissedByUser.value = false
    selectedLayerGraphNode.value = node
    layerGraphCmdError.value = ''
    layerGraphCmdErrorTraceId.value = ''
    layerGraphEditRunTargetJobId.value = ''
  }

  const callLayerGraphJobAction = (jobId, action, failLabel) =>
    _callLayerGraphJobAction(jobId, action, failLabel, { ...deps, layerGraphBusyActionKey })

  const callLayerGraphLayerDelete = (layerId) =>
    _callLayerGraphLayerDelete(layerId, { ...deps, layerGraphBusyActionKey })

  const onLayerGraphJobRedo = async (jobId) => {
    if (!containerEndpointRegistered.value) {
      showRequestError('容器业务端点尚未就绪，无法重新执行')
      return
    }
    if (containerHttpUnreachable.value) {
      showRequestError('容器当前无法连接，请待恢复后重试')
      return
    }
    await callLayerGraphJobAction(jobId, 'redo', '重新执行')
  }

  const onLayerGraphJobInterrupt = async (jobId) => {
    if (!containerEndpointRegistered.value) {
      showRequestError('容器业务端点尚未就绪，无法中断')
      return
    }
    if (containerHttpUnreachable.value) {
      showRequestError('容器当前无法连接，请待恢复后重试')
      return
    }
    await callLayerGraphJobAction(jobId, 'interrupt', '中断')
  }

  const onLayerGraphJobContinue = async (jobId) => {
    if (!containerEndpointRegistered.value) {
      showRequestError('容器业务端点尚未就绪，无法继续')
      return
    }
    if (containerHttpUnreachable.value) {
      showRequestError('容器当前无法连接，请待恢复后重试')
      return
    }
    await callLayerGraphJobAction(jobId, 'continue', '继续')
  }

  const onLayerGraphJobDelete = async (jobId) => {
    if (!containerEndpointRegistered.value) {
      showRequestError('容器业务端点尚未就绪，无法删除')
      return
    }
    if (containerHttpUnreachable.value) {
      showRequestError('容器当前无法连接，请待恢复后重试')
      return
    }
    const jid = String(jobId || '').trim()
    if (!jid) return
    const ok = window.confirm('确定删除该任务及其可写层目录？此操作不可恢复。')
    if (!ok) return
    const deletedCurrent =
      selectedLayerGraphNode.value != null &&
      ((selectedLayerGraphNode.value.nodeKind === 'job' &&
        String(selectedLayerGraphNode.value.id) === jid) ||
        String(selectedLayerGraphNode.value.jobId || '') === jid)
    const done = await callLayerGraphJobAction(jid, 'delete', '删除')
    if (!done) return
    if (deletedCurrent) {
      selectedLayerGraphNode.value = null
      layerGraphCommandText.value = ''
      layerGraphCommandKind.value = 'trae'
      layerGraphCmdError.value = '节点已删除'
      layerGraphCmdErrorTraceId.value = ''
      layerGraphEditRunTargetJobId.value = ''
    }
  }

  const onLayerGraphLayerDelete = async (layerId) => {
    if (!containerEndpointRegistered.value) {
      showRequestError('容器业务端点尚未就绪，无法删除')
      return
    }
    if (containerHttpUnreachable.value) {
      showRequestError('容器当前无法连接，请待恢复后重试')
      return
    }
    const lid = String(layerId || '').trim()
    if (!lid) return
    const ok = window.confirm('确定删除该层及其所有衍生层？此操作不可恢复。')
    if (!ok) return
    const deletedCurrent =
      selectedLayerGraphNode.value != null &&
      String(selectedLayerGraphNode.value.layerId || '') === lid
    const done = await callLayerGraphLayerDelete(lid)
    if (!done) return
    if (deletedCurrent) {
      selectedLayerGraphNode.value = null
      layerGraphCommandText.value = ''
      layerGraphCommandKind.value = 'trae'
      layerGraphCmdError.value = '节点已删除'
      layerGraphCmdErrorTraceId.value = ''
      layerGraphEditRunTargetJobId.value = ''
    }
  }

  const onLayerGraphJobEditRun = async (node) => {
    if (!containerEndpointRegistered.value) {
      showRequestError('容器业务端点尚未就绪，无法执行')
      return
    }
    if (containerHttpUnreachable.value) {
      showRequestError('容器当前无法连接，请待恢复后重试')
      return
    }
    const jid = String(node?.jobId || '').trim()
    if (!jid) return
    layerGraphBusyActionKey.value = `edit:job:${jid}`
    selectedLayerGraphNode.value = node
    layerGraphCommandKind.value = node?.commandKind === 'shell' ? 'shell' : 'trae'
    layerGraphCommandText.value = typeof node?.command === 'string' ? node.command : ''
    layerGraphCmdError.value = ''
    layerGraphCmdErrorTraceId.value = ''
    layerGraphEditRunTargetJobId.value = jid
    await nextTick()
    taskLayerAssociationPanelRef.value?.focusLayerGraphCommandInput?.()
    taskLayerAssociationPanelRef.value?.selectLayerGraphCommandInput?.()
    layerGraphBusyActionKey.value = ''
  }

  return {
    layerGraphSeenLayerIdSet,
    layerGraphRefreshing,
    layerGraphCommandKind,
    layerGraphAutoIterationCount,
    layerGraphModelProvider,
    layerGraphDefaultModel,
    layerGraphModelOptions,
    layerGraphSelectedModel,
    layerGraphModelLoading,
    layerGraphModelLoadError,
    layerGraphModelLoadErrorTraceId,
    layerGraphCmdError,
    layerGraphCmdErrorTraceId,
    layerGraphCmdSending,
    layerGraphEditRunTargetJobId,
    layerGraphBusyActionKey,
    layerGraphSelectionDismissedByUser,
    layerGitIdentityOptions,
    layerGitIdentityLoading,
    layerGitGithubAppOauthConnected,
    layerRepoGitIdentityLoading,
    layerRepoGitIdentityFetchError,
    layerRepoGitIdentityRows,
    perRepoGitIdentitySyncing,
    perRepoGitIdentitySyncError,
    selectedLayerGraphNode,
    layerGraphCommandText,
    taskLayerAssociationPanelRef,
    pickNewestAddedLayerIdFromSnapshot,
    refreshLayerGraph,
    layerGraphZNodes,
    layerGraphMetaLine,
    layerGraphLayerIdsKey,
    layerGraphModelSelectDisabled,
    fetchLayerGraphModelOptions,
    fetchLayerGitIdentityOptions,
    onLayerGraphNodeSelect,
    onLayerGraphJobRedo,
    onLayerGraphJobInterrupt,
    onLayerGraphJobContinue,
    onLayerGraphJobDelete,
    onLayerGraphLayerDelete,
    onLayerGraphJobEditRun,
    normalizeSelectedAgentModels,
    containerGitIdentityRowForRepoUrl,
    containerGitIdentityLineForRepoUrl,
  }
}

export function installTaskDetailLayerGraphWatchers(lg, deps) {
  const {
    layerGraphSnapshot,
    effectiveTenantId,
    effectiveWorkspaceId,
    localTask,
  } = deps

  watch(
    lg.layerGraphLayerIdsKey,
    () => {
      const snap = layerGraphSnapshot.value
      if (!snap) {
        lg.layerGraphSeenLayerIdSet.value = new Set()
        return
      }
      const currentIds = (snap.layers || [])
        .map((l) => (l && l.layer_id != null ? String(l.layer_id).trim() : ''))
        .filter(Boolean)
      const currentSet = new Set(currentIds)
      const prev = lg.layerGraphSeenLayerIdSet.value
      const added = currentIds.filter((id) => !prev.has(id))
      const isFirstLayers = prev.size === 0 && currentIds.length > 0
      const isNewLayers = prev.size > 0 && added.length > 0
      if (isFirstLayers || isNewLayers) {
        const idList = isFirstLayers ? currentIds : added
        const newestId = pickNewestAddedLayerIdFromSnapshot(snap, idList)
        if (newestId) {
          const target = lg.layerGraphZNodes.value.find(
            (n) => n && n.nodeKind === 'layer' && String(n.layerId || '') === newestId,
          )
          if (target) {
            lg.onLayerGraphNodeSelect(target)
          }
        }
      }
      lg.layerGraphSeenLayerIdSet.value = currentSet
    },
  )

  watch(
    () => [
      effectiveTenantId?.value ?? '',
      effectiveWorkspaceId?.value ?? '',
      localTask?.value?.feature_params_source ?? '',
      localTask?.value?.personal_feature_params_config_id ?? '',
    ],
    () => {
      void lg.fetchLayerGraphModelOptions()
    },
  )
}
