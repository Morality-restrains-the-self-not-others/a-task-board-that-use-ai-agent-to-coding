<template>
  <div class="border border-gray-200 rounded-lg p-4 hover:shadow-sm transition-shadow">
    <div class="flex justify-between items-start">
      <div>
        <h4 class="font-medium text-gray-900">{{ platformLabel }}</h4>
        <p class="text-sm text-gray-500 mt-1">
          授权方式: OAuth 2.0
        </p>
        <p class="text-sm text-gray-500 mt-1">
          创建时间: {{ formatDate(authorization.created_at) }}
        </p>
        <p class="text-sm text-gray-500 mt-1" v-if="authorization.expires_at">
          过期时间: {{ formatDate(authorization.expires_at) }}
        </p>
      </div>
      <div class="flex space-x-2">
        <button 
          class="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200 transition-colors"
          @click="handleDelete"
        >
          删除
        </button>
      </div>
    </div>
    <div class="mt-3 flex items-center justify-between">
      <span 
        class="px-2 py-1 text-xs rounded-full"
        :class="authorization.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'"
      >
        {{ authorization.is_active ? '已启用' : '已禁用' }}
      </span>
      <div>
        <button 
          class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-300"
          :class="authorization.is_active ? 'bg-primary' : 'bg-gray-200'"
          @click="handleToggleActive"
          :aria-pressed="authorization.is_active"
        >
          <span 
            class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
            :class="authorization.is_active ? 'translate-x-6' : 'translate-x-1'"
          ></span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

// Props
const props = defineProps({
  authorization: {
    type: Object,
    required: true
  },
  cloudPlatforms: {
    type: Array,
    required: true
  }
})

// Emits
const emit = defineEmits(['delete', 'toggle-active'])

// 计算属性：获取云平台名称
const platformLabel = computed(() => {
  const platform = props.cloudPlatforms.find(p => p.value === props.authorization.platform_type)
  return platform ? platform.label : props.authorization.platform_type
})

// 方法：格式化日期
const formatDate = (dateString) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 方法：处理删除
const handleDelete = () => {
  console.log('删除OAuth Token事件:', {
    authorizationId: props.authorization.id,
    platformType: props.authorization.platform_type,
    timestamp: new Date().toISOString()
  });
  emit('delete', props.authorization.id, props.authorization.platform_type)
}

// 方法：处理激活状态切换
const handleToggleActive = () => {
  emit('toggle-active', props.authorization.id, !props.authorization.is_active)
}
</script>

<style scoped>
/* 组件内样式 */
</style>