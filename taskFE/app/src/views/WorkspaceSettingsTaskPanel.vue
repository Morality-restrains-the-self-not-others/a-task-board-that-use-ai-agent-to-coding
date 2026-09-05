<template>
<!-- 500-line rule: 工作空间列表/面板 CRUD/设置模态已抽到 useWorkspaceSettingsTaskPanel，添加/编辑面板模态已抽到 WorkspaceTaskPanelAddEditModal。 -->
  <TenantPageAccessEmpty v-if="!accessAllowed" page-key="settings.task_panel" />
  <div v-else class="p-8">
    <div class="flex flex-col space-y-6">
      <div class="flex justify-between items-center">
        <div>
          <h2 class="text-2xl font-bold text-text">工作空间管理</h2>
          <p class="text-text-light mt-1">管理工作空间</p>
        </div>
      </div>

      <!-- 工作空间选择器 -->
      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">工作空间管理</h3>
          <button
            class="btn-primary"
            @click="showCreateWorkspaceModal = true"
          >
            <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
            </svg>
            添加工作空间
          </button>
        </div>

        <div class="space-y-4">
          <div v-if="loadingWorkspaces" class="text-center py-8">
            <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary mx-auto"></div>
            <p class="mt-4 text-gray-600">加载中...</p>
          </div>
          <div v-else-if="workspaceOptions.length === 0" class="text-center py-8">
            <p class="text-gray-600">暂无工作空间</p>
            <button
              class="mt-4 btn-primary"
              @click="showCreateWorkspaceModal = true"
            >
              添加第一个工作空间
            </button>
          </div>
          <div
            v-else
            class="space-y-3"
          >
            <div
              v-for="workspace in workspaceOptions"
              :key="workspace.value"
              class="border border-gray-200 rounded-lg hover:bg-gray-50"
              :data-workspace-id="workspace.value"
            >
              <div class="flex items-center justify-between p-3">
                <div class="flex items-center space-x-4">
                  <span class="font-medium">{{ workspace.text }}</span>
                  <!-- 默认 badge 跟 is_default（租户默认工作空间），不是 is_current（当前选中） -->
                  <span
                    v-if="workspace.is_default"
                    class="px-2 py-0.5 bg-primary/10 text-primary rounded text-xs"
                    data-testid="workspace-default-badge"
                  >默认</span>
                  <span
                    class="text-xs text-gray-400"
                    data-alias="task-archive-tier"
                    data-testid="task-archive-tier-label"
                    :title="archiveTierLabel(workspace.task_archive_tier)"
                  >存档：{{ archiveTierLabel(workspace.task_archive_tier) }}</span>
                </div>
                <WorkspaceSettingsTaskPanelActions
                  :workspace="workspace"
                  @archive="openTaskArchiveSettings"
                  @deliverable="handleDeliverableSystemSettings"
                  @progress="handleProgressSystemSettings"
                  @task-kind="(ws) => optionsModalsRef?.openTaskKindOptionsSettings(ws)"
                  @code-lang="(ws) => optionsModalsRef?.openCodeLangOptionsSettings(ws)"
                  @create-fields="(ws) => optionsModalsRef?.openCreateTaskFieldSettings(ws)"
                  @access="handleAccessManagement"
                  @feature-params="openFeatureParamsSettings"
                  @machine-policy="openMachinePolicySettings"
                  @edit="editWorkspace"
                  @delete="deleteWorkspace"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>

    <TaskArchiveSettingsModal
      :show="showTaskArchiveModal"
      :tier="taskArchiveTierDraft"
      :saving="savingTaskArchive"
      @update:tier="taskArchiveTierDraft = $event"
      @close="showTaskArchiveModal = false"
      @save="saveTaskArchiveTier"
    />

    <WorkspaceCreateEditModal
      :show="showCreateWorkspaceModal"
      :editing="Boolean(editingWorkspace)"
      :form="newWorkspace"
      :saving="savingWorkspace"
      @save="saveWorkspace"
      @cancel="showCreateWorkspaceModal = false; resetNewWorkspace()"
    />

    <!-- 添加面板模态框 -->
    <WorkspaceTaskPanelAddEditModal
      :show="showAddModal"
      mode="add"
      :panel="newPanel"
      :available-colors="availableColors"
      @save="submitAddPanel"
      @cancel="cancelAddPanel"
    />

    <!-- 编辑面板模态框 -->
    <WorkspaceTaskPanelAddEditModal
      :show="showEditModal"
      mode="edit"
      :panel="editingPanel"
      :available-colors="availableColors"
      @save="submitEditPanel"
      @cancel="cancelEditPanel"
    />

    <!-- 交付物体系设置模态框 -->
    <div v-if="showDeliverableSystemSettingsModal" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <DeliverableSystemSettings
          :workspaceId="currentDeliverableSystemWorkspaceId"
          :tenantId="getCurrentTenantId()"
          @close="handleCloseDeliverableSystemSettings"
        />
      </div>
    </div>

    <!-- 进度体系设置模态框 -->
    <ProgressSystemSettingsModal
      :show="showProgressSystemSettingsModal"
      :workspaceId="currentProgressSystemWorkspaceId"
      :tenantId="getCurrentTenantId()"
      @close="handleCloseProgressSystemSettings"
    />

    <!-- 访问管理模态框 -->
    <AccessManagementModal
      :show="showAccessManagementModal"
      :workspaceId="currentAccessManagementWorkspaceId"
      :tenantId="getCurrentTenantId()"
      @close="handleCloseAccessManagement"
    />

    <WorkspaceMachinePolicyModal
      :show="showMachinePolicyModal"
      :tenant-id="getCurrentTenantId()"
      :workspace-id="machinePolicyWorkspaceId"
      :workspace-name="machinePolicyWorkspaceName"
      @close="showMachinePolicyModal = false"
    />

    <WorkspaceSettingsTaskPanelOptionsModals
      ref="optionsModalsRef"
      :tenant-id="getCurrentTenantId()"
    />

  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DeliverableSystemSettings from '../components/DeliverableSystemSettings.vue'
import ProgressSystemSettings from '../components/ProgressSystemSettings.vue'
import ProgressSystemSettingsModal from '../components/ProgressSystemSettingsModal.vue'
import AccessManagementModal from '../components/AccessManagementModal.vue'
import WorkspaceMachinePolicyModal from '../components/WorkspaceMachinePolicyModal.vue'
import WorkspaceSettingsTaskPanelActions from '../components/WorkspaceSettingsTaskPanelActions.vue'
import WorkspaceCreateEditModal from '../components/WorkspaceCreateEditModal.vue'
import WorkspaceSettingsTaskPanelOptionsModals from '../components/WorkspaceSettingsTaskPanelOptionsModals.vue'
import TaskArchiveSettingsModal from '../components/TaskArchiveSettingsModal.vue'
import WorkspaceTaskPanelAddEditModal from '../components/WorkspaceTaskPanelAddEditModal.vue'
import { archiveTierLabel } from '../utils/taskArchiveTiers.js'
import { useTenantPageAccess } from '../composables/useTenantPageAccess.js'
import { useWorkspaceSettingsTaskPanel } from '../composables/useWorkspaceSettingsTaskPanel.js'
import { useWorkspaceSettingsModals } from '../composables/useWorkspaceSettingsModals.js'
import TenantPageAccessEmpty from '../components/TenantPageAccessEmpty.vue'

const { accessAllowed } = useTenantPageAccess('settings.task_panel')

const route = useRoute()
const router = useRouter()

const getCurrentTenantId = () => route.params.tenant || ''

const {
  // 工作空间
  workspaceOptions,
  loadingWorkspaces,
  showCreateWorkspaceModal,
  newWorkspace,
  editingWorkspace,
  savingWorkspace,
  saveWorkspace,
  editWorkspace,
  deleteWorkspace,
  resetNewWorkspace,
  // 任务面板
  showAddModal,
  showEditModal,
  newPanel,
  editingPanel,
  availableColors,
  submitAddPanel,
  cancelAddPanel,
  submitEditPanel,
  cancelEditPanel,
  loadWorkspaces,
} = useWorkspaceSettingsTaskPanel({ tenantId: getCurrentTenantId })

const {
  // 各设置模态
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
} = useWorkspaceSettingsModals({
  tenantId: getCurrentTenantId,
  router,
  loadWorkspaces,
})

onMounted(() => {
  loadWorkspaces()
})
</script>
<style scoped>
/* 组件内样式 */
</style>
