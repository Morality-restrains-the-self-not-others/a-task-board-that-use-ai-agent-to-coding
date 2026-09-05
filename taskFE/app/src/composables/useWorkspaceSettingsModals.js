/**
 * settings/task-panel 工作空间管理页：各设置模态（任务存档 / 交付物 / 进度 /
 * 访问管理 / 机器策略 / 选项 / 功能参数）的开关状态与打开关闭逻辑。
 * 依赖 useWorkspaceSettingsTaskPanel 的 loadWorkspaces（存档保存后刷新列表）。
 */
import { ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

export function useWorkspaceSettingsModals({ tenantId, router, loadWorkspaces }) {
  const getCurrentTenantId = tenantId || (() => '')

  // 交付物体系相关
  const showDeliverableSystemSettingsModal = ref(false)
  const currentDeliverableSystemWorkspaceId = ref(null)

  // 进度体系相关
  const showProgressSystemSettingsModal = ref(false)
  const currentProgressSystemWorkspaceId = ref(null)

  // 访问管理相关
  const showAccessManagementModal = ref(false)
  const currentAccessManagementWorkspaceId = ref(null)

  const showMachinePolicyModal = ref(false)
  const machinePolicyWorkspaceId = ref('')
  const machinePolicyWorkspaceName = ref('')

  const optionsModalsRef = ref(null)

  // 任务存档
  const showTaskArchiveModal = ref(false)
  const taskArchiveModalWorkspace = ref(null)
  const taskArchiveTierDraft = ref('7d')
  const savingTaskArchive = ref(false)

  const openMachinePolicySettings = (workspace) => {
    machinePolicyWorkspaceId.value = workspace?.value || ''
    machinePolicyWorkspaceName.value = workspace?.text || ''
    showMachinePolicyModal.value = true
  }

  const openTaskArchiveSettings = (workspace) => {
    taskArchiveModalWorkspace.value = workspace
    taskArchiveTierDraft.value = workspace.task_archive_tier || '7d'
    showTaskArchiveModal.value = true
  }

  const saveTaskArchiveGuard = createClickGuard()

  const saveTaskArchiveTier = async () => {
    if (!taskArchiveModalWorkspace.value) return
    // OPT-20260819-038: 连点/超时重试会双发 PATCH — createClickGuard 在途锁 + Idempotency-Key
    await saveTaskArchiveGuard.run(async ({ idempotencyKey }) => {
      savingTaskArchive.value = true
      try {
        const tenantId = getCurrentTenantId()
        const wid = taskArchiveModalWorkspace.value.value
        const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}/${wid}/`, {
          method: 'PATCH',
          headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
          body: JSON.stringify({ task_archive_tier: taskArchiveTierDraft.value }),
        })

        if (!response.ok) {
          const err = new Error('Network response was not ok')
          err.traceId = response.traceId || ''
          throw err
        }
        showTaskArchiveModal.value = false
        taskArchiveModalWorkspace.value = null
        if (typeof loadWorkspaces === 'function') {
          await loadWorkspaces()
        }
      } catch (e) {
        console.error(e)
        showRequestError('保存失败，请重试', e)
      } finally {
        savingTaskArchive.value = false
      }
    })
  }

  // 处理交付物体系设置显示/隐藏
  const handleDeliverableSystemSettings = (workspaceId) => {
    currentDeliverableSystemWorkspaceId.value = workspaceId
    showDeliverableSystemSettingsModal.value = true
  }

  const handleCloseDeliverableSystemSettings = () => {
    showDeliverableSystemSettingsModal.value = false
    currentDeliverableSystemWorkspaceId.value = null
  }

  // 处理进度体系设置显示/隐藏
  const handleProgressSystemSettings = (workspaceId) => {
    currentProgressSystemWorkspaceId.value = workspaceId
    showProgressSystemSettingsModal.value = true
  }

  const handleCloseProgressSystemSettings = () => {
    showProgressSystemSettingsModal.value = false
    currentProgressSystemWorkspaceId.value = null
  }

  // 处理访问管理显示/隐藏
  const handleAccessManagement = (workspaceId) => {
    console.log('Opening AccessManagementModal with:', { workspaceId, tenantId: getCurrentTenantId() })
    currentAccessManagementWorkspaceId.value = workspaceId
    showAccessManagementModal.value = true
  }

  // 处理打开工作空间功能参数设置
  const openFeatureParamsSettings = (workspaceId) => {
    const tenantId = getCurrentTenantId()
    router.push(`/tenant/${tenantId}/settings/workspace/${workspaceId}/feature-params/`)
  }

  // 处理关闭访问管理
  const handleCloseAccessManagement = () => {
    showAccessManagementModal.value = false
    currentAccessManagementWorkspaceId.value = null
  }

  return {
    showDeliverableSystemSettingsModal,
    currentDeliverableSystemWorkspaceId,
    showProgressSystemSettingsModal,
    currentProgressSystemWorkspaceId,
    showAccessManagementModal,
    currentAccessManagementWorkspaceId,
    showMachinePolicyModal,
    machinePolicyWorkspaceId,
    machinePolicyWorkspaceName,
    optionsModalsRef,
    showTaskArchiveModal,
    taskArchiveModalWorkspace,
    taskArchiveTierDraft,
    savingTaskArchive,
    openMachinePolicySettings,
    openTaskArchiveSettings,
    saveTaskArchiveTier,
    handleDeliverableSystemSettings,
    handleCloseDeliverableSystemSettings,
    handleProgressSystemSettings,
    handleCloseProgressSystemSettings,
    handleAccessManagement,
    handleCloseAccessManagement,
    openFeatureParamsSettings,
  }
}
