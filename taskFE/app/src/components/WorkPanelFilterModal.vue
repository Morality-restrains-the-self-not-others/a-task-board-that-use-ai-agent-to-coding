<template>
  <div
    v-if="show"
    id="filter-modal"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999"
  >
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6 z-10000 relative" @keydown.enter="$emit('apply')">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-xl font-bold text-gray-900">筛选交付物</h3>
        <button class="text-gray-500 hover:text-gray-700" @click="$emit('close')">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      <div class="space-y-4">
        <div>
          <label for="filter-search" class="block text-sm font-medium text-gray-700">搜索</label>
          <input
            id="filter-search"
            type="text"
            class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
            placeholder="搜索交付物标题或描述"
            :value="filterOptions.search"
            @input="$emit('update:search', $event.target.value)"
          >
        </div>
        <div>
          <label for="filter-priority" class="block text-sm font-medium text-gray-700">优先级</label>
          <select
            id="filter-priority"
            class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
            :value="filterOptions.priority"
            @change="$emit('update:priority', $event.target.value === 'null' || $event.target.value === '' ? null : Number($event.target.value))"
          >
            <option :value="null">全部</option>
            <option :value="0">高优先级</option>
            <option :value="1">中优先级</option>
            <option :value="2">低优先级</option>
          </select>
        </div>
        <WorkPanelFilterModalAccess
          :access-filter="accessFilter"
          :people="accessPeople"
          :groups="accessGroups"
          :loading="accessSubjectsLoading"
          :tab="accessFilterTab"
          @update:tab="$emit('access-filter-tab', $event)"
          @select="$emit('access-filter-select', $event)"
          @clear="$emit('clear-access-filter')"
        />
      </div>
      <div class="mt-6 flex justify-end space-x-3">
        <button
          type="button"
          class="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
          @click="$emit('reset')"
        >
          重置
        </button>
        <button
          type="button"
          class="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
          @click="$emit('apply')"
        >
          应用筛选
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import WorkPanelFilterModalAccess from './WorkPanelFilterModalAccess.vue'

defineProps({
  show: { type: Boolean, default: false },
  filterOptions: { type: Object, required: true },
  accessFilter: { type: Object, default: null },
  accessPeople: { type: Array, default: () => [] },
  accessGroups: { type: Array, default: () => [] },
  accessSubjectsLoading: { type: Boolean, default: false },
  accessFilterTab: { type: String, default: 'people' },
})

defineEmits([
  'close',
  'reset',
  'apply',
  'update:search',
  'update:priority',
  'access-filter-tab',
  'access-filter-select',
  'clear-access-filter',
])
</script>
