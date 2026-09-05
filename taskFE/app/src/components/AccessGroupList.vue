<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h5 class="font-medium">小组权限</h5>
      <button type="button" class="flex items-center justify-center w-8 h-8 rounded-full border border-gray-300 text-gray-600 hover:bg-gray-50 hover:border-primary hover:text-primary transition-colors" title="添加小组" @click="$emit('add')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
        </svg>
      </button>
    </div>
    <div class="space-y-3">
      <div v-if="loading" class="text-center py-4">
        <div class="animate-spin rounded-full h-6 w-6 border-t-2 border-b-2 border-primary mx-auto"></div>
        <p class="mt-2 text-sm text-gray-600">加载小组中...</p>
      </div>
      <div v-else-if="groups.length === 0" class="text-center py-4">
        <p class="text-gray-600">该工作空间暂无已添加的小组</p>
      </div>
      <div v-else class="space-y-2">
        <div v-for="group in groups" :key="group.id" class="flex items-center justify-between p-3 border border-gray-200 rounded-md">
          <div class="flex items-center space-x-3">
            <div class="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
              </svg>
            </div>
            <div><p class="font-medium">{{ group.name }}</p></div>
          </div>
          <div class="flex items-center space-x-2">
            <select v-model="access[group.id]" class="border border-gray-300 rounded-md px-3 py-1 text-sm" @change="$emit('updateAccess', group.id)">
              <option value="">无权限</option>
              <option value="view">仅查看</option>
              <option value="edit">可编辑</option>
              <option value="admin">管理员</option>
            </select>
            <button type="button" class="w-6 h-6 flex items-center justify-center text-red-500 hover:text-red-700 transition-colors" title="移除小组" @click="$emit('remove', group.id)">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  groups: { type: Array, default: () => [] },
  access: { type: Object, default: () => ({}) },
  loading: { type: Boolean, default: false },
})
defineEmits(['add', 'updateAccess', 'remove'])
</script>
