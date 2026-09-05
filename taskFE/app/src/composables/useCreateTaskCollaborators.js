import { computed, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { getStoredUserId } from '../utils/sessionUserIdUtils.js'
import { normalizeListPayload, parseJsonSafe, warnNetworkFailure, warnOptionalApiFailure } from '../utils/workPanelApiUtils.js'

/** @param {{ editingTask: () => object|null, tenantId: () => string|number|null, currentWorkspace: () => object|null, show: () => boolean }} opts */
export function useCreateTaskCollaborators({ editingTask, tenantId, currentWorkspace, show }) {
  const showAssigneePicker = ref(false)
  const showOwnerPicker = ref(false)
  const showOperatorPicker = ref(false)
  const ownerFilterQuery = ref('')
  const operatorFilterQuery = ref('')
  const assigneeFilterQuery = ref('')
  const localCollaborators = ref([])
  const isCollaboratorsLoading = ref(false)
  const collaboratorsError = ref(null)

  const COLLABORATOR_NAME_FALLBACK = '未设置公司昵称'
  const ownerInputPlaceholder = '输入关键字筛选公司成员昵称…'

  const getCollaboratorDisplayName = (person) => {
    if (!person) return COLLABORATOR_NAME_FALLBACK
    const memberName = String(person.member_name || '').trim()
    return memberName || COLLABORATOR_NAME_FALLBACK
  }

  const ownerDisplayLabel = computed(() => {
    if (!editingTask()?.owner) return ''
    const person = localCollaborators.value.find((item) => String(item.id) === String(editingTask().owner))
    return person ? getCollaboratorDisplayName(person) : COLLABORATOR_NAME_FALLBACK
  })

  const operatorDisplayLabel = computed(() => {
    if (!editingTask()?.operator) return ''
    const person = localCollaborators.value.find((item) => String(item.id) === String(editingTask().operator))
    return person ? getCollaboratorDisplayName(person) : COLLABORATOR_NAME_FALLBACK
  })

  const collaboratorMatchesFilter = (person, queryLower) => {
    if (!queryLower) return true
    const memberName = String(person?.member_name || '').trim().toLowerCase()
    if (!memberName) return false
    return memberName.includes(queryLower)
  }

  const filteredOwnersForPicker = computed(() => {
    const q = ownerFilterQuery.value.trim().toLowerCase()
    return localCollaborators.value.filter((p) => collaboratorMatchesFilter(p, q))
  })

  const filteredOperatorsForPicker = computed(() => {
    const q = operatorFilterQuery.value.trim().toLowerCase()
    return localCollaborators.value.filter((p) => collaboratorMatchesFilter(p, q))
  })

  const normalizeAssignees = () => {
    const task = editingTask()
    if (!task) return []
    if (!Array.isArray(task.assignees)) {
      task.assignees = []
    }
    return task.assignees
  }

  const selectedAssigneeTags = computed(() => {
    const currentAssignees = normalizeAssignees().map((id) => String(id))
    return currentAssignees.map((id) => {
      const person = localCollaborators.value.find((item) => String(item.id) === id)
      return {
        id,
        name: person ? getCollaboratorDisplayName(person) : COLLABORATOR_NAME_FALLBACK,
      }
    })
  })

  const availableCollaborators = computed(() => {
    const selectedSet = new Set(normalizeAssignees().map((id) => String(id)))
    return localCollaborators.value.filter((person) => !selectedSet.has(String(person.id)))
  })

  const filteredAvailableCollaborators = computed(() => {
    const q = assigneeFilterQuery.value.trim().toLowerCase()
    return availableCollaborators.value.filter((p) => collaboratorMatchesFilter(p, q))
  })

  /**
   * 将负责人/操作员字段对齐为工作空间成员主键：
   * - 已是合法成员 id → 保留
   * - 空值且为创建任务（无 id）→ 默认当前登录用户对应成员
   * - 预填为 userId → 映射为成员 id；无效则清空（创建时再回退当前用户）
   */
  const alignPersonFieldWithCollaborators = (field) => {
    const task = editingTask()
    if (!task) return
    const coll = localCollaborators.value
    if (!coll.length) return
    const memberIds = new Set(coll.map((c) => String(c.id)))
    const raw = String(task[field] || '').trim()
    if (raw && memberIds.has(raw)) return

    const uid = getStoredUserId()
    const currentMember = uid
      ? coll.find((c) => String(c.user) === String(uid))
      : null
    const isCreate = !task.id

    if (!raw) {
      if (isCreate && currentMember) {
        task[field] = String(currentMember.id)
      }
      return
    }

    const matchUser = coll.find((c) => String(c.user) === raw)
    if (matchUser) {
      task[field] = String(matchUser.id)
      return
    }

    if (isCreate && currentMember) {
      task[field] = String(currentMember.id)
      return
    }
    task[field] = ''
  }

  const alignOwnerAndOperatorWithCollaborators = () => {
    alignPersonFieldWithCollaborators('owner')
    alignPersonFieldWithCollaborators('operator')
  }

  const fetchCollaborators = async () => {
    const ws = currentWorkspace()
    if (!tenantId() || !ws?.id || ws.id === 'default') {
      localCollaborators.value = []
      return
    }

    isCollaboratorsLoading.value = true
    collaboratorsError.value = null

    try {
      const url = `/api/projects/workspace-access/workspace-collaborators/tenant_id/${tenantId()}/?workspace_id=${ws.id}`
      const response = await apiFetch(url, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (response.ok) {
        const data = await parseJsonSafe(response)
        if (data == null) {
          warnOptionalApiFailure('协作人员(JSON)', response)
          localCollaborators.value = []
          collaboratorsError.value = '协作人员列表响应异常'
        } else {
          localCollaborators.value = normalizeListPayload(data)
          alignOwnerAndOperatorWithCollaborators()
        }
      } else {
        warnOptionalApiFailure('协作人员', response)
        localCollaborators.value = []
        collaboratorsError.value = `获取协作人员失败（HTTP ${response.status}）`
      }
    } catch (e) {
      warnNetworkFailure('协作人员', e)
      localCollaborators.value = []
      collaboratorsError.value = '网络错误，请稍后重试'
    } finally {
      isCollaboratorsLoading.value = false
    }
  }

  const onOwnerInputFocus = () => {
    showAssigneePicker.value = false
    showOperatorPicker.value = false
    assigneeFilterQuery.value = ''
    operatorFilterQuery.value = ''
    showOwnerPicker.value = true
    ownerFilterQuery.value = ''
  }

  const onOwnerSearchInput = (event) => {
    showAssigneePicker.value = false
    showOperatorPicker.value = false
    assigneeFilterQuery.value = ''
    operatorFilterQuery.value = ''
    ownerFilterQuery.value = event.target.value
    showOwnerPicker.value = true
  }

  const selectOwnerMember = (person) => {
    const task = editingTask()
    if (!task) return
    task.owner = person ? String(person.id) : ''
    showOwnerPicker.value = false
    ownerFilterQuery.value = ''
  }

  const onOperatorInputFocus = () => {
    showAssigneePicker.value = false
    showOwnerPicker.value = false
    assigneeFilterQuery.value = ''
    ownerFilterQuery.value = ''
    showOperatorPicker.value = true
    operatorFilterQuery.value = ''
  }

  const onOperatorSearchInput = (event) => {
    showAssigneePicker.value = false
    showOwnerPicker.value = false
    assigneeFilterQuery.value = ''
    ownerFilterQuery.value = ''
    operatorFilterQuery.value = event.target.value
    showOperatorPicker.value = true
  }

  const selectOperatorMember = (person) => {
    const task = editingTask()
    if (!task) return
    task.operator = person ? String(person.id) : ''
    showOperatorPicker.value = false
    operatorFilterQuery.value = ''
  }

  const addAssignee = (assigneeId) => {
    const task = editingTask()
    if (!task) return
    const next = normalizeAssignees().map((id) => String(id))
    const targetId = String(assigneeId)
    if (next.includes(targetId)) return
    task.assignees = [...next, targetId]
  }

  const removeAssignee = (assigneeId) => {
    const task = editingTask()
    if (!task) return
    const targetId = String(assigneeId)
    task.assignees = normalizeAssignees()
      .map((id) => String(id))
      .filter((id) => id !== targetId)
  }

  const toggleAssigneePicker = () => {
    const willOpen = !showAssigneePicker.value
    if (willOpen) {
      showOwnerPicker.value = false
      showOperatorPicker.value = false
      assigneeFilterQuery.value = ''
    }
    showAssigneePicker.value = willOpen
  }

  const closePickers = () => {
    showAssigneePicker.value = false
    showOwnerPicker.value = false
    showOperatorPicker.value = false
    ownerFilterQuery.value = ''
    operatorFilterQuery.value = ''
    assigneeFilterQuery.value = ''
  }

  watch(
    () => [show(), tenantId(), currentWorkspace()?.id],
    () => {
      if (!show()) return
      fetchCollaborators()
    },
    { immediate: true },
  )

  watch(
    () => show(),
    (visible) => {
      if (!visible) closePickers()
    },
  )

  return {
    showAssigneePicker,
    showOwnerPicker,
    showOperatorPicker,
    ownerFilterQuery,
    operatorFilterQuery,
    assigneeFilterQuery,
    localCollaborators,
    isCollaboratorsLoading,
    collaboratorsError,
    ownerInputPlaceholder,
    ownerDisplayLabel,
    operatorDisplayLabel,
    filteredOwnersForPicker,
    filteredOperatorsForPicker,
    selectedAssigneeTags,
    filteredAvailableCollaborators,
    availableCollaborators,
    getCollaboratorDisplayName,
    onOwnerInputFocus,
    onOwnerSearchInput,
    selectOwnerMember,
    onOperatorInputFocus,
    onOperatorSearchInput,
    selectOperatorMember,
    addAssignee,
    removeAssignee,
    toggleAssigneePicker,
    closePickers,
    fetchCollaborators,
  }
}
