<template>
  <!-- 纵轴=进度列（每列仅一进度）；横轴=交付物过滤上下文（由上层分区提供） -->
  <div
    class="flex space-x-6 pb-0 w-full overflow-x-auto h-full min-h-0 flex-1"
    :id="containerId"
    data-alias="deliverable-kanban-board"
  >
    <div
      v-for="status in orderedStatuses"
      :key="`${sectionKey}-progress-${status.id}`"
      class="progress-column task-column flex-shrink-0"
      data-alias="progress-column"
      :data-progress-column-id="String(status.id)"
      :data-section-key="sectionKey"
      :aria-label="status.name"
    >
      <div class="flex items-center gap-2 mb-2 px-0.5 min-w-0">
        <span
          class="inline-block w-2.5 h-2.5 rounded-full shrink-0"
          :style="{ backgroundColor: status.color || '#94a3b8' }"
          aria-hidden="true"
        />
        <span class="text-xs font-medium text-gray-700 truncate">{{ status.name }}</span>
        <span
          class="text-[10px] text-gray-400 tabular-nums ml-auto shrink-0"
          data-alias="progress-column-count"
        >
          {{ tasksInProgress(status.id).length }}
        </span>
      </div>

      <!-- 每个纵轴有且仅有一个进度列 -->
      <div
        class="progress-lane flex flex-col flex-grow min-h-0"
        data-alias="progress-lane"
        :data-progress-column-id="String(status.id)"
      >
        <div
          class="task-cards-container"
          data-alias="progress-task-cards"
          :data-progress-column-id="String(status.id)"
        >
          <TaskDetail
            v-for="todo in tasksInProgress(status.id)"
            :key="todo.id"
            :task="todo"
            :tenantId="tenantId"
            :workspaceId="workspaceId"
            :taskStatuses="taskStatuses"
            :runtimeIndicator="runtimeIndicators[String(todo.id)] || null"
            :collaborator-name-by-id="collaboratorNameById"
            @create-comment="(_taskId, payload) => $emit('create-comment', todo.id, payload)"
            @update-comment="(commentId, content) => $emit('update-comment', todo.id, commentId, content)"
            @progress-column-change="$emit('progress-column-change', $event)"
            @task-clicked="$emit('task-clicked', todo)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import TaskDetail from './TaskDetail.vue'
import { filterTodosByKanbanColumn } from '../utils/workPanelKanbanUtils.js'

const props = defineProps({
  sectionKey: { type: String, required: true },
  containerId: { type: String, default: '' },
  todos: { type: Array, default: () => [] },
  taskTypes: { type: Array, default: () => [] },
  taskStatuses: { type: Array, default: () => [] },
  tenantId: { type: [String, Number], default: null },
  workspaceId: { type: [String, Number], default: null },
  runtimeIndicators: { type: Object, default: () => ({}) },
  collaboratorNameById: { type: Object, default: () => ({}) },
})

defineEmits([
  'task-clicked',
  'create-comment',
  'update-comment',
  'progress-column-change',
])

const orderedStatuses = computed(() =>
  Array.isArray(props.taskStatuses) ? props.taskStatuses : [],
)

function tasksInProgress(statusId) {
  return filterTodosByKanbanColumn(props.todos, statusId, props.taskStatuses)
}
</script>

<style scoped src="../views/TaskPanel.css"></style>
