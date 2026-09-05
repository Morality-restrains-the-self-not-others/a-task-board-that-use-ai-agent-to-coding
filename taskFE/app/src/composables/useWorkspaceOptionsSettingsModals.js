import { ref } from 'vue'

/** 工作区设置页：创建任务字段（含任务类型/编程语言可选值）弹窗状态。 */
export function useWorkspaceOptionsSettingsModals() {
  const showCreateTaskFieldSettingsModal = ref(false)
  const createTaskFieldSettingsWorkspaceId = ref('')
  const createTaskFieldSettingsWorkspaceName = ref('')

  const openCreateTaskFieldSettings = (workspace) => {
    createTaskFieldSettingsWorkspaceId.value = workspace?.value || ''
    createTaskFieldSettingsWorkspaceName.value = workspace?.text || ''
    showCreateTaskFieldSettingsModal.value = true
  }

  const closeCreateTaskFieldSettings = () => {
    showCreateTaskFieldSettingsModal.value = false
    createTaskFieldSettingsWorkspaceId.value = ''
    createTaskFieldSettingsWorkspaceName.value = ''
  }

  return {
    showCreateTaskFieldSettingsModal,
    createTaskFieldSettingsWorkspaceId,
    createTaskFieldSettingsWorkspaceName,
    openCreateTaskFieldSettings,
    closeCreateTaskFieldSettings,
  }
}
