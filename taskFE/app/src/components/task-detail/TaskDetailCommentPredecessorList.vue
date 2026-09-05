<template>
  <div
    class="rounded-md border border-amber-200 bg-amber-50/70"
    data-testid="comment-execution-predecessor-list"
  >
    <p
      v-if="unfinishedCount"
      class="px-2 pt-1.5 text-[11px] text-amber-900"
      data-testid="comment-execution-predecessor-unfinished"
    >{{ unfinishedCount }} 个未完成</p>
    <ul
      v-if="predecessors.length"
      class="px-2 py-1.5 space-y-1"
      data-testid="comment-execution-predecessor-rows"
    >
      <li
        v-for="row in predecessors"
        :key="row.id"
      >
        <button
          type="button"
          class="w-full flex flex-wrap items-center gap-1.5 rounded-md border border-amber-100 bg-white/80 px-2 py-1 text-left hover:bg-white"
          data-testid="comment-execution-predecessor-row"
          :data-predecessor-id="row.id"
          :data-predecessor-status="row.status || ''"
          @click.stop="emit('focus-predecessor', row.id)"
        >
          <span
            class="min-w-0 flex-1 text-[11px] text-gray-800 break-all"
            data-testid="comment-execution-predecessor-summary"
          >{{ row.summary }}</span>
          <span
            class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium border shrink-0"
            :class="statusClass(row)"
            data-testid="comment-execution-predecessor-status"
          >{{ row.statusLabel }}</span>
        </button>
      </li>
    </ul>
    <p
      v-if="hasMissing"
      class="px-2 pb-1.5 text-[11px] text-amber-800/80"
      data-testid="comment-execution-predecessor-unloaded-hint"
    >加载更早评论后可查看前序</p>
    <p
      v-else-if="!predecessors.length"
      class="px-2 py-1.5 text-[11px] text-amber-900"
      data-testid="comment-execution-predecessor-empty"
    >前序不在当前页，可加载更早评论</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  predecessors: { type: Array, default: () => [] },
})

const emit = defineEmits(['focus-predecessor'])

const unfinishedCount = computed(() =>
  props.predecessors.filter((row) => row && row.finished !== true).length,
)

const hasMissing = computed(() =>
  props.predecessors.some((row) => row && row.missing === true),
)

function statusClass(row) {
  const s = String(row?.status || '')
  if (s === 'failed') return 'bg-red-50 text-red-800 border-red-100'
  if (s === 'completed') return 'bg-emerald-50 text-emerald-800 border-emerald-100'
  if (s === 'running' || s === 'starting') return 'bg-sky-50 text-sky-900 border-sky-100'
  if (s === 'waiting_previous' || s === 'pending') return 'bg-amber-50 text-amber-900 border-amber-100'
  return 'bg-slate-50 text-slate-700 border-slate-200'
}
</script>
