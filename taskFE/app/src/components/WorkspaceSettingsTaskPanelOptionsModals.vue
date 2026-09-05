<template>
  <CreateTaskFieldSettingsModal
    :show="showCreateTaskFieldSettingsModal"
    :tenant-id="tenantId"
    :workspace-id="createTaskFieldSettingsWorkspaceId"
    :workspace-name="createTaskFieldSettingsWorkspaceName"
    @close="closeCreateTaskFieldSettings"
  />
</template>

<script setup>
import CreateTaskFieldSettingsModal from './CreateTaskFieldSettingsModal.vue'
import { useWorkspaceOptionsSettingsModals } from '../composables/useWorkspaceOptionsSettingsModals.js'

defineProps({
  tenantId: { type: [String, Number], default: '' },
})

const {
  showCreateTaskFieldSettingsModal,
  createTaskFieldSettingsWorkspaceId,
  createTaskFieldSettingsWorkspaceName,
  openCreateTaskFieldSettings,
  closeCreateTaskFieldSettings,
} = useWorkspaceOptionsSettingsModals()

defineExpose({
  openCreateTaskFieldSettings,
})
</script>
