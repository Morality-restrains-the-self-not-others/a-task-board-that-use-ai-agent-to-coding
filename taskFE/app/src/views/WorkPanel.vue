<template>
  <div data-alias="view-work-panel" class="h-full min-h-0 flex flex-col overflow-hidden bg-gray-50 font-sans">
    <!-- 主内容区域：填满路由区剩余高度，看板底边可贴浏览器底部 -->
    <div class="flex-1 min-h-0 overflow-hidden flex flex-col">
      <div class="flex flex-col h-full min-h-0 w-full">
        <!-- 上方组件：工作面板头部 -->
        <WorkPanelHeader
          v-bind="headerBind"
          :access-code="String(route.query.accessCode || '')"
          class="shrink-0"
          @create-task="handleCreateDeliverable"
          @filter="handleFilter"
          @workspace-switched="handleWorkspaceSwitched"
          @machine-filter="handleMachineRuntimeFilter"
          @clear-machine-filter="clearMachineRuntimeFilter"
          @toggle-access-filter-panel="toggleAccessFilterPanel"
          @close-access-filter-panel="closeAccessFilterPanel"
          @access-filter-tab="setAccessFilterTab"
          @access-filter-select="selectAccessSubject"
          @clear-access-filter="clearAccessFilter"
        />

        <WorkPanelMachineFilterMismatchHint
          class="shrink-0"
          :visible="machineFilterMismatch"
          @clear="clearMachineRuntimeFilter"
          @reload="reloadBoardAfterMismatch"
        />

        <!-- 下方组件：显示交付物面板（占满剩余高度） -->
        <div class="flex-1 min-h-0 flex flex-col overflow-hidden">
          <TaskPanel 
            :tenant-id="tenantId"
            :workspace-id="currentWorkspace?.id"
            :currentWorkspace="currentWorkspace"
            :taskStatuses="taskStatuses"
            :taskTypes="taskTypes"
            :todos="filteredTodos"
            :runtime-indicators="runtimeIndicators"
            :collaborator-name-by-id="collaboratorNameById"
            :filterOptions="filterOptions"
            :deliverable-filter-bars="boardDeliverableFilterBars"
            @update:deliverableFilterBars="deliverableFilterBars = $event"
            @task-clicked="handleTaskClicked"
            @tasks-updated="fetchTodos"
          />
        </div>
      </div>
    </div>
    
    <WorkPanelTaskDetailModal
      :show="showTaskDetailModal"
      :task="selectedTask"
      :tenant-id="tenantId"
      :workspace-id="currentWorkspace.id || 'default'"
      :new-tab-href="taskDetailNewTabHref"
      @close="closeTaskDetailModal"
      @task-updated="fetchTodos"
    />

    <!-- 创建交付物模态框 -->
    <CreateTaskModal 
      :show="showCreateTaskModal"
      :editingTask="editingTask"
      :taskStatuses="taskStatuses"
      :taskTypes="taskTypes"
      :taskKindOptions="taskKindOptions"
      :codeLangOptions="codeLangOptions"
      :fieldSettings="createTaskFieldSettings"
      :todos="todos"
      :projects="projects"
      :installedImages="installedImages"
      :isProjectsLoading="isProjectsLoading"
      :projectsError="projectsError"
      :isInstalledImagesLoading="isInstalledImagesLoading"
      :installedImagesError="installedImagesError"
      :isTaskTypesLoading="isTaskTypesLoading"
      :taskTypesError="taskTypesError"
      :currentWorkspace="currentWorkspace"
      :tenantId="tenantId"
      :companyUserName="companyUserName"
      @close="closeCreateTaskModal"
      @submit="submitDeliverableForm"
    />
    
    <WorkPanelFilterModal
      :show="showFilterModal"
      :filter-options="filterOptions"
      :access-filter="accessFilter"
      :access-people="accessPeople"
      :access-groups="accessGroups"
      :access-subjects-loading="accessSubjectsLoading"
      :access-filter-tab="accessFilterTab"
      @close="closeFilterModal"
      @reset="resetFilter"
      @apply="applyFilter"
      @update:search="filterOptions.search = $event"
      @update:priority="filterOptions.priority = $event"
      @access-filter-tab="setAccessFilterTab"
      @access-filter-select="selectAccessSubject"
      @clear-access-filter="clearAccessFilter"
    />
  </div>
</template>

<script setup>
/* @alias:view-work-panel */
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import WorkPanelHeader from './WorkPanelHeader.vue'
import TaskPanel from './TaskPanel.vue'
import CreateTaskModal from '../components/CreateTaskModal.vue'
import WorkPanelMachineFilterMismatchHint from '../components/WorkPanelMachineFilterMismatchHint.vue'
import WorkPanelTaskDetailModal from '../components/WorkPanelTaskDetailModal.vue'
import WorkPanelFilterModal from '../components/WorkPanelFilterModal.vue'
import { getStoredUserId } from '../utils/sessionUserIdUtils.js'
import { apiFetch } from '../utils/apiUtils.js'
import {
  normalizeListPayload,
  parseJsonSafe,
  warnNetworkFailure,
  warnOptionalApiFailure
} from '../utils/workPanelApiUtils.js'
import {
  buildBranchStrategyProjectMappings as buildBranchStrategyProjectMappingsFromWorkspace,
  buildDefaultWorkBranchName as buildDefaultWorkBranchNameRaw,
} from '../utils/workPanelBranchHelpers.js'
import { DEFAULT_TASK_STATUSES } from '../utils/workPanelKanbanUtils.js'
import { buildCollaboratorNameById } from '../utils/taskCardPeopleDisplay.js'
import { defaultDeliverableFilterBars } from '../utils/workPanelDeliverableFilterBars.js'
import { useWorkPanelFilterPersistence } from '../composables/useWorkPanelFilterPersistence.js'
import { useWorkPanelBoardFilters } from '../composables/useWorkPanelBoardFilters.js'
import { useWorkPanelTaskStatusSse } from '../composables/useWorkPanelTaskStatusSse.js'
import { useWorkPanelTaskIdDeepLink } from '../composables/useWorkPanelTaskIdDeepLink.js'
import { useWorkPanelDeliverableForm } from '../composables/useWorkPanelDeliverableForm.js'
import { useWorkPanelTaskMetaOptions } from '../composables/useWorkPanelTaskMetaOptions.js'
import {
  fetchCollaborators as fetchCollaboratorsRemote,
  fetchCurrentCompanyUserName as fetchCurrentCompanyUserNameRemote,
  fetchInstalledImages as fetchInstalledImagesRemote,
  fetchProjects as fetchProjectsRemote,
  fetchTaskStatuses as fetchTaskStatusesRemote,
  fetchTaskTypes as fetchTaskTypesRemote,
  fetchTodos as fetchTodosRemote,
  shouldShowEmptyTaskHint,
} from '../utils/workPanelDataFetch.js'
import { formatApiErrorMessage } from '../utils/workPanelFormat.js'
import { extractTraceId } from '../utils/traceId.js'
import modalService from '../utils/modalService.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { loadLegacyBoardModalAndComments } from '../utils/workPanelLegacyBoard.js'
import {
  buildProfileHref,
  promptNavigateOnTenantAccessDenied,
} from '../utils/tenantAccessDeniedPrompt.js'
import { resolveWorkPanelInitialWorkspace } from '../utils/workPanelWorkspaceInit.js'

// 响应式状态
const route = useRoute()
const router = useRouter()

/** 弹窗内「新标签打开任务详情」须带上与工作区列表页相同的 query，避免 accessCode 丢失。 */
const taskDetailNewTabHref = computed(() => {
  if (!tenantId.value || !selectedTask.value?.id || !currentWorkspace.value?.id) {
    return '#'
  }
  const path = `/tenant/${tenantId.value}/workspace/${currentWorkspace.value.id}/task-detail/${selectedTask.value.id}/`
  const q = new URLSearchParams()
  const rq = route.query || {}
  for (const k of ['accessCode', 'github']) {
    const v = rq[k]
    if (v == null || v === '') continue
    q.set(k, Array.isArray(v) ? String(v[0]) : String(v))
  }
  const s = q.toString()
  return s ? `${path}?${s}` : path
})

const currentWorkspace = ref({
  id: null,
  name: '默认工作空间'
})

const tenantId = ref(null)
const taskStatuses = ref([])
const taskTypes = ref([])
const isTaskTypesLoading = ref(false)
const taskTypesError = ref(null)
const projects = ref([])
const isProjectsLoading = ref(false)
const projectsError = ref(null)
const installedImages = ref([])
const isInstalledImagesLoading = ref(false)
const installedImagesError = ref(null)
const todos = ref([])
const todosError = ref(null)
const todosErrorTraceId = ref('')
const showTaskDetailModal = ref(false)
const showCreateTaskModal = ref(false)
const selectedTask = ref(null)
const editingTask = ref(null)
const filterOptions = ref({
  priority: null,
  search: ''
})
const deliverableFilterBars = ref(defaultDeliverableFilterBars())
const collaborators = ref([])
const companyUserName = ref('')
const collaboratorNameById = computed(() => buildCollaboratorNameById(collaborators.value))

// 模态框状态
const showFilterModal = ref(false)

// 工作空间刷新触发器
const workspaceRefreshTrigger = ref(0)
const initialDataLoaded = ref(false)

const showEmptyTaskHint = computed(() => shouldShowEmptyTaskHint({
  workspaceId: currentWorkspace.value?.id,
  statusCount: taskStatuses.value.length,
  todoCount: todos.value.length,
  loadError: todosError.value,
}))

const {
  runtimeIndicators, filteredTodos, headerBind, accessFilter, accessPeople, accessGroups,
  accessSubjectsLoading, accessFilterTab, handleMachineRuntimeFilter, clearMachineRuntimeFilter,
  startMachineSummaryPolling, selectAccessSubject, hydrateAccessFilterPref, clearAccessFilter,
  toggleAccessFilterPanel, closeAccessFilterPanel, setAccessFilterTab, resetBoardFilters, refreshBoardFilterData,
  boardDeliverableFilterBars, machineFilterMismatch,
} = useWorkPanelBoardFilters({
  apiFetch, tenantId, currentWorkspace, todos, todosError, todosErrorTraceId,
  collaborators, workspaceRefreshTrigger, showEmptyTaskHint, deliverableFilterBars,
})

const { loadDeliverableFilterPrefs, pendingAccessFilterPref } = useWorkPanelFilterPersistence({
  apiFetch, tenantId, currentWorkspace, deliverableFilterBars, accessFilter, initialDataLoaded,
})

const wpDeps = () => ({
  tenantId, currentWorkspace, filterOptions, todos, todosError, todosErrorTraceId,
  taskStatuses, taskTypes, isTaskTypesLoading, taskTypesError, projects, isProjectsLoading, projectsError,
  installedImages, isInstalledImagesLoading, installedImagesError, collaborators,
  companyUserName, apiFetch, parseJsonSafe, normalizeListPayload,
  warnOptionalApiFailure, warnNetworkFailure, formatApiErrorMessage, extractTraceId,
})

const buildDefaultWorkBranchName = (presetType = 'feature', taskTitle = '', createdAt) =>
  buildDefaultWorkBranchNameRaw(presetType, taskTitle, companyUserName.value, createdAt)

const buildBranchStrategyProjectMappings = (task) =>
  buildBranchStrategyProjectMappingsFromWorkspace(task, projects.value)

const fetchTodos = async () => fetchTodosRemote(wpDeps())
const { notifyBoardReady: notifyWorkPanelSseReady } = useWorkPanelTaskStatusSse({
  tenantId,
  workspaceId: () => currentWorkspace.value?.id,
  todos,
  fetchTodos,
  refreshMachineSummary: refreshBoardFilterData,
})
const reloadBoardAfterMismatch = async () => {
  await fetchTodos()
  await refreshBoardFilterData()
}
const fetchTaskStatuses = async () => fetchTaskStatusesRemote(wpDeps())
const fetchTaskTypes = async () => fetchTaskTypesRemote(wpDeps())
const {
  taskKindOptions, codeLangOptions, createTaskFieldSettings, fetchTaskMetaOptions, resetTaskMetaOptions,
} = useWorkPanelTaskMetaOptions({
  apiFetch, tenantId, currentWorkspace, parseJsonSafe, warnOptionalApiFailure, warnNetworkFailure,
})
const fetchProjects = async () => fetchProjectsRemote(wpDeps())
const fetchInstalledImages = async () => fetchInstalledImagesRemote(wpDeps())
const fetchCollaborators = async () => fetchCollaboratorsRemote(wpDeps())

const initData = async () => {
  try {
    const url = new URL(window.location.href)
    const workspaceIdFromUrl = url.searchParams.get('workspace_id')
    const pathSegments = url.pathname.split('/').filter(segment => segment)
    let tenantIdFromUrl = null
    let workspaceIdFromPath = null
    if (pathSegments.length >= 2 && pathSegments[0] === 'tenant') {
      tenantIdFromUrl = pathSegments[1]
      if (pathSegments.length >= 4 && pathSegments[2] === 'workspace') {
        workspaceIdFromPath = pathSegments[3]
      }
    }
    const finalWorkspaceId = workspaceIdFromUrl || workspaceIdFromPath
    // OPT-20260810-032：localStorage 优先（getStoredUserId），HttpOnly cookie 场景下 JS 读不到 cookie 也能识别登录态
    const userId = getStoredUserId()
    let userData = null

    if (userId) {
      const tenantParam = tenantIdFromUrl ? `?tenant_id=${encodeURIComponent(tenantIdFromUrl)}` : ''
      const userResponse = await apiFetch(`/api/accounts/users/me/${tenantParam}`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (userResponse.ok) {
        userData = await parseJsonSafe(userResponse)
        if (!userData) warnOptionalApiFailure('用户信息(JSON 无效)', userResponse)
      } else {
        // 403/404：禁止静默 replace(profile)（易与「工作面板」导航形成循环回弹）。
        // 弹窗确认后整页跳转；取消则留在本页，继续用 URL 推断租户降级初始化。
        if (userResponse.status === 403 || userResponse.status === 404) {
          console.warn('[WorkPanel] /me/ 返回 %d，征求用户是否离开本页', userResponse.status)
          const outcome = await promptNavigateOnTenantAccessDenied({
            targetHref: buildProfileHref(getStoredUserId()),
          })
          if (outcome === 'navigated') return
          warnOptionalApiFailure('用户信息', userResponse)
        } else {
          warnOptionalApiFailure('用户信息', userResponse)
        }
      }
    } else {
      warnNetworkFailure('用户信息', new Error('Cookie 中无 userId，将仅用 URL 推断租户/工作空间'))
    }

    tenantId.value = tenantIdFromUrl
      || userData?.current_company?.id
      || userData?.companies?.[0]?.id
      || null
    await fetchCurrentCompanyUserNameRemote(wpDeps(), tenantId.value, userData)

    // OPT-20260816-047：URL workspace_id（查询参数/路径）优先，禁止用 /me/
    // current_workspace 覆盖后删除 query；仅 URL 未指定时回退用户默认工作空间。
    currentWorkspace.value = resolveWorkPanelInitialWorkspace({
      urlWorkspaceId: finalWorkspaceId,
      userWorkspace: userData?.current_workspace,
    })

    taskStatuses.value = DEFAULT_TASK_STATUSES.map((s) => ({ ...s }))

    if (currentWorkspace.value.id && currentWorkspace.value.id !== 'default') {
      await loadDeliverableFilterPrefs(currentWorkspace.value)
      await fetchTaskStatuses()
      await fetchTaskTypes()
      await fetchTaskMetaOptions()
      await fetchProjects()
      await fetchInstalledImages()
      await fetchCollaborators()
      await fetchTodos()
      notifyWorkPanelSseReady()
    }
  } catch (error) {
    warnNetworkFailure('初始化数据', error)
    // 显示提示消息，告知用户API请求失败
    modalService.alert('部分数据加载失败，已使用默认或 URL 中的工作空间信息', '提示', { autoClose: 3000 })
    // 即使API请求失败，也要尝试从URL中获取tenant_id和workspace_id
    const url = new URL(window.location.href)
    const pathSegments = url.pathname.split('/').filter(segment => segment)
    if (pathSegments.length >= 2 && pathSegments[0] === 'tenant') {
      tenantId.value = pathSegments[1]
      
      // 检查路径中是否有workspace_id
      if (pathSegments.length >= 4 && pathSegments[2] === 'workspace') {
        const workspaceIdFromPath = pathSegments[3]
        currentWorkspace.value = {
          id: workspaceIdFromPath,
          name: '工作空间'
        }
      } else {
        // 从查询参数中获取workspace_id
        const workspaceIdFromUrl = url.searchParams.get('workspace_id')
        if (workspaceIdFromUrl) {
          currentWorkspace.value = {
            id: workspaceIdFromUrl,
            name: '工作空间'
          }
        } else {
          // 没有找到工作空间，使用默认工作空间
          currentWorkspace.value = {
            id: null,
            name: '默认工作空间'
          }
        }
      }
    } else {
      // 没有找到工作空间，使用默认工作空间
      currentWorkspace.value = {
        id: null,
        name: '默认工作空间'
      }
    }
    
    // 使用默认数据
    taskStatuses.value = DEFAULT_TASK_STATUSES.map((s) => ({ ...s }))
    todos.value = []
  } finally {
    // 标记首轮初始化结束，供 workspace 切换事件去重
    initialDataLoaded.value = true
  }
}

const handleFilter = () => { showFilterModal.value = true; document.body.style.overflow = 'hidden' }
const applyFilter = () => { fetchTodos(); showFilterModal.value = false; document.body.style.overflow = '' }
const resetFilter = () => { filterOptions.value = { priority: null, search: '' }; clearAccessFilter(); fetchTodos() }
const closeFilterModal = () => { showFilterModal.value = false; document.body.style.overflow = '' }

const {
  handleCreateDeliverable,
  openEditTaskModal,
  submitDeliverableForm,
  closeCreateTaskModal,
} = useWorkPanelDeliverableForm({
  tenantId,
  currentWorkspace,
  todos,
  taskStatuses,
  taskTypes,
  projects,
  deliverableFilterBars,
  editingTask,
  showCreateTaskModal,
  fetchTodos,
  buildDefaultWorkBranchName,
  buildBranchStrategyProjectMappings,
})

// 方法：处理任务卡片点击
const handleTaskClicked = (task) => {
  selectedTask.value = task
  showTaskDetailModal.value = true
  document.body.style.overflow = 'hidden'
}

const { clearTaskIdQuery } = useWorkPanelTaskIdDeepLink({
  route,
  router,
  todos,
  openTask: handleTaskClicked,
})

const closeTaskDetailModal = () => {
  showTaskDetailModal.value = false
  selectedTask.value = null
  clearTaskIdQuery()
  document.body.style.overflow = ''
}

const deleteTask = async (taskId) => {
  try {
    await modalService.confirm('确定要删除这个任务吗？')
    const response = await apiFetch(
      `/api/tasks/todos/tenant_id/${tenantId.value}/workspace_id/${currentWorkspace.value?.id}/${taskId}/`,
      { method: 'DELETE', credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (response.ok) {
      await fetchTodos()
      return
    }
    warnOptionalApiFailure('删除任务', response)
    const errBody = await parseJsonSafe(response)
    showRequestError(formatApiErrorMessage(errBody) || `删除任务失败（HTTP ${response.status}）`, errBody)
  } catch (error) {
    if (!error) return
    warnNetworkFailure('删除任务', error)
    showRequestError(error?.message || '删除任务出错，请稍后重试', error)
  }
}

const handleWorkspaceSwitched = (workspace) => {
  const incoming = workspace?.id != null ? String(workspace.id) : ''
  const current = currentWorkspace.value?.id != null ? String(currentWorkspace.value.id) : ''
  if (initialDataLoaded.value && incoming && incoming === current) return
  currentWorkspace.value = workspace
  resetBoardFilters()
  taskTypes.value = []
  resetTaskMetaOptions()
  isTaskTypesLoading.value = false
  taskTypesError.value = null
  projects.value = []
  isProjectsLoading.value = false
  projectsError.value = null
  loadDeliverableFilterPrefs(workspace).then(async () => {
    await Promise.all([
      fetchTodos(), fetchTaskStatuses(), fetchTaskTypes(), fetchTaskMetaOptions(),
      fetchProjects(), fetchInstalledImages(),
    ])
    await fetchCollaborators()
    await refreshBoardFilterData()
    await hydrateAccessFilterPref(pendingAccessFilterPref.value)
    notifyWorkPanelSseReady()
  })
}
onMounted(async () => {
  await initData()
  await refreshBoardFilterData()
  await hydrateAccessFilterPref(pendingAccessFilterPref.value)
  startMachineSummaryPolling()
  await loadLegacyBoardModalAndComments()
  if (window.modalModule?.initModals) setTimeout(() => window.modalModule.initModals(), 200)
})
onUnmounted(() => { document.body.style.overflow = '' })
</script>
