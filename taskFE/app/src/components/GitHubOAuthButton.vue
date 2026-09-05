<template>
  <div>
    <span v-if="gitRepo" :class="['inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium', isAccessible ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800']">
      {{ isAccessible ? '可访问' : (accessMessage || '不可访问') }}
    </span>
    <button
      v-if="gitRepo"
      @click="handleClick"
      :disabled="loading"
      :class="[
        'px-3 py-1 rounded-md text-sm font-medium transition-colors ml-2 inline-flex items-center gap-1.5',
        isAccessible ? 'bg-red-600 hover:bg-red-700 text-white border border-red-700' : 'bg-blue-500 hover:bg-blue-600 text-white',
        loading && 'opacity-70 cursor-not-allowed'
      ]"
    >
      <span v-if="loading" class="animate-spin inline-block w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full"></span>
      {{ loading ? '授权中...' : (isAccessible ? '解除 OAuth' : 'OAuth 授权') }}
    </button>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

// 定义属性
const props = defineProps({
  gitRepo: {
    type: String,
    default: ''
  },
  isAccessible: {
    type: Boolean,
    default: false
  },
  accessMessage: {
    type: String,
    default: ''
  },
  tenantId: {
    type: String,
    required: true
  },
  projectId: {
    type: String,
    required: true
  },
  loading: {
    type: Boolean,
    default: false
  }
})

// 定义事件
const emit = defineEmits(['authorize', 'revoke'])

// 点击处理：已授权时解除，未授权时授权
const handleClick = () => {
  if (props.loading) return
  const payload = {
    tenantId: props.tenantId,
    projectId: props.projectId,
    repoUrl: props.gitRepo,
  }
  if (props.isAccessible) {
    emit('revoke', payload)
  } else {
    emit('authorize', payload)
  }
}
</script>

<style scoped>
/* 组件内样式可以根据需要添加 */
</style>
