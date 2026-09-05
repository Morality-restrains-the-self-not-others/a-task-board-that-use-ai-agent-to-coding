<template>
  <summary
    class="cursor-pointer select-none px-2 py-1 text-xs text-gray-700 flex items-center justify-between gap-2"
  >
    <span class="min-w-0">
      共
      <span class="font-medium">{{ displayCount }}</span>
      个文件发生变动，点击查看文件列表
    </span>
    <button
      v-if="showRefreshButton"
      type="button"
      class="shrink-0 inline-flex items-center rounded border border-gray-300 bg-white px-2 py-0.5 text-[11px] font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
      :disabled="refreshDisabled || refreshing"
      data-testid="task-detail-layer-changes-refresh"
      @click.stop.prevent="$emit('refresh')"
    >
      {{ refreshing ? '刷新中…' : '刷新' }}
    </button>
  </summary>
</template>

<script setup>
defineProps({
  displayCount: { type: Number, default: 0 },
  showRefreshButton: { type: Boolean, default: false },
  refreshing: { type: Boolean, default: false },
  refreshDisabled: { type: Boolean, default: false },
})

defineEmits(['refresh'])
</script>
