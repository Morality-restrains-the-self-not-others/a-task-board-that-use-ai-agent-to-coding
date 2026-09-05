import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch, parseCompanyMembersResponse } from '../utils/apiUtils.js'

export function useBillingUsage() {
  const route = useRoute()
  const tenantId = computed(() => route.params.tenant)

  const filters = ref({
    startDate: '',
    endDate: '',
    billingUnitType: '',
    projectId: '',
    workspaceId: '',
    userId: '',
    taskId: ''
  })

  const billingUnitTypeOptions = ref([])
  const RECHARGE_UNIT_TYPES = new Set(['recharge'])

  const usageRecords = ref([])
  const isLoading = ref(false)
  const currentPage = ref(1)
  const pageSize = ref(20)
  const totalPages = ref(1)
  const totalCount = ref(0)

  const projectSearch = ref('')
  const selectedProjectName = ref('')
  const showProjectDropdown = ref(false)
  const projectOptions = ref([])
  const allProjects = ref([])
  let projectSearchTimeout = null

  const workspaceSearch = ref('')
  const selectedWorkspaceName = ref('')
  const showWorkspaceDropdown = ref(false)
  const workspaceOptions = ref([])
  const allWorkspaces = ref([])
  let workspaceSearchTimeout = null

  const userSearch = ref('')
  const selectedUserName = ref('')
  const showUserDropdown = ref(false)
  const userOptions = ref([])
  const allUsers = ref([])
  let userSearchTimeout = null

  const taskSearch = ref('')
  const selectedTaskName = ref('')
  const showTaskDropdown = ref(false)
  const taskOptions = ref([])
  const allTasks = ref([])
  let taskSearchTimeout = null

  const formatDate = (dateString) => {
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  const normalizeSearchText = (text) => String(text || '').trim().toLowerCase()

  const getNameAcronym = (name) => {
    const normalized = String(name || '').trim()
    if (!normalized) return ''
    const words = normalized
      .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
      .split(/[\s\-_./]+/)
      .filter(Boolean)
    return words.map((word) => word[0]).join('').toLowerCase()
  }

  const filterNamedOptions = (items, keyword = '') => {
    const normalizedKeyword = normalizeSearchText(keyword)
    if (!normalizedKeyword) {
      return items.slice(0, 20)
    }
    return items
      .filter((item) => {
        const itemName = normalizeSearchText(item.name)
        const itemId = normalizeSearchText(item.id)
        const acronym = getNameAcronym(item.name)
        return (
          itemName.includes(normalizedKeyword) ||
          itemId.includes(normalizedKeyword) ||
          acronym.includes(normalizedKeyword)
        )
      })
      .slice(0, 20)
  }

  const buildProjectOptions = (keyword = '') => filterNamedOptions(allProjects.value, keyword)
  const buildWorkspaceOptions = (keyword = '') => filterNamedOptions(allWorkspaces.value, keyword)

  const fetchBillingUnitTypeOptions = async () => {
    try {
      const response = await apiFetch(`/api/tenant/${tenantId.value}/billing/units/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' }
      })
      if (!response.ok) {
        billingUnitTypeOptions.value = []
        return
      }
      const units = await response.json()
      if (!Array.isArray(units)) {
        billingUnitTypeOptions.value = []
        return
      }
      const seen = new Set()
      billingUnitTypeOptions.value = units
        .filter((u) => u && u.is_active !== false && u.unit_type && !RECHARGE_UNIT_TYPES.has(u.unit_type))
        .filter((u) => {
          if (seen.has(u.unit_type)) return false
          seen.add(u.unit_type)
          return true
        })
        .map((u) => ({ value: u.unit_type, label: u.name || u.unit_type }))
        .sort((a, b) => String(a.label).localeCompare(String(b.label), 'zh-CN'))
    } catch (error) {
      console.error('获取计费单元类型失败:', error)
      billingUnitTypeOptions.value = []
    }
  }

  const fetchProjects = async () => {
    try {
      const response = await apiFetch(`/api/projects/tenant_id/${tenantId.value}`, {
        credentials: 'include',
        headers: { Accept: 'application/json' }
      })
      if (!response.ok) return
      const data = await response.json()
      allProjects.value = Array.isArray(data)
        ? data.map((project) => ({
            id: String(project.id),
            name: project.name || String(project.id)
          }))
        : []
      projectOptions.value = buildProjectOptions(projectSearch.value)
    } catch (error) {
      console.error('获取项目列表失败:', error)
    }
  }

  const fetchWorkspaces = async () => {
    try {
      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId.value}?mine=1`, {
        credentials: 'include',
        headers: { Accept: 'application/json' }
      })
      if (!response.ok) return
      const data = await response.json()
      allWorkspaces.value = Array.isArray(data)
        ? data.map((workspace) => ({
            id: String(workspace.id),
            name: workspace.name || String(workspace.id)
          }))
        : []
      workspaceOptions.value = buildWorkspaceOptions(workspaceSearch.value)
    } catch (error) {
      console.error('获取工作空间列表失败:', error)
    }
  }

  const fetchUsers = async () => {
    try {
      const response = await apiFetch(
        `/api/tenant/${tenantId.value}/accounts/members/company_members/`,
        { credentials: 'include', headers: { Accept: 'application/json' } }
      )
      if (!response.ok) return
      const data = await response.json()
      const { members } = parseCompanyMembersResponse(data)
      allUsers.value = members
        .map((m) => {
          const userId = m.user_id || m.user?.id || m.account_user_id || m.id
          if (userId == null || userId === '') return null
          return {
            id: String(userId),
            name: m.member_name || m.username || m.email || String(userId)
          }
        })
        .filter(Boolean)
      userOptions.value = filterNamedOptions(allUsers.value, userSearch.value)
    } catch (error) {
      console.error('获取成员列表失败:', error)
    }
  }

  const fetchTasks = async (query = '') => {
    try {
      const params = new URLSearchParams()
      const q = String(query || '').trim()
      if (q) params.set('q', q)
      if (filters.value.workspaceId) params.set('workspace_id', filters.value.workspaceId)
      params.set('limit', '50')
      const response = await apiFetch(
        `/api/tasks/search/tenant_id/${tenantId.value}/?${params}`,
        { credentials: 'include', headers: { Accept: 'application/json' } }
      )
      if (!response.ok) return
      const data = await response.json()
      const rows = Array.isArray(data?.results) ? data.results : []
      allTasks.value = rows.map((t) => ({
        id: String(t.id),
        name: t.title || t.name || String(t.id)
      }))
      taskOptions.value = allTasks.value
    } catch (error) {
      console.error('获取任务列表失败:', error)
    }
  }

  const searchProjects = () => {
    if (filters.value.projectId && projectSearch.value !== selectedProjectName.value) {
      filters.value.projectId = ''
      selectedProjectName.value = ''
    }
    if (projectSearchTimeout) clearTimeout(projectSearchTimeout)
    projectSearchTimeout = setTimeout(async () => {
      if (!allProjects.value.length) await fetchProjects()
      projectOptions.value = buildProjectOptions(projectSearch.value)
      showProjectDropdown.value = true
    }, 200)
  }

  const openProjectDropdown = async () => {
    if (!allProjects.value.length) await fetchProjects()
    projectOptions.value = buildProjectOptions(projectSearch.value)
    showProjectDropdown.value = true
  }

  const selectProject = (option) => {
    filters.value.projectId = option.id
    selectedProjectName.value = option.name
    projectSearch.value = option.name
    showProjectDropdown.value = false
    projectOptions.value = []
  }

  const searchWorkspaces = () => {
    if (filters.value.workspaceId && workspaceSearch.value !== selectedWorkspaceName.value) {
      filters.value.workspaceId = ''
      selectedWorkspaceName.value = ''
    }
    if (workspaceSearchTimeout) clearTimeout(workspaceSearchTimeout)
    workspaceSearchTimeout = setTimeout(async () => {
      if (!allWorkspaces.value.length) await fetchWorkspaces()
      workspaceOptions.value = buildWorkspaceOptions(workspaceSearch.value)
      showWorkspaceDropdown.value = true
    }, 200)
  }

  const openWorkspaceDropdown = async () => {
    if (!allWorkspaces.value.length) await fetchWorkspaces()
    workspaceOptions.value = buildWorkspaceOptions(workspaceSearch.value)
    showWorkspaceDropdown.value = true
  }

  const selectWorkspace = (option) => {
    filters.value.workspaceId = option.id
    selectedWorkspaceName.value = option.name
    workspaceSearch.value = option.name
    showWorkspaceDropdown.value = false
    workspaceOptions.value = []
    allTasks.value = []
    if (filters.value.taskId) {
      filters.value.taskId = ''
      selectedTaskName.value = ''
      taskSearch.value = ''
    }
  }

  const searchUsers = () => {
    if (filters.value.userId && userSearch.value !== selectedUserName.value) {
      filters.value.userId = ''
      selectedUserName.value = ''
    }
    if (userSearchTimeout) clearTimeout(userSearchTimeout)
    userSearchTimeout = setTimeout(async () => {
      if (!allUsers.value.length) await fetchUsers()
      userOptions.value = filterNamedOptions(allUsers.value, userSearch.value)
      showUserDropdown.value = true
    }, 200)
  }

  const openUserDropdown = async () => {
    if (!allUsers.value.length) await fetchUsers()
    userOptions.value = filterNamedOptions(allUsers.value, userSearch.value)
    showUserDropdown.value = true
  }

  const selectUser = (option) => {
    filters.value.userId = option.id
    selectedUserName.value = option.name
    userSearch.value = option.name
    showUserDropdown.value = false
    userOptions.value = []
  }

  const searchTasks = () => {
    if (filters.value.taskId && taskSearch.value !== selectedTaskName.value) {
      filters.value.taskId = ''
      selectedTaskName.value = ''
    }
    if (taskSearchTimeout) clearTimeout(taskSearchTimeout)
    taskSearchTimeout = setTimeout(async () => {
      await fetchTasks(taskSearch.value)
      showTaskDropdown.value = true
    }, 200)
  }

  const openTaskDropdown = async () => {
    await fetchTasks(taskSearch.value)
    showTaskDropdown.value = true
  }

  const selectTask = (option) => {
    filters.value.taskId = option.id
    selectedTaskName.value = option.name
    taskSearch.value = option.name
    showTaskDropdown.value = false
    taskOptions.value = []
  }

  const handleClickOutside = (event) => {
    if (!event.target.closest('.project-filter')) {
      showProjectDropdown.value = false
    }
    if (!event.target.closest('.workspace-filter')) {
      showWorkspaceDropdown.value = false
    }
    if (!event.target.closest('.user-filter')) {
      showUserDropdown.value = false
    }
    if (!event.target.closest('.task-filter')) {
      showTaskDropdown.value = false
    }
  }

  const buildQueryParams = () => {
    const params = new URLSearchParams()
    if (filters.value.startDate) params.append('start_date', filters.value.startDate)
    if (filters.value.endDate) params.append('end_date', filters.value.endDate)
    if (filters.value.billingUnitType) {
      params.append('billing_unit_type', filters.value.billingUnitType)
    }
    if (filters.value.projectId) params.append('project_id', filters.value.projectId)
    if (filters.value.workspaceId) params.append('workspace_id', filters.value.workspaceId)
    if (filters.value.userId) params.append('user_id', filters.value.userId)
    if (filters.value.taskId) params.append('task_id', filters.value.taskId)
    params.append('page', currentPage.value)
    params.append('page_size', pageSize.value)
    return params.toString()
  }

  const parsePagedList = (data) => {
    if (Array.isArray(data)) {
      return {
        results: data,
        total: data.length,
        totalPages: Math.max(1, Math.ceil(data.length / pageSize.value) || 1)
      }
    }
    const results = Array.isArray(data?.results) ? data.results : []
    const total = Number(data?.total ?? 0)
    const size = Number(data?.page_size ?? pageSize.value) || pageSize.value
    return {
      results,
      total,
      totalPages: Math.max(1, Math.ceil(total / size) || 1)
    }
  }

  const fetchUsageRecords = async () => {
    isLoading.value = true
    try {
      const queryParams = buildQueryParams()
      const response = await apiFetch(
        `/api/tenant/${tenantId.value}/billing/usages/?${queryParams}`,
        {
          credentials: 'include',
          headers: { Accept: 'application/json' }
        }
      )
      if (response.ok) {
        const data = await response.json()
        const parsed = parsePagedList(data)
        usageRecords.value = parsed.results
        totalCount.value = parsed.total
        totalPages.value = parsed.totalPages
      } else {
        console.error('获取使用记录失败')
      }
    } catch (error) {
      console.error('获取使用记录出错:', error)
    } finally {
      isLoading.value = false
    }
  }

  const applyFilters = () => {
    currentPage.value = 1
    fetchUsageRecords()
  }

  const resetFilters = () => {
    filters.value = {
      startDate: '',
      endDate: '',
      billingUnitType: '',
      projectId: '',
      workspaceId: '',
      userId: '',
      taskId: ''
    }
    projectSearch.value = ''
    selectedProjectName.value = ''
    projectOptions.value = []
    showProjectDropdown.value = false
    workspaceSearch.value = ''
    selectedWorkspaceName.value = ''
    workspaceOptions.value = []
    showWorkspaceDropdown.value = false
    userSearch.value = ''
    selectedUserName.value = ''
    userOptions.value = []
    showUserDropdown.value = false
    taskSearch.value = ''
    selectedTaskName.value = ''
    taskOptions.value = []
    showTaskDropdown.value = false
    currentPage.value = 1
    fetchUsageRecords()
  }

  const changePage = (page) => {
    if (page < 1 || page > totalPages.value) return
    currentPage.value = page
    fetchUsageRecords()
  }

  onMounted(() => {
    fetchBillingUnitTypeOptions()
    fetchProjects()
    fetchWorkspaces()
    fetchUsers()
    fetchUsageRecords()
    document.addEventListener('click', handleClickOutside)
  })

  onUnmounted(() => {
    document.removeEventListener('click', handleClickOutside)
    if (projectSearchTimeout) clearTimeout(projectSearchTimeout)
    if (workspaceSearchTimeout) clearTimeout(workspaceSearchTimeout)
    if (userSearchTimeout) clearTimeout(userSearchTimeout)
    if (taskSearchTimeout) clearTimeout(taskSearchTimeout)
  })

  return {
    tenantId,
    filters,
    billingUnitTypeOptions,
    usageRecords,
    isLoading,
    currentPage,
    totalPages,
    totalCount,
    projectSearch,
    selectedProjectName,
    showProjectDropdown,
    projectOptions,
    workspaceSearch,
    selectedWorkspaceName,
    showWorkspaceDropdown,
    workspaceOptions,
    userSearch,
    selectedUserName,
    showUserDropdown,
    userOptions,
    taskSearch,
    selectedTaskName,
    showTaskDropdown,
    taskOptions,
    formatDate,
    searchProjects,
    openProjectDropdown,
    selectProject,
    searchWorkspaces,
    openWorkspaceDropdown,
    selectWorkspace,
    searchUsers,
    openUserDropdown,
    selectUser,
    searchTasks,
    openTaskDropdown,
    selectTask,
    applyFilters,
    resetFilters,
    changePage
  }
}
