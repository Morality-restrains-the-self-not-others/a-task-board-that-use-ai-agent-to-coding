<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h5 class="font-medium">成员权限</h5>
      <button type="button" class="flex items-center justify-center w-8 h-8 rounded-full border border-gray-300 text-gray-600 hover:bg-gray-50 hover:border-primary hover:text-primary transition-colors" title="添加成员" @click="$emit('add')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
        </svg>
      </button>
    </div>
    <div class="space-y-3">
      <div v-if="loading" class="text-center py-4">
        <div class="animate-spin rounded-full h-6 w-6 border-t-2 border-b-2 border-primary mx-auto"></div>
        <p class="mt-2 text-sm text-gray-600">加载成员中...</p>
      </div>
      <div v-else-if="members.length === 0" class="text-center py-4">
        <p class="text-gray-600">该工作空间暂无已添加的成员</p>
      </div>
      <div v-else class="space-y-2">
        <div v-for="member in members" :key="member.company_member_id" class="flex items-center justify-between p-3 border border-gray-200 rounded-md">
          <div class="flex items-center space-x-3">
            <div class="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center">
              <span class="text-sm font-medium">{{ (member.member_name || '').charAt(0).toUpperCase() }}</span>
            </div>
            <div>
              <div class="flex items-center space-x-2">
                <p class="font-medium">{{ member.member_name || '未知用户' }}</p>
                <span v-if="member.is_tenant" class="px-2 py-0.5 text-xs rounded-full bg-purple-100 text-purple-800">租户</span>
                <span v-else-if="access[member.company_member_id] === 'admin'" class="px-2 py-0.5 text-xs rounded-full bg-blue-100 text-blue-800">该空间管理员</span>
                <span v-else class="px-2 py-0.5 text-xs rounded-full bg-gray-100 text-gray-800">普通成员</span>
              </div>
              <p class="text-sm text-gray-500">{{ member.email || '未知邮箱' }}</p>
            </div>
          </div>
          <div class="flex items-center space-x-2">
            <select v-model="access[member.company_member_id]" class="border border-gray-300 rounded-md px-3 py-1 text-sm" @change="$emit('updateAccess', member.company_member_id)" :disabled="member.is_tenant">
              <option value="">无权限</option>
              <option value="view">仅查看</option>
              <option value="edit">可编辑</option>
              <option value="admin">管理员</option>
            </select>
            <button v-if="!member.is_tenant" type="button" class="w-6 h-6 flex items-center justify-center text-red-500 hover:text-red-700 transition-colors" title="移除成员" @click="$emit('remove', member.company_member_id)">
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
  members: { type: Array, default: () => [] },
  access: { type: Object, default: () => ({}) },
  loading: { type: Boolean, default: false },
})
defineEmits(['add', 'updateAccess', 'remove'])
</script>
