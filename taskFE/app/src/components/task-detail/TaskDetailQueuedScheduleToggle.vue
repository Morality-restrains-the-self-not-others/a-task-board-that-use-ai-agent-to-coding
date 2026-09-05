<template>
  <div class="space-y-1.5" data-testid="task-queued-auto-run-toggle">
    <div class="flex flex-wrap items-center gap-2">
      <button
        v-if="!queuedOn"
        type="button"
        class="px-3 py-1.5 text-sm bg-gray-900 text-white rounded hover:bg-gray-800 transition-all duration-150 active:scale-95 disabled:opacity-60"
        data-testid="queued-auto-run-join"
        :disabled="saving || saveGuard.isBusy()"
        :aria-busy="saving || saveGuard.isBusy() ? 'true' : undefined"
        @click="onJoinQueue"
      >
        加入自动执行队列
      </button>
      <button
        v-else
        type="button"
        class="px-3 py-1.5 text-sm border border-gray-300 text-gray-800 bg-white rounded hover:bg-gray-50 transition-all duration-150 active:scale-95 disabled:opacity-60"
        data-testid="queued-auto-run-leave"
        :disabled="saving || saveGuard.isBusy()"
        @click="leaveQueue"
      >
        离开队列
      </button>
      <span
        v-if="statusText"
        class="text-sm text-amber-700"
        data-testid="queued-auto-run-status-chip"
      >
        {{ statusText }}
      </span>
    </div>
    <router-link
      :to="schedulePageRoute"
      class="text-sm text-primary hover:underline"
      data-testid="queued-schedule-page-link"
    >
      前往「自动调度安排」设置时段
    </router-link>
    <p
      v-if="saveError"
      class="text-xs text-red-600"
      :data-traceId="saveErrorTraceId || undefined"
      data-testid="queued-auto-run-save-error"
    >
      {{ saveError }}
    </p>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useQueuedAutoRunPanel } from './useQueuedAutoRunPanel.js'
import { fetchWorkspaceAutoScheduleEnabled } from '../../utils/workspaceAutoScheduleEnabled.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../../utils/clickGuard.js'
import modalService from '../../utils/modalService.js'

// 任务详情「添加评论」自动执行卡内嵌：加入/离开队列 + 状态 chip + 「自动调度安排」入口。
// 时段/并发等节奏设置仍在工作空间「自动调度安排」页（OPT-20260824-067）。
const props = defineProps({
  task: { type: Object, required: true },
  tenantId: { type: String, required: true },
  workspaceId: { type: String, required: true },
})
const emit = defineEmits(['updated'])

const modalOpen = ref(false) // 仅用于 useQueuedAutoRunPanel 快照触发；本组件不弹窗
const saving = ref(false)
const saveError = ref('')
const saveErrorTraceId = ref('')

const schedulePageRoute = computed(
  () => `/tenant/${props.tenantId}/queue-schedule/?workspace_id=${encodeURIComponent(props.workspaceId)}`,
)

const saveGuard = createClickGuard()

async function patchTask(body) {
  // OPT-20260819-038: 加入/离开队列是写操作，防连点双发 PATCH
  const outcome = await saveGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    saveError.value = ''
    saveErrorTraceId.value = ''
    try {
      const path = `/api/tenant/${props.tenantId}/workspace/${props.workspaceId}/todos/${props.task.id}/`
      const apiFetch = window.apiFetch
      if (typeof apiFetch !== 'function') {
        throw new Error('apiFetch unavailable')
      }
      const resp = await apiFetch(path, {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify(body),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        const err = new Error(data?.error || data?.message || `HTTP ${resp.status}`)
        err.traceId = resp.headers?.get?.('X-Trace-Id') || data?.trace_id || ''
        throw err
      }
      emit('updated', data)
      return true
    } catch (e) {
      saveError.value = e?.message || '保存失败'
      saveErrorTraceId.value = e?.traceId || ''
      return false
    } finally {
      saving.value = false
    }
  })
  return outcome ? Boolean(outcome.result) : false
}

const {
  queuedOn,
  statusText,
  joinQueue,
  leaveQueue,
} = useQueuedAutoRunPanel(props, { patchTask, modalOpen })

async function promptEnableAutoSchedule() {
  try {
    await modalService.confirm(
      '当前工作空间尚未启用自动调度。请先前往「自动调度安排」开启后再加入队列。',
      '尚未启用自动调度',
      '前往设置',
      '取消',
    )
  } catch {
    return
  }
  // 确认后整页跳转设置页（真实地址），非拦截 <a> 左键。
  window.location.assign(schedulePageRoute.value)
}

async function onJoinQueue() {
  if (saving.value || saveGuard.isBusy()) return
  saving.value = true
  saveError.value = ''
  saveErrorTraceId.value = ''
  try {
    const { enabled } = await fetchWorkspaceAutoScheduleEnabled({
      tenantId: props.tenantId,
      workspaceId: props.workspaceId,
    })
    if (!enabled) {
      try {
        await promptEnableAutoSchedule()
      } finally {
        saving.value = false
      }
      return
    }
  } catch (e) {
    saveError.value = e?.message || '无法确认自动调度状态'
    saveErrorTraceId.value = e?.traceId || ''
    saving.value = false
    return
  }
  saving.value = false
  await joinQueue()
}
</script>
