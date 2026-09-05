import { ref, computed, watch } from 'vue'
import { isLoopbackHttpUrl, replaceHttpUrlHostname } from '../../utils/httpUrlHost.js'
import {
  buildServerRuntimeStatusDetails,
  resolveRuntimeUptimeSource,
} from '../../utils/serverRuntimeStatusDetails.js'
import {
  notifyRuntimeAbsentHydrate,
  notifyRuntimeHydrate,
  shouldRematchRuntimeHydrate,
} from './serverRuntimeHydrate.js'
import { isServerRuntimeNoInstanceMessage, isServerRuntimeStartFailedMessage } from '../../utils/serverRuntimeAbsent.js'
import {
  buildJumpToServerUrl,
  serverJumpDefaultPort,
  serverVscodeDefaultPort,
} from '../../utils/serverConfigJumpUrl.js'
import { resolveTaskWorkspaceId } from '../../utils/serverConfigRouteHelpers.js'
import {
  commentIdFromButtonArg,
  createScopedCommentIdMemory,
} from '../../utils/cloudComputeCommentQuery.js'
import {
  fetchServerContentApi,
  fetchServerRuntimeStatusApi,
  fetchServerStartHistoryApi,
  openWorkbenchLinkApi,
  stopServerApi,
} from './useServerConfigRuntimeFetch.js'
import { createCommentRuntimeSnapshotStore } from './commentRuntimeSnapshotStore.js'
import { createCommentContentSnapshotStore } from './commentContentSnapshotStore.js'
import { buildCommentRuntimePanelDisplay } from './buildCommentRuntimePanelDisplay.js'
import { commentIdForRuntimeRefresh, shouldRefreshRuntimeOnStatusMessage, applyPushedRuntimeSnapshot } from './runtimeStatusRefreshPolicy.js'
import { isReleasedRuntimeStatus } from './useCommentExecutionContext.js'
import { bindingRefreshRequested } from './perContainerHeartbeatBus.js'

/**
 * ServerConfig 云运行状态、内容/历史拉取、跳转 URL 与停止服务器。
 */
export function useServerConfigRuntime({
  props,
  route,
  installedImages,
  activeServerSection,
  isRelayToTraeEnabled,
  relayToTraeDefaultAppliedGetter,
}) {
  const serverRuntimeStatus = ref('')
  const serverRuntimeStatusMessage = ref('')
  const serverRuntimeStatusTraceId = ref('')
  const isServerRuntimeStatusLoading = ref(false)
  const isWorkbenchLinkLoading = ref(false)
  const serverRuntimeStatusResponse = ref(null)
  const isServerStartHistoryLoading = ref(false)
  const serverStartHistoryRecords = ref([])
  const serverStartHistoryMessage = ref('')
  const serverStartHistoryMessageTraceId = ref('')
  const runtimeUptimeTick = ref(0)
  let runtimeUptimeIntervalId = null
  const scopedComment = createScopedCommentIdMemory()
  const runtimeSnapshots = createCommentRuntimeSnapshotStore()
  const contentSnapshots = createCommentContentSnapshotStore()

  const serverRuntimeStatusDisplayText = computed(() => {
    if (!serverRuntimeStatus.value) {
      if (
        isServerRuntimeStartFailedMessage(
          serverRuntimeStatusMessage.value,
          serverRuntimeStatusResponse.value,
        )
      ) {
        return '启动失败'
      }
      if (isServerRuntimeNoInstanceMessage(
        serverRuntimeStatusMessage.value,
        serverRuntimeStatusResponse.value,
      )) {
        return '未创建'
      }
      return '未知'
    }
    const statusMap = {
      Running: '运行中',
      Stopped: '已停止',
      Starting: '启动中',
      Stopping: '停止中',
      Rebooting: '重启中',
      Pending: '创建中',
      Initializing: '初始化中',
      Released: '已释放',
      Failed: '启动失败',
    }
    return statusMap[serverRuntimeStatus.value] || serverRuntimeStatus.value
  })

  const showRuntimeActionButtons = computed(() => serverRuntimeStatus.value === 'Running')
  const isServerRuntimeRunning = computed(() => serverRuntimeStatus.value === 'Running')

  const applyDefaultServerTabByRuntime = () => {
    const relayDefaultApplied = relayToTraeDefaultAppliedGetter?.()
    if (isRelayToTraeEnabled?.value) {
      if (!relayDefaultApplied?.value) {
        activeServerSection.value = 'relayDirect'
        if (relayDefaultApplied) {
          relayDefaultApplied.value = true
        }
      }
      return
    }
    // 硬件已并入镜像卡；服务器信息区默认停在「运行状态」提示 Tab
    activeServerSection.value = 'runtime'
  }

  const runtimePublicIp = computed(() => {
    const ips = serverRuntimeStatusResponse.value?.instance_attribute?.body?.PublicIpAddress?.IpAddress
    if (!Array.isArray(ips) || ips.length === 0) {
      return ''
    }
    return String(ips[0] || '').trim()
  })

  const serverJumpUrl = computed(() =>
    buildJumpToServerUrl(props.serverUrl, runtimePublicIp.value, serverJumpDefaultPort),
  )

  const effectiveContainerVscodeUrl = computed(() => {
    if (props.containerHeartbeatStatus !== 'connected') return ''
    const pip = runtimePublicIp.value
    const fromApi = String(props.containerVscodeUrl || '').trim()
    if (fromApi) {
      let resolved = fromApi
      if (pip) {
        const rewritten = replaceHttpUrlHostname(fromApi, pip)
        if (rewritten) {
          resolved = rewritten
        }
      }
      if (isLoopbackHttpUrl(resolved)) return ''
      return resolved
    }
    if (!showRuntimeActionButtons.value) return ''
    const fallback = buildJumpToServerUrl(
      props.serverUrl,
      pip,
      serverVscodeDefaultPort,
    )
    if (!fallback) return ''
    if (isLoopbackHttpUrl(fallback)) return ''
    return fallback
  })

  const aliyunEcsInstanceConsoleUrl = computed(() => {
    const data = serverRuntimeStatusResponse.value
    if (!data) {
      return ''
    }
    const platform = String(data.platform || '').toLowerCase()
    if (platform !== 'aliyun') {
      return ''
    }
    const instanceBody = data.instance_attribute?.body || {}
    const instanceId = String(data.instance_id || instanceBody.InstanceId || '').trim()
    const regionId = String(data.region || '').trim()
    if (!instanceId || !regionId) {
      return ''
    }
    const idSeg = encodeURIComponent(instanceId)
    const regionSeg = encodeURIComponent(regionId)
    return `https://ecs.console.aliyun.com/server/${idSeg}/detail?regionId=${regionSeg}`
  })

  const runningContainerImageDisplayForRuntimeDetails = computed(() => {
    const cid = props.task?.container_image_id ?? props.task?.container_image?.id
    if (!cid) {
      return ''
    }
    const ci = props.task?.container_image || {}
    const installed = installedImages.value.find((i) => String(i.id) === String(cid))
    const name = String(ci.name || installed?.name || '').trim()
    const url = String(ci.image_url || installed?.image_url || '').trim()
    const verRaw = ci.version ?? installed?.version
    const ver = verRaw != null && String(verRaw).trim() !== '' ? String(verRaw).trim() : ''
    let title = name
    if (name && ver) {
      title = `${name} (${ver})`
    } else if (!name && ver) {
      title = ver
    }
    if (title && url) {
      return `${title} — ${url}`
    }
    if (url) {
      return url
    }
    if (title) {
      return title
    }
    return String(cid)
  })

  const serverRuntimeStatusDetails = computed(() => {
    const tick = runtimeUptimeTick.value
    return buildServerRuntimeStatusDetails(
      serverRuntimeStatusResponse.value,
      tick,
      runningContainerImageDisplayForRuntimeDetails.value,
    )
  })

  const serverRuntimeStatusRawJson = computed(() => {
    if (!serverRuntimeStatusResponse.value) {
      return ''
    }
    try {
      return JSON.stringify(serverRuntimeStatusResponse.value, null, 2)
    } catch (error) {
      console.error('格式化服务器运行状态响应失败:', error)
      return ''
    }
  })

  const resolveRuntimeApiWorkspaceId = () =>
    resolveTaskWorkspaceId({
      route,
      task: props.task,
      workspaceId: props.workspaceId,
    })

  const fetchServerRuntimeStatus = async (commentId) => {
    const cid = scopedComment.fromAction(commentId)
    const slotRefs = runtimeSnapshots.refsFor(cid)
    await fetchServerRuntimeStatusApi({
      props,
      route,
      resolveRuntimeApiWorkspaceId,
      commentId: cid,
      serverRuntimeStatus: slotRefs.serverRuntimeStatus,
      serverRuntimeStatusMessage: slotRefs.serverRuntimeStatusMessage,
      serverRuntimeStatusTraceId: slotRefs.serverRuntimeStatusTraceId,
      serverRuntimeStatusResponse: slotRefs.serverRuntimeStatusResponse,
      isServerRuntimeStatusLoading: slotRefs.isServerRuntimeStatusLoading,
      applyDefaultServerTabByRuntime,
    })
    const snap = runtimeSnapshots.get(cid)
    serverRuntimeStatus.value = snap.status
    serverRuntimeStatusMessage.value = snap.message
    serverRuntimeStatusTraceId.value = snap.traceId
    serverRuntimeStatusResponse.value = snap.response
    isServerRuntimeStatusLoading.value = snap.loading
    // OPT-20260822-062: 云实例 Released/Terminated 时 binding 可能仍为 running，
    // 请求重拉绑定列表，使滞后的「容器 运行中」尽快对齐服务端终态。
    if (isReleasedRuntimeStatus(snap.status)) {
      bindingRefreshRequested.value = { commentId: cid, ts: Date.now() }
    }
  }

  watch(
    [() => props.isServerRunning, serverRuntimeStatus, serverRuntimeStatusMessage],
    ([running, rs, msg]) => {
      const runtimeAbsent = !rs && isServerRuntimeNoInstanceMessage(msg, serverRuntimeStatusResponse.value)
      if (!shouldRematchRuntimeHydrate(running, rs, { runtimeAbsent })) return
      if (runtimeAbsent) {
        notifyRuntimeAbsentHydrate(props.updateServerStatus)
        return
      }
      notifyRuntimeHydrate(props.updateServerStatus, rs)
    },
  )

  const fetchServerContent = async (commentId) => {
    const cid = scopedComment.fromAction(commentId)
    const slotRefs = contentSnapshots.refsFor(cid)
    await fetchServerContentApi({
      props,
      route,
      resolveRuntimeApiWorkspaceId,
      serverContentMessage: slotRefs.serverContentMessage,
      serverContentMessageTraceId: slotRefs.serverContentMessageTraceId,
      serverContentText: slotRefs.serverContentText,
      serverContentTargetUrl: slotRefs.serverContentTargetUrl,
      isServerContentLoading: slotRefs.isServerContentLoading,
      commentId: cid,
    })
  }

  const fetchServerStartHistory = () => fetchServerStartHistoryApi({
    props,
    route,
    resolveRuntimeApiWorkspaceId,
    serverStartHistoryMessage,
    serverStartHistoryMessageTraceId,
    serverStartHistoryRecords,
    isServerStartHistoryLoading,
  })

  const openWorkbenchLink = (commentId) => openWorkbenchLinkApi({
    props,
    route,
    resolveRuntimeApiWorkspaceId,
    serverRuntimeStatusMessage,
    serverRuntimeStatusTraceId,
    isWorkbenchLinkLoading,
    commentId: commentIdFromButtonArg(commentId),
  })

  const peekScopedCommentId = () => scopedComment.forPoll()

  const stopServer = (commentId) => stopServerApi({
    props,
    route,
    resolveRuntimeApiWorkspaceId,
    fetchServerRuntimeStatus: () => fetchServerRuntimeStatus(scopedComment.forPoll()),
    fetchServerStartHistory,
    commentId: scopedComment.fromAction(commentId),
  })

  const resetRuntimeForNewTask = () => {
    scopedComment.reset()
    runtimeSnapshots.clear()
    contentSnapshots.clear()
    serverRuntimeStatus.value = ''
    serverRuntimeStatusMessage.value = ''
    serverRuntimeStatusTraceId.value = ''
    serverRuntimeStatusResponse.value = null
    serverStartHistoryRecords.value = []
    serverStartHistoryMessage.value = ''
    serverStartHistoryMessageTraceId.value = ''
  }

  const watchStatusMessageForRefresh = (message, eventCommentId, runtimeStatus) => {
    const cid = commentIdForRuntimeRefresh({ message, eventCommentId })
    if (!cid) {
      if (shouldRefreshRuntimeOnStatusMessage(message)) {
        console.warn('[runtime] skip snapshot apply: start/stop success without event comment_id')
      }
      return
    }
    const applied = applyPushedRuntimeSnapshot(runtimeSnapshots, {
      commentId: cid,
      runtimeStatus,
      message,
    })
    if (applied) {
      const snap = runtimeSnapshots.get(cid)
      serverRuntimeStatus.value = snap.status
      serverRuntimeStatusMessage.value = snap.message
      serverRuntimeStatusTraceId.value = snap.traceId
    }
  }

  watch(
    () => resolveRuntimeUptimeSource(serverRuntimeStatusResponse.value),
    (uptimeSource) => {
      if (runtimeUptimeIntervalId !== null) {
        clearInterval(runtimeUptimeIntervalId)
        runtimeUptimeIntervalId = null
      }
      if (uptimeSource) {
        runtimeUptimeTick.value = Date.now()
        runtimeUptimeIntervalId = window.setInterval(() => {
          runtimeUptimeTick.value = Date.now()
        }, 30000)
      }
    },
    { immediate: true },
  )

  const disposeRuntimeTimers = () => {
    if (runtimeUptimeIntervalId !== null) {
      clearInterval(runtimeUptimeIntervalId)
      runtimeUptimeIntervalId = null
    }
  }

  /**
   * 「服务器运行状态」展示面板数据源（透传给评论区「执行细节」Tab）。
   * 由 ServerConfig.logic.vue defineExpose 暴露，供 TaskDetail 桥接到评论区。
   * 评论列表顶端不再放全局镜像运行 header；本面板即评论内唯一运行态入口。
   */
  const serverRuntimeStatusPanel = computed(() => {
    const slots = runtimeSnapshots.state.value
    return {
      showRuntimeActionButtons: showRuntimeActionButtons.value,
      isWorkbenchLinkLoading: isWorkbenchLinkLoading.value,
      effectiveContainerVscodeUrl: effectiveContainerVscodeUrl.value,
      isServerRuntimeStatusLoading: isServerRuntimeStatusLoading.value,
      serverRuntimeStatusDisplayText: serverRuntimeStatusDisplayText.value,
      serverJumpUrl: serverJumpUrl.value,
      serverJumpDefaultPort,
      serverRuntimeStatusMessage: serverRuntimeStatusMessage.value,
      serverRuntimeStatusTraceId: serverRuntimeStatusTraceId.value,
      serverRuntimeStatusDetails: serverRuntimeStatusDetails.value,
      serverRuntimeStatusRawJson: serverRuntimeStatusRawJson.value,
      snapshotForComment: (commentId, snapOpts = {}) => buildCommentRuntimePanelDisplay(
        slots[String(commentId || '').trim()] || {},
        {
          tick: runtimeUptimeTick.value,
          imageDisplay: runningContainerImageDisplayForRuntimeDetails.value,
          serverUrl: props.serverUrl,
          commentCreatedAt: snapOpts.commentCreatedAt,
        },
      ),
      openWorkbenchLink,
      fetchServerRuntimeStatus,
      stopServer,
    }
  })

  const serverContentPanel = computed(() => {
    const slots = contentSnapshots.state.value
    return {
      snapshotForComment: (commentId) => {
        const s = slots[String(commentId || '').trim()] || {
          loading: false,
          message: '',
          messageTraceId: '',
          targetUrl: '',
          text: '',
        }
        return {
          isServerContentLoading: s.loading,
          serverContentMessage: s.message,
          serverContentMessageTraceId: s.messageTraceId,
          serverContentTargetUrl: s.targetUrl,
          serverContentText: s.text,
        }
      },
      fetchServerContent,
    }
  })

  return {
    serverRuntimeStatus,
    serverRuntimeStatusMessage,
    serverRuntimeStatusTraceId,
    isServerRuntimeStatusLoading,
    isWorkbenchLinkLoading,
    serverRuntimeStatusResponse,
    isServerStartHistoryLoading,
    serverStartHistoryRecords,
    serverStartHistoryMessage,
    serverStartHistoryMessageTraceId,
    serverRuntimeStatusDisplayText,
    showRuntimeActionButtons,
    isServerRuntimeRunning,
    runtimePublicIp,
    serverJumpUrl,
    serverJumpDefaultPort,
    effectiveContainerVscodeUrl,
    aliyunEcsInstanceConsoleUrl,
    serverRuntimeStatusDetails,
    serverRuntimeStatusRawJson,
    fetchServerRuntimeStatus,
    fetchServerContent,
    fetchServerStartHistory,
    openWorkbenchLink,
    stopServer,
    peekScopedCommentId,
    serverRuntimeStatusPanel,
    serverContentPanel,
    resetRuntimeForNewTask,
    watchStatusMessageForRefresh,
    disposeRuntimeTimers,
  }
}
