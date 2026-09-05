<template>
  <TaskDetailView
    :task="task"
    :tenant-id="resolvedTenantId"
    :workspace-id="resolvedWorkspaceId"
    :task-id="resolvedTaskId"
    @close="emit('close')"
    @task-updated="(updated) => emit('task-updated', updated)"
  />
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import TaskDetailView from '../views/TaskDetail.vue'
import { resolveTaskRouteIds } from '../utils/resolveTaskRouteIds.js'

const props = defineProps({
  task: {
    type: Object,
    default: null
  },
  tenantId: {
    type: [String, Number],
    default: null
  },
  workspaceId: {
    type: [String, Number],
    default: null
  },
  taskId: {
    type: [String, Number],
    default: null
  }
})

const emit = defineEmits(['close', 'task-updated'])
const route = useRoute()

const routeIds = computed(() => resolveTaskRouteIds({
  tenantId: props.tenantId,
  workspaceId: props.workspaceId,
  taskId: props.taskId,
  task: props.task,
  routeParams: route.params,
}))
const resolvedTenantId = computed(() => routeIds.value.tenantId)
const resolvedWorkspaceId = computed(() => routeIds.value.workspaceId)
const resolvedTaskId = computed(() => routeIds.value.taskId)
</script>
