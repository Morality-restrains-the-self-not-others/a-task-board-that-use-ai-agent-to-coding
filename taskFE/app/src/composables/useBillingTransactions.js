import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch, parseCompanyMembersResponse } from '../utils/apiUtils.js'

export function useBillingTransactions() {
  const route = useRoute()
  const tenantId = computed(() => route.params.tenant)

  const filters = ref({
    startDate: '',
    endDate: '',
    projectId: '',
    userId: '',
    workspaceId: '',
    taskId: '',
    transactionType: '',
    billingUnitType: '',
    pointsSourceType: ''
  })

  const billingUnitTypeOptions = ref([])
  const rechargeSourceOptions = [
    { value: 'user_recharge_paypal', label: 'PayPal 支付' },
    { value: 'user_recharge_wechat', label: '微信支付' },
    { value: 'user_recharge_admin', label: '管理员直充' },
    { value: 'user_recharge', label: '用户支付（历史）' },
    { value: 'admin_grant', label: '后台赠送' },
    { value: 'promotion', label: '活动赠送' },
    { value: 'adjustment', label: '人工调账' }
  ]
  const RECHARGE_UNIT_TYPES = new Set(['recharge'])

  const transactions = ref([])
  const isLoading = ref(false)
  const currentPage = ref(1)
  const pageSize = ref(20)
  const totalPages = ref(1)
  const totalCount = ref(0)

  const projectSearch = ref('')
  const userSearch = ref('')
  const workspaceSearch = ref('')
  const taskSearch = ref('')

  const projectOptions = ref([])
  const userOptions = ref([])
  const workspaceOptions = ref([])
  const taskOptions = ref([])

  const allProjects = ref([])
  const allUsers = ref([])
  const allWorkspaces = ref([])
  const allTasks = ref([])

  const showProjectDropdown = ref(false)
  const showUserDropdown = ref(false)
  const showWorkspaceDropdown = ref(false)
  const showTaskDropdown = ref(false)

  const selectedProjectName = ref('')
  const selectedUserName = ref('')
  const selectedWorkspaceName = ref('')
  const selectedTaskName = ref('')

  let projectSearchTimeout = null
  let userSearchTimeout = null
  let workspaceSearchTimeout = null
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

  const getTransactionTypeText = (type) => {
    const typeMap = {
      recharge: '入账',
      consumption: '消耗',
      refund: '退款'
    }
    return typeMap[type] || type
  }

  const normalizeSearchText = (text) => String(text || '').trim().toLowerCase()

  const getNameAcronym = (name) => {
    const normalized = String(name || '').trim()
    if (!normalized) return ''
    return normalized
      .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
      .split(/[\s\-_./]+/)
      .filter(Boolean)
      .map((word) => word[0])
      .join('')
      .toLowerCase()
  }

  const filterNamedOptions = (items, keyword = '') => {
    const normalizedKeyword = normalizeSearchText(keyword)
    if (!normalizedKeyword) return items.slice(0, 20)
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

  const buildQueryParams = () => {
    const params = new URLSearchParams()
    if (filters.value.startDate) params.append('start_date', filters.value.startDate)
    if (filters.value.endDate) params.append('end_date', filters.value.endDate)
    if (filters.value.projectId) params.append('project_id', filters.value.projectId)
    if (filters.value.userId) params.append('user_id', filters.value.userId)
    if (filters.value.workspaceId) params.append('workspace_id', filters.value.workspaceId)
    if (filters.value.taskId) params.append('task_id', filters.value.taskId)
    if (filters.value.transactionType) params.append('transaction_type', filters.value.transactionType)
    if (filters.value.billingUnitType) params.append('billing_unit_type', filters.value.billingUnitType)
    if (filters.value.pointsSourceType) params.append('points_source_type', filters.value.pointsSourceType)
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
      console.error('获取消耗（元）分类失败:', error)
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
        ? data.map((p) => ({ id: String(p.id), name: p.name || String(p.id) }))
        : []
      projectOptions.value = filterNamedOptions(allProjects.value, projectSearch.value)
    } catch (error) {
      console.error('获取项目列表失败:', error)
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

  const fetchWorkspaces = async () => {
    try {
      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId.value}?mine=1`, {
        credentials: 'include',
        headers: { Accept: 'application/json' }
      })
      if (!response.ok) return
      const data = await response.json()
      allWorkspaces.value = Array.isArray(data)
        ? data.map((w) => ({ id: String(w.id), name: w.name || String(w.id) }))
        : []
      workspaceOptions.value = filterNamedOptions(allWorkspaces.value, workspaceSearch.value)
    } catch (error) {
      console.error('获取工作空间列表失败:', error)
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

  const fetchTransactions = async () => {
    isLoading.value = true
    try {
      const queryParams = buildQueryParams()
      const response = await apiFetch(
        `/api/tenant/${tenantId.value}/billing/transactions/list_filtered/?${queryParams}`,
        { credentials: 'include', headers: { Accept: 'application/json' } }
      )
      if (response.ok) {
        const parsed = parsePagedList(await response.json())
        transactions.value = parsed.results
        totalCount.value = parsed.total
        totalPages.value = parsed.totalPages
      } else {
        console.error('获取交易记录失败')
      }
    } catch (error) {
      console.error('获取交易记录出错:', error)
    } finally {
      isLoading.value = false
    }
  }

  const bindSearch = (filtersKey, searchRef, selectedNameRef, allRef, optionsRef, showRef, timeoutRef, fetchFn) => {
    if (filters.value[filtersKey] && searchRef.value !== selectedNameRef.value) {
      filters.value[filtersKey] = ''
      selectedNameRef.value = ''
    }
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    timeoutRef.current = setTimeout(async () => {
      if (!allRef.value.length) await fetchFn()
      optionsRef.value = filterNamedOptions(allRef.value, searchRef.value)
      showRef.value = true
    }, 200)
  }

  const projectTimeout = { current: null }
  const userTimeout = { current: null }
  const workspaceTimeout = { current: null }
  const taskTimeout = { current: null }

  const searchProjects = () =>
    bindSearch('projectId', projectSearch, selectedProjectName, allProjects, projectOptions, showProjectDropdown, projectTimeout, fetchProjects)
  const searchUsers = () =>
    bindSearch('userId', userSearch, selectedUserName, allUsers, userOptions, showUserDropdown, userTimeout, fetchUsers)
  const searchWorkspaces = () =>
    bindSearch('workspaceId', workspaceSearch, selectedWorkspaceName, allWorkspaces, workspaceOptions, showWorkspaceDropdown, workspaceTimeout, fetchWorkspaces)
  const searchTasks = () => {
    if (filters.value.taskId && taskSearch.value !== selectedTaskName.value) {
      filters.value.taskId = ''
      selectedTaskName.value = ''
    }
    if (taskTimeout.current) clearTimeout(taskTimeout.current)
    taskTimeout.current = setTimeout(async () => {
      await fetchTasks(taskSearch.value)
      showTaskDropdown.value = true
    }, 200)
  }

  const openProjectDropdown = async () => {
    if (!allProjects.value.length) await fetchProjects()
    projectOptions.value = filterNamedOptions(allProjects.value, projectSearch.value)
    showProjectDropdown.value = true
  }
  const openUserDropdown = async () => {
    if (!allUsers.value.length) await fetchUsers()
    userOptions.value = filterNamedOptions(allUsers.value, userSearch.value)
    showUserDropdown.value = true
  }
  const openWorkspaceDropdown = async () => {
    if (!allWorkspaces.value.length) await fetchWorkspaces()
    workspaceOptions.value = filterNamedOptions(allWorkspaces.value, workspaceSearch.value)
    showWorkspaceDropdown.value = true
  }
  const openTaskDropdown = async () => {
    await fetchTasks(taskSearch.value)
    showTaskDropdown.value = true
  }

  const selectProject = (option) => {
    filters.value.projectId = option.id
    selectedProjectName.value = option.name
    projectSearch.value = option.name
    showProjectDropdown.value = false
    projectOptions.value = []
  }
  const selectUser = (option) => {
    filters.value.userId = option.id
    selectedUserName.value = option.name
    userSearch.value = option.name
    showUserDropdown.value = false
    userOptions.value = []
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
  const selectTask = (option) => {
    filters.value.taskId = option.id
    selectedTaskName.value = option.name
    taskSearch.value = option.name
    showTaskDropdown.value = false
    taskOptions.value = []
  }

  const applyFilters = () => {
    currentPage.value = 1
    fetchTransactions()
  }

  const resetFilters = () => {
    filters.value = {
      startDate: '',
      endDate: '',
      projectId: '',
      userId: '',
      workspaceId: '',
      taskId: '',
      transactionType: '',
      billingUnitType: '',
      pointsSourceType: ''
    }
    projectSearch.value = ''
    userSearch.value = ''
    workspaceSearch.value = ''
    taskSearch.value = ''
    selectedProjectName.value = ''
    selectedUserName.value = ''
    selectedWorkspaceName.value = ''
    selectedTaskName.value = ''
    showProjectDropdown.value = false
    showUserDropdown.value = false
    showWorkspaceDropdown.value = false
    showTaskDropdown.value = false
    currentPage.value = 1
    fetchTransactions()
  }

  const changePage = (page) => {
    if (page < 1 || page > totalPages.value) return
    currentPage.value = page
    fetchTransactions()
  }

  const handleClickOutside = (event) => {
    if (!event.target.closest('.project-filter')) showProjectDropdown.value = false
    if (!event.target.closest('.user-filter')) showUserDropdown.value = false
    if (!event.target.closest('.workspace-filter')) showWorkspaceDropdown.value = false
    if (!event.target.closest('.task-filter')) showTaskDropdown.value = false
  }

  onMounted(() => {
    fetchBillingUnitTypeOptions()
    fetchProjects()
    fetchUsers()
    fetchWorkspaces()
    fetchTransactions()
    document.addEventListener('click', handleClickOutside)
  })

  onUnmounted(() => {
    document.removeEventListener('click', handleClickOutside)
    ;[projectTimeout, userTimeout, workspaceTimeout, taskTimeout].forEach((t) => {
      if (t.current) clearTimeout(t.current)
    })
    if (projectSearchTimeout) clearTimeout(projectSearchTimeout)
    if (userSearchTimeout) clearTimeout(userSearchTimeout)
    if (workspaceSearchTimeout) clearTimeout(workspaceSearchTimeout)
    if (taskSearchTimeout) clearTimeout(taskSearchTimeout)
  })

  return {
    tenantId,
    filters,
    billingUnitTypeOptions,
    rechargeSourceOptions,
    transactions,
    isLoading,
    currentPage,
    totalPages,
    totalCount,
    projectSearch,
    userSearch,
    workspaceSearch,
    taskSearch,
    projectOptions,
    userOptions,
    workspaceOptions,
    taskOptions,
    showProjectDropdown,
    showUserDropdown,
    showWorkspaceDropdown,
    showTaskDropdown,
    selectedProjectName,
    selectedUserName,
    selectedWorkspaceName,
    selectedTaskName,
    formatDate,
    getTransactionTypeText,
    searchProjects,
    searchUsers,
    searchWorkspaces,
    searchTasks,
    openProjectDropdown,
    openUserDropdown,
    openWorkspaceDropdown,
    openTaskDropdown,
    selectProject,
    selectUser,
    selectWorkspace,
    selectTask,
    applyFilters,
    resetFilters,
    changePage
  }
}
