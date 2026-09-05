import { ref, computed, onUnmounted } from 'vue'
import {
  fetchWorkspaceMachineSummary,
  formatWorkspaceMachineSummaryLabel,
} from '../utils/workPanelMachineSummary.js'
import { fetchWorkspaceRuntimeIndicators } from '../utils/workPanelRuntimeIndicators.js'
import {
  filterTodosByMachineRuntime,
  toggleMachineRuntimeFilter,
} from '../utils/workPanelMachineRuntimeFilter.js'
import { warnNetworkFailure } from '../utils/workPanelApiUtils.js'

/** 连续刷新失败超过该轮数即标记数据陈旧（无后台 interval；由 SSE/可见性/工作区切换驱动） */
const MACHINE_SUMMARY_STALE_AFTER_FAILURES = 3

/**
 * OPT-20260808-021: 轮询结果与当前 ref 值做内容比对，内容未变时不替换 ref
 * （保持对象身份稳定），避免 15s 轮询周期性地引发看板全量重渲染。
 * @param {{ startedCount?: unknown, startingCount?: unknown, idleCount?: unknown, busyCount?: unknown, idleRecycleMinutes?: unknown } | null | undefined} a
 * @param {{ startedCount?: unknown, startingCount?: unknown, idleCount?: unknown, busyCount?: unknown, idleRecycleMinutes?: unknown } | null | undefined} b
 */
function sameMachineSummary(a, b) {
  if (a === b) return true
  if (!a || !b) return false
  return (
    a.startedCount === b.startedCount &&
    a.startingCount === b.startingCount &&
    a.idleCount === b.idleCount &&
    a.busyCount === b.busyCount &&
    a.idleRecycleMinutes === b.idleRecycleMinutes
  )
}

/**
 * 逐 taskId 比对运行态 map（machineRunning / containerRunning）。
 * @param {Record<string, { machineRunning?: boolean, containerRunning?: boolean }> | null | undefined} a
 * @param {Record<string, { machineRunning?: boolean, containerRunning?: boolean }> | null | undefined} b
 */
function sameRuntimeIndicators(a, b) {
  if (a === b) return true
  if (!a || !b) return false
  const keysA = Object.keys(a)
  const keysB = Object.keys(b)
  if (keysA.length !== keysB.length) return false
  for (const k of keysA) {
    const x = a[k]
    const y = b[k]
    if (x === y) continue
    if (!x || !y) return false
    if (x.machineRunning !== y.machineRunning) return false
    if (x.containerRunning !== y.containerRunning) return false
  }
  return true
}

/**
 * @param {{
 *   apiFetch: typeof fetch,
 *   tenantId: import('vue').Ref<string|null>,
 *   currentWorkspace: import('vue').Ref<{id?: string|null}|null>,
 *   todos: import('vue').Ref<unknown[]>,
 * }} opts
 */
export function useWorkPanelMachineSummary({ apiFetch, tenantId, currentWorkspace, todos }) {
  const machineSummary = ref(null)
  const machineSummaryLabel = computed(() => formatWorkspaceMachineSummaryLabel(machineSummary.value))
  const runtimeIndicators = ref(Object.create(null))
  /** 至少成功/失败完成过一次 indicators 拉取后为 true */
  const runtimeIndicatorsReady = ref(false)
  /**
   * OPT-20260809-012: 摘要/指示器轮询失败请求的 traceId（header-summary-row data-traceId 数据源）。
   * 任一侧失败即记录该侧 traceId，下一次成对成功时清除（对齐全站「错误元素带 data-traceId」约定）。
   */
  const machineSummaryErrorTraceId = ref('')
  /**
   * OPT-20260809-029: 最近一次成对成功的时间戳（null = 尚未成功过）。
   * 用于判断「存在可陈旧的数据」：从未成功过时不标记陈旧（无可陈旧快照）。
   */
  const machineSummaryLastSuccessAt = ref(null)
  /**
   * OPT-20260809-029: 数据开始陈旧的时间戳（null = 当前快照新鲜）。
   * 任一侧轮询连续失败超过阈值时置位，成对成功恢复即清除（对齐全站「陈旧提示可恢复」约定）。
   */
  const machineSummaryStaleSince = ref(null)
  /** 连续失败轮数（任一侧失败即计一轮；成对成功清零） */
  let consecutiveFailureCount = 0
  /** @type {import('vue').Ref<'started' | 'idle' | null>} */
  const machineRuntimeFilter = ref(null)
  const filteredTodos = computed(() =>
    filterTodosByMachineRuntime(todos.value, runtimeIndicators.value, machineRuntimeFilter.value, {
      indicatorsReady: runtimeIndicatorsReady.value,
    }),
  )
  let detachVisibilityListener = null

  const refreshMachineSummary = async () => {
    const tid = tenantId.value
    const wid = currentWorkspace.value?.id
    if (!tid || !wid || wid === 'default') {
      machineSummary.value = null
      runtimeIndicators.value = Object.create(null)
      runtimeIndicatorsReady.value = false
      machineSummaryLastSuccessAt.value = null
      machineSummaryStaleSince.value = null
      consecutiveFailureCount = 0
      return
    }
    // 摘要与卡片指示器成对拉取：任一侧失败都不单侧应用（保留上一对一致快照），
    // 避免头部「已启动 N」与卡片运行态指示在瞬时故障下展示不同数据源状态。
    const [summaryRes, indicatorsRes] = await Promise.allSettled([
      fetchWorkspaceMachineSummary({
        apiFetch,
        tenantId: tid,
        workspaceId: wid,
      }),
      fetchWorkspaceRuntimeIndicators({
        apiFetch,
        tenantId: tid,
        workspaceId: wid,
      }),
    ])
    if (summaryRes.status === 'rejected') {
      warnNetworkFailure('workspace-machine-summary', summaryRes.reason)
      // 优先记录 summary 侧 traceId；空时回退 indicators 侧
      if (!machineSummaryErrorTraceId.value) {
        machineSummaryErrorTraceId.value = summaryRes.reason?.traceId || ''
      }
    }
    if (indicatorsRes.status === 'rejected') {
      warnNetworkFailure('workspace-runtime-indicators', indicatorsRes.reason)
      if (!machineSummaryErrorTraceId.value) {
        machineSummaryErrorTraceId.value = indicatorsRes.reason?.traceId || ''
      }
    }
    // 与旧 finally 语义一致：完成过一次拉取尝试即视为 ready（内容保持上一对快照），
    // 供 machineRuntimeFilter 基于最近已知快照过滤。
    runtimeIndicatorsReady.value = true
    if (summaryRes.status !== 'fulfilled' || indicatorsRes.status !== 'fulfilled') {
      // OPT-20260809-029: 本轮失败（保留上一对一致快照）。连续失败超过阈值且此前有过
      // 新鲜快照时标记陈旧；成对成功恢复即清除。
      consecutiveFailureCount += 1
      if (
        machineSummaryLastSuccessAt.value !== null
        && consecutiveFailureCount >= MACHINE_SUMMARY_STALE_AFTER_FAILURES
        && machineSummaryStaleSince.value === null
      ) {
        machineSummaryStaleSince.value = Date.now()
      }
      return
    }
    // 成对成功：数据恢复新鲜
    machineSummaryLastSuccessAt.value = Date.now()
    consecutiveFailureCount = 0
    machineSummaryStaleSince.value = null
    const next = summaryRes.value
    const nextIndicators = indicatorsRes.value
    // 内容未变则不替换 ref，保持身份稳定避免无谓重渲染
    if (!sameMachineSummary(machineSummary.value, next)) {
      machineSummary.value = next
    }
    if (!sameRuntimeIndicators(runtimeIndicators.value, nextIndicators)) {
      runtimeIndicators.value = nextIndicators
    }
    runtimeIndicatorsReady.value = true
    // 成对成功：清除失败 traceId
    machineSummaryErrorTraceId.value = ''
  }

  const handleMachineRuntimeFilter = (kind) => {
    machineRuntimeFilter.value = toggleMachineRuntimeFilter(machineRuntimeFilter.value, kind)
  }

  const clearMachineRuntimeFilter = () => {
    machineRuntimeFilter.value = null
  }

  const resetMachineRuntimeFilter = clearMachineRuntimeFilter

  const stopMachineSummaryLiveUpdates = () => {
    detachVisibilityListener?.()
  }

  /**
   * OPT-20260827-029: 禁止 15s 后台 GET。只在回到前台时拉一次只读快照；
   * 运行中更新由 work-panel SSE（task_status_changed / 重建同步）驱动。
   */
  const startMachineSummaryPolling = () => {
    if (typeof document === 'undefined') return
    if (detachVisibilityListener) return
    const onVisibilityChange = () => {
      if (document.visibilityState === 'hidden') return
      void refreshMachineSummary()
    }
    document.addEventListener('visibilitychange', onVisibilityChange)
    detachVisibilityListener = () => {
      document.removeEventListener('visibilitychange', onVisibilityChange)
      detachVisibilityListener = null
    }
  }

  onUnmounted(() => {
    stopMachineSummaryLiveUpdates()
  })

  return {
    machineSummary,
    machineSummaryLabel,
    runtimeIndicators,
    runtimeIndicatorsReady,
    machineSummaryErrorTraceId,
    machineSummaryStaleSince,
    machineRuntimeFilter,
    filteredTodos,
    refreshMachineSummary,
    handleMachineRuntimeFilter,
    clearMachineRuntimeFilter,
    resetMachineRuntimeFilter,
    startMachineSummaryPolling,
  }
}
