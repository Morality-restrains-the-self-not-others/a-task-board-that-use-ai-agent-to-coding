<template>
  <div class="border border-gray-200 rounded-lg p-4 hover:shadow-sm transition-shadow">
    <div class="flex justify-between items-start">
      <div>
        <h4 class="font-medium text-gray-900">{{ getPlatformLabel(authorization.platform_type) }}</h4>
        <p class="text-sm text-gray-500 mt-1">
          授权方式: {{ getAuthorizationTypeLabel(authorization.authorization_type) }}
        </p>
        <p class="text-sm text-gray-500 mt-1">
          {{ authorization.remark ? `备注: ${authorization.remark}` : authorization.secret_id }}
        </p>
        <p class="text-sm text-gray-500 mt-1">
          创建时间: {{ formatDate(authorization.created_at) }}
        </p>
      </div>
      <div class="flex flex-wrap gap-2 justify-end">
        <button
          v-if="authorization.authorization_type === 'access_key'"
          type="button"
          class="px-3 py-1 text-sm bg-emerald-50 text-emerald-800 rounded hover:bg-emerald-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="isVerifyingCredentials"
          @click="$emit('verify-credentials', authorization)"
        >
          {{ isVerifyingCredentials ? '校验中…' : '测试密钥' }}
        </button>
        <button 
          class="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded hover:bg-gray-200 transition-colors"
          @click="$emit('edit', authorization)"
        >
          编辑
        </button>
        <button 
          class="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200 transition-colors"
          @click="handleDeleteClick"
        >
          删除
        </button>
      </div>
    </div>
    <div class="mt-3 flex items-center justify-between">
      <div>
        <span 
          class="px-2 py-1 text-xs rounded-full"
          :class="authorization.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'"
        >
          {{ authorization.is_active ? '已启用' : '已禁用' }}
        </span>
      </div>
      <div>
        <button 
          class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-300"
          :class="authorization.is_active ? 'bg-primary' : 'bg-gray-200'"
          @click="$emit('toggle-active', authorization.id, !authorization.is_active)"
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

// 定义组件属性
const props = defineProps({
  verifyingAuthorizationId: {
    type: [String, Number],
    default: null
  },
  authorization: {
    type: Object,
    required: true
  },
  cloudPlatforms: {
    type: Array,
    default: () => [
      { value: 'aliyun', label: '阿里云' },
      { value: 'tencentcloud', label: '腾讯云' },
      { value: 'huaweicloud', label: '华为云' },
      { value: 'ctyun', label: '天翼云' },
      { value: 'cmcc', label: '移动云' },
      { value: 'cucloud', label: '联通云' },
      { value: 'baiducloud', label: '百度智能云' },
      { value: 'aws', label: 'AWS' }
    ]
  }
})

const isVerifyingCredentials = computed(() => {
  const id = props.authorization?.id
  const pending = props.verifyingAuthorizationId
  return pending != null && String(pending) === String(id)
})

// 定义组件事件
const emit = defineEmits(['edit', 'delete', 'toggle-active', 'verify-credentials'])

// 获取云平台名称
const getPlatformLabel = (value) => {
  const platform = props.cloudPlatforms.find(p => p.value === value)
  return platform ? platform.label : value
}

// 获取授权方式标签
const getAuthorizationTypeLabel = (value) => {
  const typeMap = {
    'oauth': 'OAuth 2.0',
    'access_key': 'Access Key'
  };
  return typeMap[value] || value;
}

// 格式化日期
const formatDate = (dateString) => {
  if (!dateString) {
    return '-';
  }
  
  let date = new Date(dateString);
  
  if (isNaN(date.getTime())) {
    let processedDateStr = String(dateString);
    processedDateStr = processedDateStr.replace(/(\+\d{2}:\d{2}|Z)$/, '');
    processedDateStr = processedDateStr.replace('T', ' ');
    processedDateStr = processedDateStr.replace(/\.\d+/, '');
    date = new Date(processedDateStr);
  }
  
  if (isNaN(date.getTime())) {
    return String(dateString);
  }
  
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
}

// 处理删除点击事件
const handleDeleteClick = () => {
  console.log('删除云平台授权事件:', {
    authorizationId: props.authorization.id,
    platformType: props.authorization.platform_type,
    timestamp: new Date().toISOString()
  });
  emit('delete', props.authorization.id, props.authorization.platform_type);
}
</script>

<style scoped>
/* 组件内样式 */
</style>