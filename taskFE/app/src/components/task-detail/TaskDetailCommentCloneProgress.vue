<template>
  <div
    v-if="rows.length"
    class="pt-1 space-y-1"
    data-testid="comment-execution-clone-progress"
  >
    <div class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-[11px]">
      <span class="text-gray-500 shrink-0">项目克隆</span>
      <span
        class="tabular-nums text-gray-800"
        data-testid="comment-execution-clone-progress-overall"
      >{{ overallPct }}%</span>
      <span
        v-if="hasSubRepos"
        class="text-gray-400"
        data-testid="comment-execution-clone-progress-overall-count"
      >{{ doneCount }}/{{ expectedTotal }} 仓库</span>
    </div>
    <div
      class="w-full bg-sky-100 rounded-full h-2 overflow-hidden"
      data-testid="comment-execution-clone-progress-overall-bar"
    >
      <div
        class="h-2 rounded-full"
        :class="overallBarClass"
        :style="{ width: overallPct + '%' }"
      />
    </div>
    <details
      v-if="hasSubRepos"
      class="rounded border border-sky-100 bg-white/70"
      data-testid="comment-execution-clone-progress-sub"
    >
      <summary
        class="cursor-pointer select-none px-1.5 py-1 text-[10px] text-sky-900"
        data-testid="comment-execution-clone-progress-sub-summary"
      >
        各仓库进度
        <template v-if="retryingCount"> · {{ retryingCount }} 个重试中</template>
        <template v-if="failedCount"> · {{ failedCount }} 个失败，可手动重试</template>
      </summary>
      <div class="px-1.5 pb-1.5 space-y-1">
        <TaskDetailCommentCloneProgressRow
          v-for="row in rows"
          :key="row.key"
          :row="row"
          :reclone-loading-by-url="recloneLoadingByUrl"
          :reclone-status-by-url="recloneStatusByUrl"
          :reclone-error-trace-id-by-url="recloneErrorTraceIdByUrl"
          :repo-reclone-global-loading="repoRecloneGlobalLoading"
          @repo-reclone="emit('repo-reclone', $event)"
        />
      </div>
    </details>
    <template v-else>
      <TaskDetailCommentCloneProgressRow
        v-for="row in rows"
        :key="row.key"
        :row="row"
        :reclone-loading-by-url="recloneLoadingByUrl"
        :reclone-status-by-url="recloneStatusByUrl"
        :reclone-error-trace-id-by-url="recloneErrorTraceIdByUrl"
        :repo-reclone-global-loading="repoRecloneGlobalLoading"
        @repo-reclone="emit('repo-reclone', $event)"
      />
    </template>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import {
  expectedCloneRepoTotal,
  overallCloneProgressPct,
} from '../../utils/commentCloneProgressFromLogs.js'
import TaskDetailCommentCloneProgressRow from './TaskDetailCommentCloneProgressRow.vue'

const props = defineProps({
  rows: { type: Array, default: () => [] },
  recloneLoadingByUrl: { type: Object, default: () => ({}) },
  recloneStatusByUrl: { type: Object, default: () => ({}) },
  recloneErrorTraceIdByUrl: { type: Object, default: () => ({}) },
  repoRecloneGlobalLoading: { type: Boolean, default: false },
})

const emit = defineEmits(['repo-reclone'])

const list = computed(() => (Array.isArray(props.rows) ? props.rows : []))
const expectedTotal = computed(() => expectedCloneRepoTotal(list.value))
const overallPct = computed(() => overallCloneProgressPct(list.value))
const hasSubRepos = computed(() => expectedTotal.value > 1 || list.value.length > 1)
const doneCount = computed(() => list.value.filter((row) => Number(row?.progress) >= 100 && !row?.failed).length)
const failedCount = computed(() => list.value.filter((row) => row?.failed && !row?.retrying).length)
const retryingCount = computed(() => list.value.filter((row) => row?.retrying).length)
const overallBarClass = computed(() => {
  if (list.value.some((row) => row?.failed && !row?.retrying)) return 'bg-red-500'
  if (list.value.some((row) => row?.retrying)) return 'bg-amber-500'
  return 'bg-sky-600'
})
</script>
