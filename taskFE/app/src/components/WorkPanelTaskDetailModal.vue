<template>
  <div
    v-if="show"
    id="task-detail-modal"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999"
  >
    <div class="bg-white rounded-lg shadow-xl w-full max-w-4xl h-full max-h-[90vh] z-10000 relative flex flex-col">
      <div class="flex justify-between items-center mb-0 p-6 border-b border-gray-200">
        <h3 class="text-xl font-bold text-gray-900">交付物详情</h3>
        <div class="flex items-center space-x-2">
          <a
            id="open-task-in-new-tab"
            :href="newTabHref"
            target="_blank"
            rel="noopener noreferrer"
            class="text-gray-500 hover:text-gray-700"
            title="在新页面打开（保留 accessCode 等参数）"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </a>
          <button id="maximize-task-detail-modal" class="text-gray-500 hover:text-gray-700" title="最大化">
            <svg class="w-5 h-5 max-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="display: block;">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" fill="none" />
            </svg>
            <svg class="w-5 h-5 restore-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="display: none;">
              <rect x="13" y="3" width="8" height="8" rx="1" ry="1" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" fill="none" />
              <rect x="3" y="13" width="8" height="8" rx="1" ry="1" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" fill="none" />
            </svg>
          </button>
          <button id="close-task-detail-modal" class="text-gray-500 hover:text-gray-700" @click="$emit('close')">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>
      <div id="task-detail-content" class="flex-grow overflow-y-auto">
        <TaskDetailContent
          :task="task"
          :tenant-id="tenantId"
          :workspace-id="workspaceId"
          :task-id="task?.id"
          data-vue-component="TaskDetailContent"
          @close="$emit('close')"
          @task-updated="$emit('task-updated')"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import TaskDetailContent from './TaskDetailContent.logic.vue'

defineProps({
  show: { type: Boolean, default: false },
  task: { type: Object, default: null },
  tenantId: { type: [String, Number], default: null },
  workspaceId: { type: String, default: 'default' },
  newTabHref: { type: String, default: '#' },
})

defineEmits(['close', 'task-updated'])
</script>
