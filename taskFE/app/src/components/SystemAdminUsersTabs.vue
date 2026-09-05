<template>
  <div class="flex space-x-1 border-b border-gray-200" data-testid="system-admin-users-tabs">
    <button
      class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
      :class="activeTab === 'active' ? 'bg-white text-primary border border-gray-200 border-b-white -mb-px' : 'text-gray-500 hover:text-gray-700'"
      @click="$emit('switch', 'active')"
    >
      活跃用户
    </button>
    <button
      class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
      :class="activeTab === 'archived' ? 'bg-white text-primary border border-gray-200 border-b-white -mb-px' : 'text-gray-500 hover:text-gray-700'"
      data-testid="system-admin-users-tab-archived"
      @click="$emit('switch', 'archived')"
    >
      已归档
    </button>
    <button
      class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
      :class="activeTab === 'tenants' ? 'bg-white text-primary border border-gray-200 border-b-white -mb-px' : 'text-gray-500 hover:text-gray-700'"
      data-testid="system-admin-users-tab-tenants"
      @click="$emit('switch', 'tenants')"
    >
      租户
    </button>
    <button
      class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
      :class="activeTab === 'referral-apps' ? 'bg-white text-primary border border-gray-200 border-b-white -mb-px' : 'text-gray-500 hover:text-gray-700'"
      data-testid="system-admin-users-tab-referral-apps"
      @click="$emit('switch', 'referral-apps')"
    >
      推荐码申请
    </button>
  </div>
</template>

<script setup>
defineProps({
  activeTab: { type: String, required: true },
})
defineEmits(['switch'])
</script>
