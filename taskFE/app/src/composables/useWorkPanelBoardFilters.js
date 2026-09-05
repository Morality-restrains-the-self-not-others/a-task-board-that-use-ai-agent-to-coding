import { computed } from 'vue'
import { useWorkPanelMachineSummary } from './useWorkPanelMachineSummary.js'
import { useWorkPanelAccessFilter } from './useWorkPanelAccessFilter.js'
import { filterTodosByAccess } from '../utils/workPanelAccessFilter.js'
import { machineRuntimeFilterMismatch } from '../utils/workPanelMachineRuntimeFilter.js'
import { defaultDeliverableFilterBars } from '../utils/workPanelDeliverableFilterBars.js'

/**
 * 工作面板看板过滤：机器运行态 ∩ 人/小组访问（AND）。
 * @param {{
 *   apiFetch: typeof fetch,
 *   tenantId: import('vue').Ref,
 *   currentWorkspace: import('vue').Ref,
 *   todos: import('vue').Ref,
 *   todosError?: import('vue').Ref,
 *   todosErrorTraceId?: import('vue').Ref<string>,
 *   collaborators: import('vue').Ref,
 *   workspaceRefreshTrigger: import('vue').Ref<number>,
 *   showEmptyTaskHint: import('vue').ComputedRef<boolean>,
 *   deliverableFilterBars?: import('vue').Ref<unknown[]>,
 * }} opts
 */
export function useWorkPanelBoardFilters({
  apiFetch,
  tenantId,
  currentWorkspace,
  todos,
  todosError,
  todosErrorTraceId,
  collaborators,
  workspaceRefreshTrigger,
  showEmptyTaskHint,
  deliverableFilterBars,
}) {
  const machine = useWorkPanelMachineSummary({
    apiFetch,
    tenantId,
    currentWorkspace,
    todos,
  })
  const access = useWorkPanelAccessFilter({
    apiFetch,
    tenantId,
    currentWorkspace,
    collaborators,
  })

  const filteredTodos = computed(() =>
    filterTodosByAccess(machine.filteredTodos.value, access.accessFilter.value),
  )

  /**
   * 机器运行态过滤生效时，暂时用根路径过滤栏，避免「runIt」等内容过滤
   * 把已启动任务藏进空分区（runIt=0 且用户以为没有任务）。
   */
  const boardDeliverableFilterBars = computed(() => {
    if (machine.machineRuntimeFilter.value) {
      return defaultDeliverableFilterBars()
    }
    const bars = deliverableFilterBars?.value
    return Array.isArray(bars) ? bars : defaultDeliverableFilterBars()
  })

  // 仅按机器运行态过滤（不含访问过滤），确保 mismatch 归因精确：
  // 访问过滤造成的空列表不应归因为"机器过滤不匹配"
  const machineOnlyFilteredTodos = computed(() =>
    Array.isArray(machine.filteredTodos.value) ? machine.filteredTodos.value : [],
  )

  const machineFilterMismatch = computed(() =>
    machineRuntimeFilterMismatch({
      filter: machine.machineRuntimeFilter.value,
      filteredCount: machineOnlyFilteredTodos.value.length,
      todosCount: Array.isArray(todos.value) ? todos.value.length : 0,
      indicatorsReady: machine.runtimeIndicatorsReady.value,
      startedCount: machine.machineSummary.value?.startedCount,
      idleCount: machine.machineSummary.value?.idleCount,
    }),
  )

  const headerBind = computed(() => ({
    tenantId: tenantId.value,
    refreshTrigger: workspaceRefreshTrigger.value,
    showEmptyTaskHint: showEmptyTaskHint.value,
    todosLoadError: todosError?.value || '',
    todosLoadErrorTraceId: todosErrorTraceId?.value || '',
    machineSummary: machine.machineSummary.value,
    machineSummaryLabel: machine.machineSummaryLabel.value,
    machineSummaryErrorTraceId: machine.machineSummaryErrorTraceId.value,
    machineSummaryStaleSince: machine.machineSummaryStaleSince.value,
    machineRuntimeFilter: machine.machineRuntimeFilter.value,
    machineFilterMismatch: machineFilterMismatch.value,
    accessFilter: access.accessFilter.value,
    accessPeople: access.accessPeople.value,
    accessGroups: access.accessGroups.value,
    accessSubjectsLoading: access.accessSubjectsLoading.value,
    accessFilterPanelOpen: access.accessFilterPanelOpen.value,
    accessFilterTab: access.accessFilterTab.value,
  }))

  const resetBoardFilters = () => {
    machine.resetMachineRuntimeFilter()
    access.resetAccessFilter()
  }

  const refreshBoardFilterData = async () => {
    await machine.refreshMachineSummary()
    await access.refreshAccessSubjects()
  }

  return {
    runtimeIndicators: machine.runtimeIndicators,
    runtimeIndicatorsReady: machine.runtimeIndicatorsReady,
    machineRuntimeFilter: machine.machineRuntimeFilter,
    machineSummary: machine.machineSummary,
    filteredTodos,
    boardDeliverableFilterBars,
    machineFilterMismatch,
    headerBind,
    accessFilter: access.accessFilter,
    accessPeople: access.accessPeople,
    accessGroups: access.accessGroups,
    accessSubjectsLoading: access.accessSubjectsLoading,
    accessFilterTab: access.accessFilterTab,
    handleMachineRuntimeFilter: machine.handleMachineRuntimeFilter,
    clearMachineRuntimeFilter: machine.clearMachineRuntimeFilter,
    startMachineSummaryPolling: machine.startMachineSummaryPolling,
    selectAccessSubject: access.selectAccessSubject,
    hydrateAccessFilterPref: access.hydrateAccessFilterPref,
    clearAccessFilter: access.clearAccessFilter,
    toggleAccessFilterPanel: access.toggleAccessFilterPanel,
    closeAccessFilterPanel: access.closeAccessFilterPanel,
    setAccessFilterTab: access.setAccessFilterTab,
    resetBoardFilters,
    refreshBoardFilterData,
  }
}
