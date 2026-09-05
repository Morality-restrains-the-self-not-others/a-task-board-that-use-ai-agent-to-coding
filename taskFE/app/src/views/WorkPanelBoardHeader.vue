<template>
  <WorkPanelHeader
    :tenant-id="tenantId"
    :refresh-trigger="refreshTrigger"
    :show-empty-task-hint="showEmptyTaskHint"
    :machine-summary="machineSummary"
    :machine-summary-label="machineSummaryLabel"
    :machine-runtime-filter="machineRuntimeFilter"
    :access-filter="accessFilter"
    :access-people="accessPeople"
    :access-groups="accessGroups"
    :access-subjects-loading="accessSubjectsLoading"
    :access-filter-panel-open="accessFilterPanelOpen"
    :access-filter-tab="accessFilterTab"
    @create-task="$emit('create-task')"
    @filter="$emit('filter')"
    @workspace-switched="$emit('workspace-switched', $event)"
    @workspace-created="$emit('workspace-created', $event)"
    @machine-filter="handleMachineRuntimeFilter"
    @clear-machine-filter="clearMachineRuntimeFilter"
    @toggle-access-filter-panel="toggleAccessFilterPanel"
    @close-access-filter-panel="closeAccessFilterPanel"
    @access-filter-tab="accessFilterTab = $event"
    @access-filter-select="selectAccessSubject"
    @clear-access-filter="clearAccessFilter"
  />
</template>

<script setup>
import WorkPanelHeader from './WorkPanelHeader.vue'
import { useWorkPanelBoardFilters } from '../composables/useWorkPanelBoardFilters.js'

const props = defineProps({
  tenantId: { type: [String, Number], default: null },
  refreshTrigger: { type: Number, default: 0 },
  showEmptyTaskHint: { type: Boolean, default: false },
  apiFetch: { type: Function, required: true },
  currentWorkspace: { type: Object, default: null },
  todos: { type: Object, required: true },
  collaborators: { type: Object, required: true },
  tenantIdRef: { type: Object, required: true },
  currentWorkspaceRef: { type: Object, required: true },
})

defineEmits(['create-task', 'filter', 'workspace-switched', 'workspace-created'])

const {
  machineSummary,
  machineSummaryLabel,
  machineRuntimeFilter,
  runtimeIndicators,
  filteredTodos,
  handleMachineRuntimeFilter,
  clearMachineRuntimeFilter,
  startMachineSummaryPolling,
  accessFilter,
  accessPeople,
  accessGroups,
  accessSubjectsLoading,
  accessFilterPanelOpen,
  accessFilterTab,
  selectAccessSubject,
  clearAccessFilter,
  toggleAccessFilterPanel,
  closeAccessFilterPanel,
  resetBoardFilters,
  refreshBoardFilterData,
} = useWorkPanelBoardFilters({
  apiFetch: props.apiFetch,
  tenantId: props.tenantIdRef,
  currentWorkspace: props.currentWorkspaceRef,
  todos: props.todos,
  collaborators: props.collaborators,
})

defineExpose({
  filteredTodos,
  runtimeIndicators,
  resetBoardFilters,
  refreshBoardFilterData,
  startMachineSummaryPolling,
})
</script>
