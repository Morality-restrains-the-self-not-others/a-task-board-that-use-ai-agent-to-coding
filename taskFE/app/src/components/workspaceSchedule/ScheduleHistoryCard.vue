<template>
  <div class="bg-white p-4 rounded-xl shadow" data-testid="schedule-history-card">
    <h3 class="text-base font-bold text-text mb-3">调度历史</h3>
    <div
      v-if="loadError"
      class="text-sm text-red-600"
      data-testid="schedule-history-error"
      :data-traceId="loadErrorTraceId || undefined"
    >
      {{ loadError }}
    </div>
    <div
      v-else-if="items.length === 0"
      class="text-sm text-text-light"
      data-testid="schedule-history-empty"
    >
      尚无调度记录。入队、改时段或自动启服后会出现在这里。
    </div>
    <ol v-else class="space-y-2" data-testid="schedule-history-list">
      <li
        v-for="row in items"
        :key="row.id"
        class="flex flex-wrap gap-x-3 gap-y-1 items-baseline text-sm border-b border-gray-100 last:border-0 pb-2 last:pb-0"
        data-testid="schedule-history-row"
      >
        <span class="text-text-light whitespace-nowrap" data-testid="schedule-history-time">
          {{ formatTime(row.created_at) }}
        </span>
        <span
          class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs text-text"
          data-testid="schedule-history-type"
        >
          {{ eventTypeLabel(row.event_type) }}
        </span>
        <span class="text-text flex-1 min-w-[8rem]" data-testid="schedule-history-message">
          {{ row.message }}
        </span>
        <a
          v-if="row.href"
          :href="row.href"
          class="text-primary hover:underline"
          :data-testid="`schedule-history-task-${row.task_id}`"
        >{{ row.task_title || '查看任务' }}</a>
      </li>
    </ol>
    <div v-if="hasMore" class="mt-3">
      <!-- Anti-Replay-OK: read-only pagination GET; in-flight lock only -->
      <button
        type="button"
        class="px-3 py-1.5 text-sm border border-gray-300 rounded-md text-gray-800 bg-white hover:bg-gray-50 disabled:opacity-50"
        data-testid="schedule-history-load-more"
        :disabled="loading"
        :aria-busy="loading ? 'true' : 'false'"
        @click="$emit('load-more')"
      >
        {{ loading ? '加载中...' : '加载更多' }}
      </button>
    </div>
  </div>
</template>

<script>
const TYPE_LABELS = {
  rhythm_saved: '保存设置',
  member_enqueued: '加入队列',
  member_dequeued: '离开队列',
  member_started: '调度启服',
  window_entered: '进入时段',
  window_exited: '离开时段',
  auto_close_warned: '关闭预告',
  auto_close_released: '释放占用',
}

export default {
  name: 'ScheduleHistoryCard',
  props: {
    items: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
    hasMore: { type: Boolean, default: false },
    loadError: { type: String, default: '' },
    loadErrorTraceId: { type: String, default: '' },
    formatTime: { type: Function, required: true },
  },
  emits: ['load-more'],
  setup() {
    function eventTypeLabel(t) {
      return TYPE_LABELS[t] || t || '调度'
    }
    return { eventTypeLabel }
  },
}
</script>
