<template>
  <div class="bg-white rounded-xl shadow-md p-6 mb-6" data-alias="BillingTransactionsFilters">
    <h2 class="text-lg font-semibold mb-4">过滤条件</h2>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="relative workspace-filter">
        <label class="block text-sm font-medium text-gray-700 mb-2">工作空间名称</label>
        <HeaderSearchFilter
          :search="workspaceSearch"
          :options="workspaceOptions"
          :show-dropdown="showWorkspaceDropdown"
          :selected-name="selectedWorkspaceName"
          placeholder="输入工作空间名称或缩略词"
          data-alias="BillingTransactionsWorkspaceSearch"
          wrapper-class="workspace-filter"
          selected-class="mt-1 text-sm text-gray-500"
          @update:search="v => $emit('update:workspaceSearch', v)"
          @search="$emit('search-workspaces')"
          @focus="$emit('focus-workspace')"
          @select="o => $emit('select-workspace', o)"
        />
      </div>

      <div class="relative task-filter">
        <label class="block text-sm font-medium text-gray-700 mb-2">任务名称</label>
        <HeaderSearchFilter
          :search="taskSearch"
          :options="taskOptions"
          :show-dropdown="showTaskDropdown"
          :selected-name="selectedTaskName"
          placeholder="输入任务名称搜索"
          data-alias="BillingTransactionsTaskSearch"
          wrapper-class="task-filter"
          selected-class="mt-1 text-sm text-gray-500"
          @update:search="v => $emit('update:taskSearch', v)"
          @search="$emit('search-tasks')"
          @focus="$emit('focus-task')"
          @select="o => $emit('select-task', o)"
        />
      </div>
    </div>

    <div class="mt-4 flex space-x-2">
      <button
        class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        @click="$emit('apply')"
      >
        应用过滤
      </button>
      <button
        class="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        @click="$emit('reset')"
      >
        重置
      </button>
    </div>
  </div>
</template>

<script setup>
import HeaderSearchFilter from './HeaderSearchFilter.vue'

defineProps({
  filters: { type: Object, required: true },
  workspaceSearch: { type: String, default: '' },
  taskSearch: { type: String, default: '' },
  workspaceOptions: { type: Array, default: () => [] },
  taskOptions: { type: Array, default: () => [] },
  showWorkspaceDropdown: { type: Boolean, default: false },
  showTaskDropdown: { type: Boolean, default: false },
  selectedWorkspaceName: { type: String, default: '' },
  selectedTaskName: { type: String, default: '' }
})

defineEmits([
  'update:filter',
  'update:workspaceSearch',
  'update:taskSearch',
  'search-workspaces',
  'search-tasks',
  'focus-workspace',
  'focus-task',
  'select-workspace',
  'select-task',
  'apply',
  'reset'
])
</script>
