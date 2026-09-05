<template>
  <div>
    <label
      class="flex items-start gap-2 cursor-pointer"
      :class="{ 'opacity-60 cursor-not-allowed': !canEnableAutoRun }"
      data-testid="task-auto-run-field"
    >
      <input
        id="task-auto-run"
        type="checkbox"
        class="mt-1 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
        data-testid="task-auto-run-checkbox"
        v-model="editingTask.auto_run"
        :disabled="!canEnableAutoRun"
      >
      <span class="text-sm">
        <span class="font-medium text-gray-700">是否自动运行</span>
        <span
          v-if="canEnableAutoRun"
          class="block text-xs text-gray-500 mt-0.5"
          data-testid="task-auto-run-enabled-hint"
        >
          {{ autoRunEnabledHint }}
        </span>
        <ul
          v-else
          class="mt-0.5 list-disc pl-4 text-xs text-gray-500 space-y-0.5"
          data-testid="task-auto-run-disabled-hints"
        >
          <li v-for="(reason, idx) in autoRunDisabledReasons" :key="idx">{{ reason }}</li>
        </ul>
      </span>
    </label>
    <CreateTaskAutoRunSteps v-if="editingTask.auto_run" :editing-task="editingTask" :installed-images="installedImages" />
    <label
      v-if="showQueuedAutoRunOption"
      class="mt-2 ml-6 flex items-start gap-2 cursor-pointer"
      data-testid="task-queued-auto-run-field"
    >
      <input
        id="task-queued-auto-run"
        type="checkbox"
        class="mt-1 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
        data-testid="task-queued-auto-run-checkbox"
        v-model="editingTask.queued_auto_run"
      >
      <span class="text-sm">
        <span class="font-medium text-gray-700">加入自动调度队列</span>
        <span
          class="block text-xs text-gray-500 mt-0.5"
          data-testid="task-queued-auto-run-hint"
        >
          {{ queuedAutoRunHint }}
        </span>
      </span>
    </label>
    <p
      v-if="scheduleLoadError"
      class="mt-1 ml-6 text-xs text-red-600"
      data-testid="task-queued-auto-run-schedule-error"
      :data-traceId="scheduleLoadErrorTraceId || undefined"
    >
      {{ scheduleLoadError }}
    </p>
    <label
      v-if="showForceAutoRunOption"
      class="mt-2 ml-6 flex items-start gap-2 cursor-pointer"
      data-testid="task-force-auto-run-field"
    >
      <input
        id="task-force-auto-run"
        type="checkbox"
        class="mt-1 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
        data-testid="task-force-auto-run-checkbox"
        v-model="editingTask.force_auto_run"
      >
      <span class="text-sm">
        <span class="font-medium text-gray-700">强制重新启动</span>
        <span class="block text-xs text-gray-500 mt-0.5" data-testid="task-force-auto-run-hint">
          {{ forceAutoRunHint }}
        </span>
      </span>
    </label>
  </div>
</template>

<script setup>
import CreateTaskAutoRunSteps from './CreateTaskAutoRunSteps.vue'
import { useCreateTaskAutoRun } from '../composables/useCreateTaskAutoRun.js'

const props = defineProps({
  editingTask: {
    type: Object,
    required: true,
  },
  projects: {
    type: Array,
    default: () => [],
  },
  installedImages: {
    type: Array,
    default: () => [],
  },
  onPrimaryProjectChange: {
    type: Function,
    default: null,
  },
  tenantId: {
    type: [String, Number],
    default: '',
  },
  workspaceId: {
    type: [String, Number],
    default: '',
  },
  show: {
    type: Boolean,
    default: false,
  },
})

const {
  canEnableAutoRun,
  autoRunDisabledReasons,
  autoRunEnabledHint,
  forceAutoRunHint,
  showForceAutoRunOption,
  showQueuedAutoRunOption,
  queuedAutoRunHint,
  scheduleLoadError,
  scheduleLoadErrorTraceId,
} = useCreateTaskAutoRun({
  editingTask: () => props.editingTask,
  projects: () => props.projects,
  installedImages: () => props.installedImages,
  onPrimaryProjectChange: props.onPrimaryProjectChange,
  tenantId: () => props.tenantId,
  workspaceId: () => props.workspaceId,
  show: () => props.show,
})
</script>
