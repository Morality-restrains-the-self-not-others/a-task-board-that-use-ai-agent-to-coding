import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { usePermissions } from './usePermissions.js'
import { canSetSuperuserFlag, omitUnauthorizedSuperuser } from './adminUserSuperuserGate.js'

/** @alias:view-system-admin-users */
export function useSystemAdminUsers() {
  const route = useRoute()
  const router = useRouter()
  const { isPlatformRole, hasPlatformPerm, load: loadPerms } = usePermissions()

  // OPT-20260819-024: ?tab=referral-apps 深链；租户 Tab 同模式 ?tab=tenants
  const PANEL_TABS = ['referral-apps', 'tenants']
  const ALL_TABS = ['active', 'archived', ...PANEL_TABS]
  const initialTab = String(route.query.tab || '').trim()
  const activeTab = ref(ALL_TABS.includes(initialTab) ? initialTab : 'active')
  const isUserListTab = computed(() => activeTab.value === 'active' || activeTab.value === 'archived')
  const addUserModalVisible = ref(false)
  const editUserModalVisible = ref(false)
  const deleteUserModalVisible = ref(false)
  const rechargeDrawerVisible = ref(false)
  const rechargeDrawerUserId = ref('')
  const kycDrawerVisible = ref(false)
  const kycDrawerUserId = ref('')

  const openRechargeDrawer = (user) => {
    rechargeDrawerUserId.value = user?.id != null ? String(user.id) : ''
    rechargeDrawerVisible.value = true
  }

  const openKycDrawer = (user) => {
    kycDrawerUserId.value = user?.id != null ? String(user.id) : ''
    kycDrawerVisible.value = true
  }

  const loadingUsers = ref(false)
  const loadingUser = ref(false)
  const addingUser = ref(false)
  const editingUser = ref(false)
  const deletingUser = ref(false)

  const addUserForm = ref({
    username: '',
    email: '',
    phone: '',
    password: '',
    is_superuser: false,
    is_staff: false,
    is_tenant: false,
    is_tester: false
  })

  const editUserForm = ref({
    username: '',
    email: '',
    phone: '',
    password: '',
    is_superuser: false,
    is_staff: false,
    is_tenant: false,
    is_tester: false
  })

  const editUserId = ref(null)
  const editUserError = ref('')
  const editUserErrorTraceId = ref('')
  const editUserErrorField = ref('')
  const editSaveGuard = createClickGuard()
  const addUserError = ref('')
  const addUserErrorTraceId = ref('')
  const addUserErrorField = ref('')
  const addSaveGuard = createClickGuard()
  const deleteUserId = ref(null)
  const deleteUserName = ref('')
  const users = ref([])
  const totalUsers = ref(null)
  // OPT-20260824-091: 后端在 q/phone/email 检索时不再套用 is_archived=false，活跃 Tab
  // 能搜到归档占用者。行上已有「已归档」徽标，此标志供页面在搜索结果含归档用户时
  // 展示提示，避免运营误以为「活跃 Tab 混入了归档数据」是筛选坏了。
  const activeTabHasArchivedResults = computed(
    () => activeTab.value === 'active' && users.value.some((u) => u?.is_archived === true),
  )
  const loadError = ref('')
  const loadErrorTraceId = ref('')
  const currentOffset = ref(0)
  const pageLimit = 50
  const searchQuery = ref('')
  const archivingUser = ref(false)
  const columnFilters = ref(emptyColumnFilters())
  let columnFilterTimer = null

  function emptyColumnFilters() {
    return {
      id: '',
      email: '',
      tenant_company: '',
      phone: '',
      login_method: '',
      referrer: '',
      date_joined_from: '',
      date_joined_to: '',
      last_login_from: '',
      last_login_to: '',
      is_active: '',
      role: '',
      has_profit_sharing: '',
    }
  }

  function appendColumnFilterParams(url) {
    const f = columnFilters.value
    Object.keys(f).forEach((key) => {
      const val = String(f[key] ?? '').trim()
      if (val) url += `&${encodeURIComponent(key)}=${encodeURIComponent(val)}`
    })
    return url
  }

  const onColumnFilterUpdate = (key, value) => {
    columnFilters.value = { ...columnFilters.value, [key]: value }
    currentOffset.value = 0
    clearTimeout(columnFilterTimer)
    columnFilterTimer = setTimeout(() => {
      refreshUsers()
    }, 400)
  }

  const resetColumnFilters = () => {
    columnFilters.value = emptyColumnFilters()
    currentOffset.value = 0
    clearTimeout(columnFilterTimer)
    refreshUsers()
  }

  const formatDate = (dateString) => {
    if (!dateString) return '-'
    if (dateString instanceof Date) {
      return dateString.toLocaleString('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit'
      })
    }
    const dateStr = String(dateString)
    let date = new Date(dateStr)
    if (isNaN(date.getTime())) {
      let processedDateStr = dateStr
        .replace(/(\+\d{2}:\d{2}|Z)$/, '')
        .replace('T', ' ')
        .replace(/\.\d+/, '')
      date = new Date(processedDateStr)
      if (isNaN(date.getTime())) {
        const dateRegex = /(\d{4})[-/](\d{2})[-/](\d{2})\s*(\d{2}):(\d{2}):(\d{2})/
        const match = processedDateStr.match(dateRegex)
        if (match) {
          const [, year, month, day, hour, minute, second] = match
          date = new Date(year, month - 1, day, hour, minute, second)
        }
      }
    }
    if (isNaN(date.getTime())) return dateStr
    return date.toLocaleString('zh-CN', {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit'
    })
  }

  const refreshUsers = async () => {
    loadingUsers.value = true
    loadError.value = ''
    loadErrorTraceId.value = ''
    try {
      let url = `/api/system-admin/users/?limit=${pageLimit}&offset=${currentOffset.value}`
      if (searchQuery.value) {
        url += `&q=${encodeURIComponent(searchQuery.value)}`
      }
      url = appendColumnFilterParams(url)
      if (activeTab.value === 'archived') {
        url += '&is_archived=true'
      } else {
        url += '&is_archived=false'
      }
      const response = await apiFetch(url, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })
      if (response.ok) {
        const data = await response.json()
        users.value = data.users || []
        totalUsers.value = typeof data.total !== 'undefined' ? Number(data.total) : null
      } else {
        let message = `请求失败 (${response.status})`
        try {
          const errorData = await response.json()
          if (errorData.detail) message = errorData.detail
          if (errorData.trace_id || response.traceId) {
            loadErrorTraceId.value = errorData.trace_id || response.traceId || ''
          }
        } catch { /* ignore */ }
        throw new Error(message)
      }
    } catch (error) {
      console.error('刷新用户列表失败:', error)
      loadError.value = error?.message || '获取用户列表失败，请检查网络后重试'
      loadErrorTraceId.value = error?.traceId || loadErrorTraceId.value || ''
    } finally {
      loadingUsers.value = false
    }
  }

  const switchTab = (tab) => {
    activeTab.value = tab
    // OPT-20260819-024: tab 同步到 query，推荐资格页跳转可直接落到申请审批
    const q = { ...route.query }
    if (PANEL_TABS.includes(tab)) q.tab = tab
    else delete q.tab
    router.replace({ query: q })
    if (PANEL_TABS.includes(tab)) return
    currentOffset.value = 0
    refreshUsers()
  }

  const handleSearch = () => {
    currentOffset.value = 0
    refreshUsers()
  }

  const clearSearch = () => {
    searchQuery.value = ''
    currentOffset.value = 0
    refreshUsers()
  }

  const handleArchiveUser = async (user) => {
    archivingUser.value = true
    try {
      const response = await apiFetch(`/api/system-admin/users/${user.id}/archive/`, {
        method: 'POST',
        headers: { 'X-Requested-With': 'XMLHttpRequest' },
        credentials: 'include'
      })
      if (response.ok) {
        await refreshUsers()
      } else {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.detail || '归档用户失败')
      }
    } catch (error) {
      console.error('归档用户失败:', error)
      showRequestError('归档用户失败: ' + error.message, error)
    } finally {
      archivingUser.value = false
    }
  }

  const handleUnarchiveUser = async (user) => {
    archivingUser.value = true
    try {
      const response = await apiFetch(`/api/system-admin/users/${user.id}/unarchive/`, {
        method: 'POST',
        headers: { 'X-Requested-With': 'XMLHttpRequest' },
        credentials: 'include'
      })
      if (response.ok) {
        await refreshUsers()
      } else {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.detail || '取消归档失败')
      }
    } catch (error) {
      console.error('取消归档失败:', error)
      showRequestError('取消归档失败: ' + error.message, error)
    } finally {
      archivingUser.value = false
    }
  }

  const goToPrevPage = () => {
    if (currentOffset.value > 0) {
      currentOffset.value = Math.max(0, currentOffset.value - pageLimit)
      refreshUsers()
    }
  }

  const goToNextPage = () => {
    if (totalUsers.value !== null && currentOffset.value + pageLimit < totalUsers.value) {
      currentOffset.value = currentOffset.value + pageLimit
      refreshUsers()
    }
  }

  const openAddUserModal = () => {
    addUserForm.value = {
      username: '', email: '', phone: '', password: '',
      is_superuser: false, is_staff: false, is_tenant: false, is_tester: false
    }
    addUserError.value = ''
    addUserErrorTraceId.value = ''
    addUserErrorField.value = ''
    addUserModalVisible.value = true
  }

  const handleAddUser = async () => {
    const run = await addSaveGuard.run(async ({ headers, idempotencyKey }) => {
      addingUser.value = true
      addUserError.value = ''
      addUserErrorTraceId.value = ''
      addUserErrorField.value = ''
      try {
        const payload = omitUnauthorizedSuperuser(
          addUserForm.value,
          canSetSuperuserFlag(isPlatformRole, hasPlatformPerm),
        )
        const response = await apiFetch('/api/system-admin/users/create/', {
          method: 'POST',
          headers: mergeIdempotencyHeaders({
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
            ...headers,
          }, idempotencyKey),
          credentials: 'include',
          body: JSON.stringify(payload)
        })
        const errorData = await response.json().catch(() => ({}))
        const headerTrace = response.headers?.get?.('X-Trace-Id') || response.headers?.get?.('x-trace-id') || ''
        if (response.ok) {
          addUserModalVisible.value = false
          await refreshUsers()
          return
        }
        const detail = errorData.detail || errorData.error || '添加用户失败'
        addUserError.value = detail
        addUserErrorTraceId.value = errorData.trace_id || headerTrace || ''
        if (errorData.code === 'phone_taken' || errorData.code === 'phone_bind_limit') {
          addUserErrorField.value = 'phone'
        } else if (errorData.code === 'email_taken') {
          addUserErrorField.value = 'email'
        }
      } catch (error) {
        console.error('添加用户失败:', error)
        addUserError.value = error?.message || '添加用户失败'
      } finally {
        addingUser.value = false
      }
    })
    return run
  }

  const openEditUserModal = (user) => {
    loadingUser.value = true
    editUserModalVisible.value = true
    editUserError.value = ''
    editUserErrorTraceId.value = ''
    editUserErrorField.value = ''
    editUserForm.value = {
      username: user.username,
      email: user.email,
      phone: user.phone || '',
      password: '',
      is_superuser: user.is_superuser,
      is_staff: user.is_staff,
      is_tenant: user.is_tenant,
      is_tester: !!user.is_tester
    }
    editUserId.value = user.id == null ? '' : String(user.id)
    loadingUser.value = false
  }

  const onTesterFlagChange = (form) => {
    if (form?.is_tester) form.is_tenant = true
  }

  const handleEditUser = async () => {
    const run = await editSaveGuard.run(async ({ headers, idempotencyKey }) => {
      editingUser.value = true
      editUserError.value = ''
      editUserErrorTraceId.value = ''
      editUserErrorField.value = ''
      try {
        const formData = omitUnauthorizedSuperuser(
          editUserForm.value,
          canSetSuperuserFlag(isPlatformRole, hasPlatformPerm),
        )
        const response = await apiFetch(`/api/system-admin/users/${editUserId.value}/`, {
          method: 'PUT',
          headers: mergeIdempotencyHeaders({
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
            ...headers,
          }, idempotencyKey),
          credentials: 'include',
          body: JSON.stringify(formData)
        })
        const errorData = await response.json().catch(() => ({}))
        const headerTrace = response.headers?.get?.('X-Trace-Id') || response.headers?.get?.('x-trace-id') || ''
        if (response.ok) {
          editUserModalVisible.value = false
          await refreshUsers()
          return
        }
        const detail = errorData.detail || errorData.error || '更新用户失败'
        editUserError.value = detail
        editUserErrorTraceId.value = errorData.trace_id || headerTrace || ''
        if (errorData.code === 'phone_taken' || errorData.code === 'phone_bind_limit') {
          editUserErrorField.value = 'phone'
        } else if (errorData.code === 'email_taken') {
          editUserErrorField.value = 'email'
        }
      } catch (error) {
        console.error('更新用户失败:', error)
        editUserError.value = error?.message || '更新用户失败'
      } finally {
        editingUser.value = false
      }
    })
    return run
  }

  const openDeleteUserModal = (user) => {
    deleteUserId.value = user.id
    deleteUserName.value = user.username
    deleteUserModalVisible.value = true
  }

  const handleDeleteUser = async () => {
    deletingUser.value = true
    try {
      const response = await apiFetch(`/api/system-admin/users/${deleteUserId.value}/delete/`, {
        method: 'DELETE',
        headers: { 'X-Requested-With': 'XMLHttpRequest' },
        credentials: 'include'
      })
      if (response.ok) {
        deleteUserModalVisible.value = false
        await refreshUsers()
      } else {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.detail || '删除用户失败')
      }
    } catch (error) {
      console.error('删除用户失败:', error)
      showRequestError('删除用户失败: ' + error.message, error)
    } finally {
      deletingUser.value = false
    }
  }

  onMounted(async () => {
    await loadPerms(apiFetch)
    if (isUserListTab.value) {
      await refreshUsers()
    }
  })

  onUnmounted(() => {
    clearTimeout(columnFilterTimer)
  })

  return {
    addUserModalVisible, editUserModalVisible, deleteUserModalVisible,
    rechargeDrawerVisible, rechargeDrawerUserId, openRechargeDrawer,
    kycDrawerVisible, kycDrawerUserId, openKycDrawer,
    loadingUsers, loadingUser, addingUser, editingUser, deletingUser, archivingUser,
    addUserForm, editUserForm, editUserId, editUserError, editUserErrorTraceId, editUserErrorField, addUserError, addUserErrorTraceId, addUserErrorField, deleteUserId, deleteUserName,
    users, totalUsers, loadError, loadErrorTraceId, currentOffset, pageLimit,
    searchQuery, activeTab, isUserListTab, activeTabHasArchivedResults, columnFilters,
    formatDate, refreshUsers, goToPrevPage, goToNextPage,
    switchTab, handleSearch, clearSearch,
    onColumnFilterUpdate, resetColumnFilters,
    handleArchiveUser, handleUnarchiveUser,
    openAddUserModal, handleAddUser, openEditUserModal, handleEditUser,
    openDeleteUserModal, handleDeleteUser, onTesterFlagChange
  }
}
