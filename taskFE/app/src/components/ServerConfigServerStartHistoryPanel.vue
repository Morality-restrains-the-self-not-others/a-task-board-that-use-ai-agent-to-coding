<template>
  <div class="p-6 bg-white border border-gray-200 rounded-lg">
    <div class="flex justify-between items-center mb-2 flex-wrap gap-2">
      <h3 class="text-sm font-medium text-gray-500">历史服务器启动记录</h3>
      <button
        type="button"
        class="px-3 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50"
        @click="emit('refresh')"
        :disabled="loading"
      >
        {{ loading ? '刷新中...' : '刷新记录' }}
      </button>
    </div>
    <div
      v-if="message"
      class="text-xs mb-3"
      :class="messageTraceId ? 'text-amber-700' : 'text-gray-500'"
      data-testid="server-start-history-message"
      :data-traceId="messageTraceId || undefined"
    >{{ message }}</div>
    <div
      v-if="records.length > 0"
      class="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(16rem,1fr))]"
    >
      <div
        v-for="record in records"
        :key="record.id"
        class="border border-gray-200 rounded-md p-3 bg-gray-50 text-xs text-gray-700 space-y-1"
        data-testid="server-start-history-card"
      >
        <div><span class="text-gray-500">启动时间：</span>{{ formatHistoryTime(record.created_at) }}</div>
        <div>
          <span class="text-gray-500">运行持续时间：</span>{{ formatServerStartHistoryDuration(record) }}
        </div>
        <div>
          <span class="text-gray-500">启动原因：</span>{{ formatServerStartHistoryStartReason(record.runtime_source) }}
        </div>
        <div>
          <span class="text-gray-500">关闭原因：</span>{{ formatServerStartHistoryStopReason(record.stop_reason) }}
        </div>
        <div><span class="text-gray-500">云平台：</span>{{ record.platform || '-' }}</div>
        <div><span class="text-gray-500">实例ID：</span>{{ record.instance_id || '-' }}</div>
        <div><span class="text-gray-500">实例规格：</span>{{ record.instance_type_id || '-' }}</div>
        <div><span class="text-gray-500">地域/可用区：</span>{{ record.region || '-' }} / {{ record.zone_id || '-' }}</div>
        <div>
          <span class="text-gray-500">硬件：</span>
          {{ record.hardware_config?.cpu_cores ?? '-' }}核 /
          {{ record.hardware_config?.memory_gb ?? '-' }}GB /
          {{ record.hardware_config?.storage_gb ?? '-' }}GB
        </div>
        <div v-if="record.error_reason" class="text-red-600 break-words">
          <span class="text-gray-500">失败原因：</span>{{ record.error_reason }}
        </div>
      </div>
    </div>
    <div v-else-if="!message && !loading" class="text-xs text-gray-500">暂无历史服务器启动记录</div>
  </div>
</template>

<script setup>
import {
  formatServerStartHistoryDuration,
  formatServerStartHistoryStartReason,
  formatServerStartHistoryStopReason,
} from '../utils/serverStartHistoryDisplay.js'

defineProps({
  records: {
    type: Array,
    default: () => [],
  },
  loading: {
    type: Boolean,
    default: false,
  },
  message: {
    type: String,
    default: '',
  },
  messageTraceId: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['refresh'])

const formatHistoryTime = (value) => {
  if (!value) {
    return '-'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }
  return date.toLocaleString()
}
</script>
