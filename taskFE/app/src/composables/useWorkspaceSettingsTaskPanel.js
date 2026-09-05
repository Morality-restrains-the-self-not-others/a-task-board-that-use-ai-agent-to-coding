/**
 * settings/task-panel 工作空间管理页：工作空间列表 / 任务面板 CRUD / 添加编辑面板模态。
 * 设置类模态（存档、交付物、进度、访问、机器策略、选项、功能参数）见
 * useWorkspaceSettingsModals.js。两个 composable 由 WorkspaceSettingsTaskPanel.vue 聚合。
 */
import { ref } from 'vue'
import { apiFetch, extractErrorMessage } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
const DEFAULT_PANEL_COLOR = '#3b82f6'
export function useWorkspaceSettingsTaskPanel({ tenantId }) {
  // tenantId: () => string — 当前租户 id 惰性读取
  const getCurrentTenantId = tenantId || (() => '')
  // 工作空间相关
  const workspaceOptions = ref([])
  const selectedWorkspaceId = ref('')
  const loadingWorkspaces = ref(true)
  const showCreateWorkspaceModal = ref(false)
  const newWorkspace = ref({ name: '', description: '', is_default: false })
  const editingWorkspace = ref(null)
  const savingWorkspace = ref(false)
  // 任务面板相关
  const taskPanels = ref([])
  const loading = ref(true)
  const showAddModal = ref(false)
  const showEditModal = ref(false)
  const draggedPanel = ref(null)
  // 新面板表单
  const newPanel = ref({ name: '', color: DEFAULT_PANEL_COLOR })
  // 编辑面板表单
  const editingPanel = ref({ id: '', name: '', color: '' })
  // 可用颜色选项
  const availableColors = ref([
    '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899',
    '#6366f1', '#14b8a6', '#f97316', '#dc2626', '#7c3aed', '#db2777',
    '#2563eb', '#059669', '#d97706', '#b91c1c', '#6d28d9', '#be185d',
  ])
  // 获取当前工作空间ID
  const getCurrentWorkspaceId = () => {
    return new URL(window.location.href).searchParams.get('workspace_id') || ''
  }
  // 加载工作空间列表
  const loadWorkspaces = () => {
    const tenantId = getCurrentTenantId()
    if (!tenantId) {
      console.error('缺少租户ID，无法加载工作空间')
      loadingWorkspaces.value = false
      return
    }
    loadingWorkspaces.value = true

    const currentUrl = new URL(window.location.href)
    const workspaceIdParam = currentUrl.searchParams.get('workspace_id')

    let apiUrl = `/api/projects/workspaces/tenant_id/${tenantId}`
    if (workspaceIdParam) {
      apiUrl += `?workspace_id=${workspaceIdParam}`
    }

    apiFetch(apiUrl, {
      headers: {
        'Accept': 'application/json',
      },
    })
      .then(response => {
        if (!response.ok) {
          const err = new Error('Network response was not ok')
          err.traceId = response.traceId || ''
          throw err
        }
        return response.json()
      })
      .then(data => {
        workspaceOptions.value = []
        let currentActiveWorkspace = null

        if (data.length > 0) {
          data.forEach(workspace => {
            workspaceOptions.value.push({
              value: workspace.id,
              text: workspace.name,
              description: workspace.description || '',
              is_current: workspace.is_current,
              is_default: workspace.is_default,
              task_archive_tier: workspace.task_archive_tier || '7d',
            })

            if (workspace.is_current) {
              selectedWorkspaceId.value = workspace.id
              currentActiveWorkspace = { id: workspace.id, name: workspace.name }
            }
          })

          if (!currentActiveWorkspace && workspaceOptions.value.length > 0) {
            selectedWorkspaceId.value = workspaceOptions.value[0].value
          }
        }

        loadingWorkspaces.value = false

        // 如果有选中的工作空间，加载其任务面板
        if (selectedWorkspaceId.value) {
          loadTaskPanels()
        }
      })
      .catch(error => {
        console.error('Error loading workspaces:', error)
        workspaceOptions.value = []
        loadingWorkspaces.value = false
      })
  }

  const saveWorkspaceGuard = createClickGuard()

  // 保存工作空间
  const saveWorkspace = async () => {
    if (!newWorkspace.value.name) {
      return
    }
    // OPT-20260819-038: 创建/编辑工作空间是资源写操作，防连点双发 POST/PUT
    await saveWorkspaceGuard.run(async ({ idempotencyKey }) => {
      savingWorkspace.value = true
      try {
        const tenantId = getCurrentTenantId()
        const url = editingWorkspace.value ? `/api/projects/workspaces/tenant_id/${tenantId}/${editingWorkspace.value}/` : `/api/projects/workspaces/tenant_id/${tenantId}`
        const method = editingWorkspace.value ? 'PUT' : 'POST'

        const response = await apiFetch(url, {
          method,
          headers: mergeIdempotencyHeaders(
            {
              'Content-Type': 'application/json',
            },
            idempotencyKey,
          ),
          body: JSON.stringify(newWorkspace.value),
        })

        if (!response.ok) {
          const err = new Error('Network response was not ok')
          err.traceId = response.traceId || ''
          throw err
        }

        await response.json()

        // 重新加载工作空间列表
        await loadWorkspaces()

        // 关闭模态框
        showCreateWorkspaceModal.value = false
        resetNewWorkspace()
        editingWorkspace.value = null
      } catch (error) {
        console.error('Error saving workspace:', error)
        showRequestError('保存工作空间失败，请重试', error)
      } finally {
        savingWorkspace.value = false
      }
    })
  }

  // 编辑工作空间
  const editWorkspace = (workspace) => {
    editingWorkspace.value = workspace.value
    newWorkspace.value = {
      name: workspace.text,
      description: workspace.description || '',
      is_default: Boolean(workspace.is_default),
    }
    showCreateWorkspaceModal.value = true
  }

  const deleteWorkspaceGuard = createClickGuard()

  // 删除工作空间
  const deleteWorkspace = async (workspace) => {
    if (confirm('确定要删除这个工作空间吗？')) {
      // OPT-20260819-038: 删除是破坏性写操作，防连点双发 DELETE
      await deleteWorkspaceGuard.run(async ({ idempotencyKey }) => {
        try {
          const tenantId = getCurrentTenantId()
          const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}/${workspace.value}/`, {
            method: 'DELETE',
            headers: mergeIdempotencyHeaders({}, idempotencyKey),
          })

          if (!response.ok) {
            const err = new Error('Network response was not ok')
            err.traceId = response.traceId || ''
            throw err
          }

          // 重新加载工作空间列表
          await loadWorkspaces()
        } catch (error) {
          console.error('Error deleting workspace:', error)
          showRequestError('删除工作空间失败，请重试', error)
        }
      })
    }
  }

  // 加载任务面板
  const loadTaskPanels = async () => {
    loading.value = true
    try {
      const workspaceId = selectedWorkspaceId.value
      const tenantId = getCurrentTenantId()
      if (!workspaceId || !tenantId) {
        console.error('缺少工作空间ID或租户ID')
        return
      }

      const response = await apiFetch(`/api/projects/manage-task-panels/tenant_id/${tenantId}?workspace_id=${workspaceId}`, {
        headers: {},
      })

      const result = await response.json()

      if (result.status === 'success') {
        taskPanels.value = result.task_panels || []
      } else {
        console.error('加载任务面板失败:', result.message)
      }
    } catch (error) {
      console.error('加载任务面板失败:', error)
    } finally {
      loading.value = false
    }
  }

  // 创建新面板
  const createPanel = async () => {
    if (!newPanel.value.name || !newPanel.value.color) {
      return
    }

    try {
      const workspaceId = selectedWorkspaceId.value
      const tenantId = getCurrentTenantId()
      if (!workspaceId || !tenantId) {
        console.error('缺少工作空间ID或租户ID')
        return
      }

      const form = new URLSearchParams()
      form.append('action', 'create')
      form.append('workspace_id', workspaceId)
      form.append('name', newPanel.value.name)
      form.append('color', newPanel.value.color)

      const response = await apiFetch(`/api/projects/manage-task-panels/tenant_id/${tenantId}`, {
        method: 'POST',
        headers: {},
        body: form,
      })

      const result = await response.json()

      if (result.status === 'success') {
        taskPanels.value = result.task_panels || []
        showAddModal.value = false
        resetNewPanel()
      } else {
        const errorMsg = extractErrorMessage(result, response, '未知错误')
        showRequestError('创建面板失败: ' + errorMsg, response)
      }
    } catch (error) {
      console.error('创建面板失败:', error)
      showRequestError('创建面板失败，请重试', error)
    }
  }

  // 编辑面板
  const editPanel = (panel) => {
    editingPanel.value = {
      id: panel.id,
      name: panel.name,
      color: panel.color,
    }
    showEditModal.value = true
  }

  // 更新面板
  const updatePanel = async () => {
    if (!editingPanel.value.id || !editingPanel.value.name || !editingPanel.value.color) {
      return
    }

    try {
      const tenantId = getCurrentTenantId()
      if (!tenantId) {
        console.error('缺少租户ID')
        return
      }

      const form = new URLSearchParams()
      form.append('action', 'update')
      form.append('panel_id', editingPanel.value.id)
      form.append('name', editingPanel.value.name)
      form.append('color', editingPanel.value.color)

      const response = await apiFetch(`/api/projects/manage-task-panels/tenant_id/${tenantId}`, {
        method: 'POST',
        headers: {},
        body: form,
      })

      const result = await response.json()

      if (result.status === 'success') {
        taskPanels.value = result.task_panels || []
        showEditModal.value = false
      } else {
        const errorMsg = extractErrorMessage(result, response, '未知错误')
        showRequestError('更新面板失败: ' + errorMsg, response)
      }
    } catch (error) {
      console.error('更新面板失败:', error)
      showRequestError('更新面板失败，请重试', error)
    }
  }

  // 删除面板
  const deletePanel = async (panel) => {
    if (confirm('确定要删除这个面板吗？如果该面板下有任务，将无法删除。')) {
      try {
        const tenantId = getCurrentTenantId()
        if (!tenantId) {
          console.error('缺少租户ID')
          return
        }

        const form = new URLSearchParams()
        form.append('action', 'delete')
        form.append('panel_id', panel.id)

        const response = await apiFetch(`/api/projects/manage-task-panels/tenant_id/${tenantId}`, {
          method: 'POST',
          headers: {},
          body: form,
        })

        const result = await response.json()

        if (result.status === 'success') {
          taskPanels.value = result.task_panels || []
        } else {
          const errorMsg = extractErrorMessage(result, response, '未知错误')
          showRequestError('删除面板失败: ' + errorMsg, response)
        }
      } catch (error) {
        console.error('删除面板失败:', error)
        showRequestError('删除面板失败，请重试', error)
      }
    }
  }

  // 拖拽排序功能
  const dragStart = (event, panel) => {
    draggedPanel.value = panel
    event.target.classList.add('opacity-50')
  }

  const drop = (event, targetPanel) => {
    event.target.classList.remove('opacity-50')

    if (draggedPanel.value && draggedPanel.value.id !== targetPanel.id) {
      // 重新排序面板
      const newPanels = [...taskPanels.value]
      const draggedIndex = newPanels.findIndex(p => p.id === draggedPanel.value.id)
      const targetIndex = newPanels.findIndex(p => p.id === targetPanel.id)

      // 移除拖动的面板
      const [dragged] = newPanels.splice(draggedIndex, 1)
      // 插入到目标位置
      newPanels.splice(targetIndex, 0, dragged)

      // 更新排序
      updatePanelOrder(newPanels)
    }

    draggedPanel.value = null
  }

  // 更新面板排序
  const updatePanelOrder = async (orderedPanels) => {
    try {
      const tenantId = getCurrentTenantId()
      if (!tenantId) {
        console.error('缺少租户ID')
        return
      }

      // 批量更新面板排序
      for (let i = 0; i < orderedPanels.length; i++) {
        const panel = orderedPanels[i]
        const form = new URLSearchParams()
        form.append('action', 'update')
        form.append('panel_id', panel.id)
        form.append('order', i)

        await apiFetch(`/api/projects/manage-task-panels/tenant_id/${tenantId}`, {
          method: 'POST',
          headers: {},
          body: form,
        })
      }

      // 重新加载面板列表
      await loadTaskPanels()
    } catch (error) {
      console.error('更新面板排序失败:', error)
      showRequestError('更新面板排序失败，请重试', error)
    }
  }

  // 重置新面板表单
  const resetNewPanel = () => {
    newPanel.value = {
      name: '',
      color: DEFAULT_PANEL_COLOR,
    }
  }

  // 面板添加/编辑模态：模态内部维护 draft，提交时同步回表单再走既有 CRUD
  const submitAddPanel = (draft) => {
    newPanel.value = { name: draft?.name || '', color: draft?.color || '' }
    return createPanel()
  }

  const cancelAddPanel = () => {
    showAddModal.value = false
    resetNewPanel()
  }

  const submitEditPanel = (draft) => {
    editingPanel.value = {
      id: String(draft?.id || ''),
      name: draft?.name || '',
      color: draft?.color || '',
    }
    return updatePanel()
  }

  const cancelEditPanel = () => {
    showEditModal.value = false
  }

  // 重置新工作空间表单
  const resetNewWorkspace = () => {
    newWorkspace.value = {
      name: '',
      description: '',
      is_default: false,
    }
    editingWorkspace.value = null
  }

  return {
    // 工作空间
    workspaceOptions,
    selectedWorkspaceId,
    loadingWorkspaces,
    showCreateWorkspaceModal,
    newWorkspace,
    editingWorkspace,
    savingWorkspace,
    loadWorkspaces,
    saveWorkspace,
    editWorkspace,
    deleteWorkspace,
    resetNewWorkspace,
    // 任务面板
    taskPanels,
    loading,
    showAddModal,
    showEditModal,
    draggedPanel,
    newPanel,
    editingPanel,
    availableColors,
    loadTaskPanels,
    createPanel,
    editPanel,
    updatePanel,
    deletePanel,
    dragStart,
    drop,
    updatePanelOrder,
    resetNewPanel,
    submitAddPanel,
    cancelAddPanel,
    submitEditPanel,
    cancelEditPanel,
    getCurrentTenantId,
    getCurrentWorkspaceId,
  }
}
