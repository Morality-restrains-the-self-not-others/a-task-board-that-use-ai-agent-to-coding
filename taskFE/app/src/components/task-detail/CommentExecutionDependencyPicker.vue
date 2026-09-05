<template>
  <div
    class="comment-execution-dependency-picker space-y-2 rounded-md border border-gray-100 bg-gray-50/80 px-2.5 py-2"
    data-testid="comment-execution-dependency-picker"
  >
    <p class="text-[11px] font-medium text-gray-600">
      {{ showQueueToggle ? '自动执行' : '执行依赖' }}
    </p>
    <TaskDetailQueuedScheduleToggle
      v-if="showQueueToggle && tenantId && workspaceId && task?.id"
      :task="task"
      :tenant-id="tenantId"
      :workspace-id="workspaceId"
      @updated="emit('task-updated', $event)"
    />
    <p
      v-if="showQueueToggle"
      class="text-[11px] text-gray-600 leading-relaxed"
      data-testid="comment-dep-queue-serial-hint"
    >
      {{ queueSerialHint }}
    </p>
    <p
      v-if="showQueueToggle"
      class="text-[11px] font-medium text-gray-600 pt-1"
    >
      执行依赖
    </p>
    <div class="flex flex-wrap gap-3 text-xs text-gray-700">
      <label class="inline-flex items-center gap-1.5 cursor-pointer">
        <input
          v-model="primaryMode"
          type="radio"
          class="text-primary"
          value="wait"
          data-testid="comment-dep-wait"
        >
        等待前序完成
      </label>
      <label
        class="inline-flex items-center gap-1.5"
        :class="queueSerialLocked ? 'cursor-not-allowed opacity-60' : 'cursor-pointer'"
      >
        <input
          v-model="primaryMode"
          type="radio"
          class="text-primary"
          value="independent"
          data-testid="comment-dep-independent"
          :disabled="queueSerialLocked"
        >
        不等待前序
      </label>
    </div>

    <div
      v-if="primaryMode === 'wait'"
      class="pl-1 space-y-2 border-l-2 border-amber-200"
      data-testid="comment-dep-wait-options"
    >
      <div class="flex flex-wrap gap-3 text-xs text-gray-700">
        <label class="inline-flex items-center gap-1.5 cursor-pointer">
          <input
            v-model="waitScope"
            type="radio"
            class="text-primary"
            value="all"
            data-testid="comment-dep-scope-all"
          >
          等前面全部评论
        </label>
        <label class="inline-flex items-center gap-1.5 cursor-pointer">
          <input
            v-model="waitScope"
            type="radio"
            class="text-primary"
            value="selected"
            data-testid="comment-dep-scope-selected"
            :disabled="!predecessorOptions.length"
          >
          等指定评论
        </label>
      </div>
      <p
        v-if="waitScope === 'selected' && !predecessorOptions.length"
        class="text-[11px] text-gray-500"
        data-testid="comment-dep-no-predecessors"
      >
        暂无前序评论可选
      </p>
      <ul
        v-if="waitScope === 'selected' && predecessorOptions.length"
        class="max-h-36 overflow-y-auto space-y-1 rounded border border-gray-200 bg-white p-2"
        data-testid="comment-dep-predecessor-list"
      >
        <li
          v-for="opt in predecessorOptions"
          :key="opt.id"
          class="flex items-start gap-2 text-xs text-gray-700"
        >
          <input
            :id="`dep-pred-${opt.id}`"
            type="checkbox"
            class="mt-0.5 text-primary"
            :checked="selectedIds.includes(opt.id)"
            data-testid="comment-dep-predecessor-item"
            @change="toggleId(opt.id, $event.target.checked)"
          >
          <label :for="`dep-pred-${opt.id}`" class="cursor-pointer min-w-0">
            <span class="font-medium text-gray-800">{{ opt.authorLabel || '评论' }}</span>
            <span class="text-gray-400 mx-1">·</span>
            <span class="text-gray-600 break-words">{{ opt.summary || '（无正文）' }}</span>
          </label>
        </li>
      </ul>
    </div>

    <div class="pt-2 mt-1 border-t border-gray-200/70">
      <label
        class="inline-flex items-start gap-2 cursor-pointer"
        data-testid="comment-dep-auto-commit"
      >
        <input
          v-model="autoCommit"
          type="checkbox"
          class="mt-0.5 text-primary"
        >
        <span class="text-xs text-gray-700 leading-relaxed">
          <span class="font-medium">自动提交</span>
          <span class="text-gray-500 block text-[11px]">
            AI 代理完成后自动暂存所有变更并提交（git add -A &amp;&amp; git commit）
          </span>
        </span>
      </label>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import TaskDetailQueuedScheduleToggle from './TaskDetailQueuedScheduleToggle.vue'

const executionMode = defineModel('executionMode', { type: String, default: 'wait_previous' })
const dependsOnCommentIds = defineModel('dependsOnCommentIds', { type: Array, default: () => [] })
const autoCommit = defineModel('autoCommit', { type: Boolean, default: false })

const props = defineProps({
  predecessorOptions: { type: Array, default: () => [] },
  showQueueToggle: { type: Boolean, default: false },
  task: { type: Object, default: null },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
})

const emit = defineEmits(['task-updated'])

const queueSerialLocked = computed(() => Boolean(props.task?.queued_auto_run))
const queueSerialHint = computed(() => {
  if (queueSerialLocked.value) {
    return '已加入自动执行队列：本任务轮到后，评论按提交顺序一条一条执行（等前序完成）。'
  }
  return '加入自动执行队列后，本任务的评论会按提交顺序一条一条执行（需先启用工作空间「自动调度」）。'
})

const primaryMode = ref(
  queueSerialLocked.value || executionMode.value !== 'independent' ? 'wait' : 'independent',
)
const waitScope = ref(
  Array.isArray(dependsOnCommentIds.value) && dependsOnCommentIds.value.length > 0
    ? 'selected'
    : 'all',
)
const selectedIds = ref(
  Array.isArray(dependsOnCommentIds.value) ? dependsOnCommentIds.value.map(String) : [],
)

const syncOut = computed(() => {
  if (queueSerialLocked.value || primaryMode.value === 'wait') {
    if (waitScope.value === 'selected') {
      return { mode: 'wait_previous', ids: selectedIds.value.map(String) }
    }
    return { mode: 'wait_previous', ids: [] }
  }
  return { mode: 'independent', ids: [] }
})

watch(
  syncOut,
  (v) => {
    if (executionMode.value !== v.mode) executionMode.value = v.mode
    const cur = Array.isArray(dependsOnCommentIds.value)
      ? dependsOnCommentIds.value.map(String)
      : []
    const same = cur.length === v.ids.length && cur.every((id, i) => id === v.ids[i])
    if (!same) dependsOnCommentIds.value = [...v.ids]
  },
  { immediate: true, deep: true },
)

watch(queueSerialLocked, (locked) => {
  if (locked) primaryMode.value = 'wait'
})

watch(primaryMode, (mode) => {
  if (mode !== 'wait') {
    waitScope.value = 'all'
    selectedIds.value = []
  }
})

watch(
  () => props.predecessorOptions.map((o) => String(o.id)).join(','),
  () => {
    const allowed = new Set(props.predecessorOptions.map((o) => String(o.id)))
    selectedIds.value = selectedIds.value.filter((id) => allowed.has(String(id)))
  },
)

function toggleId(id, checked) {
  const sid = String(id)
  if (checked) {
    if (!selectedIds.value.includes(sid)) selectedIds.value = [...selectedIds.value, sid]
  } else {
    selectedIds.value = selectedIds.value.filter((x) => x !== sid)
  }
}
</script>
