import { ref, computed } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'

export function useProjectsBatchDelete({ tenantId, projects, filteredProjects, onDeleted }) {
  const selectionMode = ref(false)
  /** @type {import('vue').Ref<Record<string, true>>} */
  const selectedProjectIds = ref({})
  const showConfirmModal = ref(false)
  const deleting = ref(false)
  const error = ref('')
  const deleteResult = ref(null)

  const selectedCount = computed(() => Object.keys(selectedProjectIds.value).length)

  const selectedProjects = computed(() =>
    projects.value.filter((p) => selectedProjectIds.value[String(p.id)]),
  )

  const allFilteredChecked = computed(() => {
    if (!filteredProjects.value.length) return false
    return filteredProjects.value.every((p) => selectedProjectIds.value[String(p.id)])
  })

  const enterSelectionMode = () => {
    selectionMode.value = true
    deleteResult.value = null
    error.value = ''
  }

  const exitSelectionMode = () => {
    if (deleting.value) return
    selectionMode.value = false
    selectedProjectIds.value = {}
    showConfirmModal.value = false
    error.value = ''
  }

  const toggleSelectionMode = () => {
    if (selectionMode.value) {
      exitSelectionMode()
    } else {
      enterSelectionMode()
    }
  }

  const toggleProject = (projectId, checked) => {
    const key = String(projectId)
    const next = { ...selectedProjectIds.value }
    if (checked) {
      next[key] = true
    } else {
      delete next[key]
    }
    selectedProjectIds.value = next
  }

  const toggleSelectAllFiltered = (checked) => {
    if (!checked) {
      const next = { ...selectedProjectIds.value }
      for (const p of filteredProjects.value) {
        delete next[String(p.id)]
      }
      selectedProjectIds.value = next
      return
    }
    selectedProjectIds.value = {
      ...selectedProjectIds.value,
      ...Object.fromEntries(filteredProjects.value.map((p) => [String(p.id), true])),
    }
  }

  const openBatchDeleteConfirm = () => {
    if (!selectedCount.value) return
    error.value = ''
    showConfirmModal.value = true
  }

  const closeConfirmModal = () => {
    if (deleting.value) return
    showConfirmModal.value = false
  }

  const executeBatchDelete = async () => {
    const tid = tenantId.value
    if (!tid) {
      error.value = '缺少租户 ID'
      return
    }
    const ids = Object.keys(selectedProjectIds.value)
    if (!ids.length) {
      error.value = '请至少选择一个项目'
      return
    }

    deleting.value = true
    error.value = ''
    let shouldExitSelection = false
    try {
      // taskProjectService handleProjectsRoute action 段分发：位置段在前、kv 键值对在后
      const response = await apiFetch(`/api/projects/batch-delete/tenant_id/${tid}/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ project_ids: ids }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        error.value = data.error || data.detail || '批量删除失败'
        return
      }

      deleteResult.value = data
      const deletedIds = new Set((data.deleted || []).map((d) => String(d.id)))
      const next = { ...selectedProjectIds.value }
      for (const id of deletedIds) {
        delete next[id]
      }
      selectedProjectIds.value = next

      if (typeof onDeleted === 'function') {
        await onDeleted(data)
      }

      if (!data.errors?.length) {
        showConfirmModal.value = false
        shouldExitSelection = true
      }
    } catch {
      error.value = '网络错误，请稍后重试'
    } finally {
      deleting.value = false
      if (shouldExitSelection) {
        exitSelectionMode()
      }
    }
  }

  return {
    selectionMode,
    selectedProjectIds,
    showConfirmModal,
    deleting,
    error,
    deleteResult,
    selectedCount,
    selectedProjects,
    allFilteredChecked,
    enterSelectionMode,
    toggleSelectionMode,
    exitSelectionMode,
    toggleProject,
    toggleSelectAllFiltered,
    openBatchDeleteConfirm,
    closeConfirmModal,
    executeBatchDelete,
  }
}
