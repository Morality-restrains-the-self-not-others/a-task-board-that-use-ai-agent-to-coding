<template>
  <div
    class="space-y-0.5"
    data-testid="comment-execution-clone-progress-row"
  >
    <div class="flex justify-between gap-2 text-[10px] text-sky-800/90">
      <span class="truncate min-w-0" :title="row.label">{{ row.label }}</span>
      <span class="tabular-nums shrink-0">{{ Math.round(Number(row.progress) || 0) }}%</span>
    </div>
    <template v-if="hasSubPhases">
      <div class="flex gap-2 min-w-0">
        <div class="min-w-0 flex-1">
          <div class="flex justify-between text-[10px] text-sky-800/90 mb-0.5">
            <span>接收</span>
            <span>{{ recvPct }}%</span>
          </div>
          <div class="w-full bg-sky-100 rounded-full h-1.5 overflow-hidden">
            <div class="h-1.5 rounded-full bg-sky-600" :style="{ width: recvPct + '%' }" />
          </div>
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex justify-between text-[10px] text-sky-800/90 mb-0.5">
            <span>解压</span>
            <span>{{ unpackPct }}%</span>
          </div>
          <div class="w-full bg-sky-100 rounded-full h-1.5 overflow-hidden">
            <div class="h-1.5 rounded-full bg-indigo-500" :style="{ width: unpackPct + '%' }" />
          </div>
        </div>
      </div>
    </template>
    <div
      v-else
      class="w-full bg-sky-100 rounded-full h-1.5 overflow-hidden"
    >
      <div
        class="h-1.5 rounded-full"
        :class="barClass"
        :style="{ width: barWidth + '%' }"
      />
    </div>
    <p
      v-if="row.message"
      class="m-0 text-[10px] leading-snug line-clamp-2"
      :class="row.failed ? 'text-red-700' : (row.retrying ? 'text-amber-800' : 'text-sky-800/90')"
      data-testid="comment-execution-clone-progress-message"
    >{{ row.message }}</p>
    <div
      v-if="showRetry"
      class="flex flex-wrap items-center gap-1"
    >
      <button
        type="button"
        class="text-[10px] px-1.5 py-0.5 border border-gray-300 rounded bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed shrink-0"
        data-testid="comment-execution-clone-progress-retry"
        :disabled="retryDisabled"
        @click.stop.prevent="onRetry"
      >
        <span
          v-if="retryLoading"
          class="inline-block animate-spin h-2.5 w-2.5 border-2 border-gray-300 border-t-sky-600 rounded-full mr-0.5 align-middle"
        />
        手动重试
      </button>
      <span
        v-if="retryStatusText"
        class="text-[10px]"
        :class="retryStatusText === 'ok' ? 'text-green-600' : 'text-red-600'"
        data-testid="comment-execution-clone-progress-retry-status"
        :data-traceId="retryStatusTraceId || undefined"
      >{{ retryStatusText === 'ok' ? '已发起' : retryStatusText }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import {
  cloneProgressRowHasSubPhases,
  cloneProgressRecvPct,
  cloneProgressUnpackPct,
} from '../../utils/taskDetailContainerCloneProgress.js'
import {
  canEmitCloneManualRetry,
  shouldShowCloneManualRetry,
} from '../../utils/commentCloneProgressFromLogs.js'

const props = defineProps({
  row: { type: Object, required: true },
  recloneLoadingByUrl: { type: Object, default: () => ({}) },
  recloneStatusByUrl: { type: Object, default: () => ({}) },
  recloneErrorTraceIdByUrl: { type: Object, default: () => ({}) },
  repoRecloneGlobalLoading: { type: Boolean, default: false },
})

const emit = defineEmits(['repo-reclone'])

const repoRef = computed(() => String(props.row?.repoUrl || props.row?.key || '').trim())

const hasSubPhases = computed(() => cloneProgressRowHasSubPhases(props.row))
const recvPct = computed(() => cloneProgressRecvPct(props.row))
const unpackPct = computed(() => cloneProgressUnpackPct(props.row))
const barWidth = computed(() => Math.min(100, Math.max(0, Number(props.row?.progress) || 0)))
const barClass = computed(() => {
  if (props.row?.failed) return 'bg-red-500'
  if (props.row?.retrying) return 'bg-amber-500'
  return 'bg-sky-600'
})
const showRetry = computed(() => shouldShowCloneManualRetry(props.row))
const retryLoading = computed(() => Boolean(props.recloneLoadingByUrl[repoRef.value]))
const retryDisabled = computed(() => (
  props.repoRecloneGlobalLoading
  || retryLoading.value
  || !canEmitCloneManualRetry(props.row)
))
const retryStatusText = computed(() => {
  const s = props.recloneStatusByUrl[repoRef.value]
  return typeof s === 'string' ? s : ''
})
const retryStatusTraceId = computed(() => {
  if (!retryStatusText.value || retryStatusText.value === 'ok') return ''
  return String(props.recloneErrorTraceIdByUrl[repoRef.value] || '').trim()
})

function onRetry() {
  if (retryDisabled.value) return
  const payload = { repoUrl: repoRef.value }
  const parentRepoUrl = String(props.row?.parentRepoUrl || '').trim()
  const cloneAlias = String(props.row?.cloneAlias || '').trim()
  if (parentRepoUrl) payload.parentRepoUrl = parentRepoUrl
  if (cloneAlias) payload.cloneAlias = cloneAlias
  emit('repo-reclone', payload)
}
</script>
