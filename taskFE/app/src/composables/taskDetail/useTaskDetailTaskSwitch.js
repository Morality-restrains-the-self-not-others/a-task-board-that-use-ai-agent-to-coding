import { watch } from 'vue'

/**
 * 监听 effectiveTaskId 切换 + workspace/props 参数变化，重置旧任务状态并初始化新任务连接。
 * 提取自 TaskDetail.vue 以控制组件行数在 500 行以内。
 */
export function useTaskDetailTaskSwitch({
  $,
  props,
  route,
  effectiveTaskId,
  localTask,
  newComment,
  commentComposerChips,
  serverConfigRef,
  _internal,
}) {
  const CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS = _internal.CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS

  async function refreshWorkspaceProjectsAndTask() {
    const jobs = [$.fetchWorkspaceProjects(), $.fetchTaskDetail()]
    await Promise.all(jobs)
  }

  // effectiveTaskId 切换 → 完整状态重置
  watch(effectiveTaskId, (newId, oldId) => {
    const n = newId != null && newId !== '' ? String(newId) : ''
    const o = oldId != null && oldId !== '' ? String(oldId) : ''
    if (n === o) return
    if (n) {
      // OPT-20260724-023: 先关闭旧任务 SSE 连接再重置状态，避免跨任务串扰
      $.closeSSEConnection()
      if (!props.task) localTask.value = null
      newComment.value = ''
      commentComposerChips.value = []
      $.abortAiInstruct()
      $.activeAiInstructId.value = null
      $.aiStreamBuffer.value = ''
      $.aiStreamBusy.value = false
      $.serverStatus.value = ''
      $.statusMessage.value = ''
      $.statusCommentId.value = ''
      if ($.statusRuntimeStatus) $.statusRuntimeStatus.value = ''
      $.statusProgress.value = 0
      $.statusLogs.value = []
      $.serverUrl.value = ''
      $.isServerRunning.value = false
      $.isServerStarting.value = false
      $.cloudRuntimeStatus.value = ''
      $.cleanupContainerTimers()
      $.containerCloneProgressByKey.value = {}
      if ($.containerCloneProgressByCommentId) $.containerCloneProgressByCommentId.value = {}
      $.containerBootstrapCloneLogFull.value = ''
      $.containerBootstrapCloneLogSegments.value = null
      $.layerGraphSnapshot.value = null
      if ($.layerPanelStore) $.layerPanelStore.clear()
      if ($.containerBootstrapFailureMessage) $.containerBootstrapFailureMessage.value = ''
      if ($.containerBootstrapFailureTraceId) $.containerBootstrapFailureTraceId.value = ''
      $.layerGraphSeenLayerIdSet.value = new Set()
      _internal.layerGraphHydrateInFlight = false
      $.resetLayerGraphFetchBackoff()
      $.resetServerRuntimeLayerGraphGateCache()
      _internal.containerUnreachableProbeBackoffMs = CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS
      _internal.layerChangesPrefetchInFlight.clear()
      $.layerJobLiveOutputMap.value = {}
      $.layerGraphEditRunTargetJobId.value = ''
      $.layerGraphBusyActionKey.value = ''
      $.layerGraphCmdSending.value = false
      $.layerGraphCommandKind.value = 'trae'
      $.layerGraphAutoIterationCount.value = ''
      $.layerGraphModelProvider.value = ''
      $.layerGraphDefaultModel.value = ''
      $.layerGraphSelectedModel.value = ''
      $.layerGitIdentityLoading.value = false
      $.repoCloneIdentityByUrl.value = {}
      $.repoCloneIdentitySaveError.value = ''
      $.recloneLoadingByUrl.value = {}
      $.recloneStatusByUrl.value = {}
      if ($.recloneErrorTraceIdByUrl) $.recloneErrorTraceIdByUrl.value = {}
      $.repoRecloneGlobalLoading.value = false
      $.selectedLayerGraphNode.value = null
      $.layerGraphSelectionDismissedByUser.value = false
      $.layerGraphCommandText.value = ''
      $.layerGraphCmdError.value = ''
      if ($.layerGraphCmdErrorTraceId) $.layerGraphCmdErrorTraceId.value = ''
      $.layerGraphModelLoadError.value = ''
      $.layerGraphModelLoadErrorTraceId.value = ''
      $.progressStatusError.value = ''
      $.isForking.value = false
      $.containerPageLinkPendingReveal.value = false
      $.markContainerTransportOk()
      $.containerEndpointRegistered.value = false
      $.containerPageUrl.value = ''
      $.containerVscodeUrl.value = ''
      $.abortLayerLog()
      $.layerExecLogLoading.value = false
      $.layerExecLogTopError.value = ''
      $.layerCloneLogText.value = ''
      $.layerCloneLogFetchError.value = ''
      $.layerCloneLogFetchErrorTraceId.value = ''
      $.layerJobLogFetchError.value = ''
      $.layerJobLogFetchErrorTraceId.value = ''
      $.layerJobExecutionPayload.value = null
      $.layerChangesByLayerId.value = {}
      $.cleanupExecLogTimers()
      $.layerExecLogCopyFeedback.value = false
      $.layerAgentStepCopyFeedbackKey.value = ''
      $.establishSSEConnection(n)
      void $.fetchContainerTaskUiContext()
      void refreshWorkspaceProjectsAndTask()
    } else {
      $.closeSSEConnection()
    }
  })

  // workspace/tenant/props 参数变更 → 刷新关联数据
  watch(() => [props.workspaceId, props.tenantId, props.taskId, route.params.workspace, route.params.workspaceId, route.params.tenant, route.params.taskId], () => {
    $.fetchWorkspaceCollaborators(); $.fetchProgressStatusOptions(); $.fetchLayerGitIdentityOptions(); $.fetchLayerGraphModelOptions()
    void refreshWorkspaceProjectsAndTask()
  })
}
