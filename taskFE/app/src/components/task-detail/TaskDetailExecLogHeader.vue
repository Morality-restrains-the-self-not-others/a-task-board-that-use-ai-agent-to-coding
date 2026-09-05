<template>
  <div class="flex items-center justify-between gap-2 mb-2">
    <h4 class="text-xs font-medium text-gray-600">执行日志</h4>
    <div class="flex items-center gap-2 shrink-0">
      <div class="flex items-center gap-2 shrink-0">
      <button
        type="button"
        :disabled="!copyable || loading"
        class="text-xs px-2 py-1 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
        data-testid="layer-exec-log-copy-btn"
        @click="emit('copy-log')"
      >
        {{ copyFeedback ? '已复制' : '复制日志' }}
      </button>
      <button
        type="button"
        :disabled="loading || !clearable"
        class="text-xs px-2 py-1 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
        data-testid="layer-exec-log-clear-btn"
        @click="emit('clear-log')"
      >
        清理日志
      </button>
    </div>
      <button
        type="button"
        :disabled="loading || !clearable"
        class="text-xs px-2 py-1 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
        data-testid="layer-exec-log-clear-btn"
        @click="emit('clear-log')"
      >
        清理日志
      </button>
    </div>
  </div>
</template>

<script setup>
defineProps({
  loading: { type: Boolean, default: false },
  copyable: { type: Boolean, default: false },
  copyFeedback: { type: Boolean, default: false },
  clearable: { type: Boolean, default: false },
})

const emit = defineEmits(['copy-log', 'clear-log'])
</script>
