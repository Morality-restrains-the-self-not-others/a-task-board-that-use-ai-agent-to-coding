<template>
  <p
    v-if="visible"
    class="mt-2 text-xs text-gray-600"
    data-testid="machine-owner-hint"
  >
    <span class="text-gray-500">机器节点所属任务：</span>
    <template v-for="(entry, index) in entries" :key="entry.taskId">
      <span v-if="index > 0">、</span>
      <a
        v-if="entry.href"
        :href="entry.href"
        class="text-primary underline hover:no-underline"
        :data-testid="`machine-owner-task-link-${entry.taskId}`"
      >{{ entry.taskId }}</a>
      <span
        v-else
        :data-testid="`machine-owner-task-${entry.taskId}`"
      >{{ entry.taskId }}</span>
      <span
        v-if="entry.isViewer"
        class="ml-1 text-gray-400"
        data-testid="machine-owner-viewer-badge"
      >（本任务）</span>
    </template>
  </p>
</template>

<script setup>
import { computed } from 'vue'
import {
  shouldShowMachineOwnerHint,
  normalizeOwnerTaskIds,
  buildMachineOwnerEntries,
} from '../utils/machineOwnerHint.js'

const props = defineProps({
  runtimeStatus: { type: Object, default: null },
  viewerTaskId: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  serverUrl: { type: String, default: '' },
  containerPageUrl: { type: String, default: '' },
  mockContainerRunning: { type: Boolean, default: false },
})

const visible = computed(() => shouldShowMachineOwnerHint(props.runtimeStatus, {
  serverUrl: props.serverUrl,
  containerPageUrl: props.containerPageUrl,
  mockContainerRunning: props.mockContainerRunning,
}))

const entries = computed(() => buildMachineOwnerEntries(
  normalizeOwnerTaskIds(props.runtimeStatus),
  props.viewerTaskId,
  { tenantId: props.tenantId, workspaceId: props.workspaceId },
))
</script>
