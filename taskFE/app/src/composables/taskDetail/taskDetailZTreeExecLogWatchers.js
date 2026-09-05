import { watch } from 'vue'

/**
 * 执行日志 ZTree 状态 watcher（OPT-20260815-027 拆分）。
 * 节点切换 → 重置刷新 + poller sync；端点/目标变化 → poller sync；快照更新 → 预取。
 * @param {object} zlog createTaskDetailZTreeExecLogState 返回值
 * @param {object} deps 外层 useTaskDetail deps
 */
export function installTaskDetailZTreeExecLogWatchers(zlog, deps) {
  const {
    selectedLayerGraphNodeLogKey,
    refreshZTreeExecutionLog,
    activeJobExecLogPoller,
    zTreeLogTargets,
    prefetchLayerChangeSummariesForDirtyLayers,
  } = zlog
  const { layerGraphSnapshot, containerEndpointRegistered, containerHttpUnreachable } = deps
  const released = () => Boolean((deps.containerReleased?.value ?? deps.containerReleased) || zlog.containerReleased?.value)

  watch(selectedLayerGraphNodeLogKey, (key, prev) => {
    if (!key && !prev) return
    // 已释放仍拉 SaaS/COS step_full；refresh 内跳过 clone-log。
    void refreshZTreeExecutionLog({ reset: true })
    activeJobExecLogPoller.sync()
  })

  watch(
    () => [
      containerEndpointRegistered.value,
      containerHttpUnreachable?.value,
      zTreeLogTargets.value.jobId,
      zTreeLogTargets.value.layerId,
    ],
    (curr, prev) => {
      const jobId = curr?.[2]
      const layerId = curr?.[3]
      const jobChanged = !prev || prev[2] !== jobId
      const layerChanged = !prev || prev[3] !== layerId
      const becameReady = Boolean(curr?.[0]) && !(prev && prev[0])
      // 历史步骤在 SaaS DB / COS：不要求容器端点已登记；释放后同样 hydrate。
      if ((jobId || layerId) && (jobChanged || layerChanged || becameReady || !prev)) {
        void refreshZTreeExecutionLog({ reset: true })
      }
      activeJobExecLogPoller.sync()
    },
  )

  watch(
    layerGraphSnapshot,
    () => {
      if (!released()) {
        void prefetchLayerChangeSummariesForDirtyLayers()
      }
      activeJobExecLogPoller.sync()
    },
    { deep: true },
  )
}
