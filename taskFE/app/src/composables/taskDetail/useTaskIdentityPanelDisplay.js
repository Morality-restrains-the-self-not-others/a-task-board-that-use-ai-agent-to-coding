import { computed, ref } from 'vue'
import {
  formatCollaboratorsDisplay,
  formatCollaboratorsDisplayFull,
  formatSingleMemberDisplay,
} from '../../utils/taskCardPeopleDisplay.js'
import { formatTaskDisplayNo, formatTaskIdTitleLabel } from '../../utils/taskIdDisplay.js'

/**
 * Identity 面板人员/上层交付物/派生自展示文案与任务 ID 复制。
 */
function workspaceSeqOf(todos, id, fetchedSeq) {
  const fetched = Number(fetchedSeq)
  if (Number.isInteger(fetched) && fetched > 0) return fetched
  if (id == null || id === '' || !Array.isArray(todos)) return 0
  const found = todos.find((t) => String(t?.id ?? '') === String(id))
  return found?.workspace_seq
}

export function useTaskIdentityPanelDisplay(props, {
  resolvedParentTaskTitle,
  resolvedParentTaskSeq,
  parentDeliverableId,
  workspaceTodos,
}) {
  const taskIdCopied = ref(false)
  let taskIdCopyTimer = null

  const copyTaskId = async () => {
    const id = props.task?.id != null ? String(props.task.id) : ''
    if (!id) return
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(id)
      } else if (typeof document !== 'undefined') {
        const ta = document.createElement('textarea')
        ta.value = id
        ta.setAttribute('readonly', '')
        ta.style.position = 'fixed'
        ta.style.left = '-9999px'
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
      }
      taskIdCopied.value = true
      if (taskIdCopyTimer) clearTimeout(taskIdCopyTimer)
      taskIdCopyTimer = setTimeout(() => {
        taskIdCopied.value = false
      }, 1500)
    } catch (err) {
      console.warn('[TaskDetailTaskIdentityPanel] copy task id failed', err)
    }
  }

  const taskDisplayNo = computed(() =>
    formatTaskDisplayNo(
      workspaceSeqOf(todosList(), props.task?.id, props.task?.workspace_seq),
    ),
  )

  const assigneesDisplayText = computed(() =>
    formatCollaboratorsDisplay(props.task?.assignees, props.collaboratorNameById),
  )
  const assigneesDisplayTitle = computed(() =>
    formatCollaboratorsDisplayFull(props.task?.assignees, props.collaboratorNameById),
  )
  const ownerDisplayText = computed(() =>
    formatSingleMemberDisplay(props.task?.owner, props.collaboratorNameById),
  )
  const operatorDisplayText = computed(() =>
    formatSingleMemberDisplay(props.task?.operator, props.collaboratorNameById),
  )
  const todosList = () => {
    const raw = workspaceTodos && typeof workspaceTodos === 'object' && 'value' in workspaceTodos
      ? workspaceTodos.value
      : workspaceTodos
    if (Array.isArray(raw)) return raw
    return Array.isArray(props.workspaceTodos) ? props.workspaceTodos : []
  }
  const parentDeliverableDisplayText = computed(() =>
    formatTaskIdTitleLabel(
      parentDeliverableId.value,
      resolvedParentTaskTitle.value,
      workspaceSeqOf(
        todosList(),
        parentDeliverableId.value,
        resolvedParentTaskSeq && typeof resolvedParentTaskSeq === 'object' && 'value' in resolvedParentTaskSeq
          ? resolvedParentTaskSeq.value
          : resolvedParentTaskSeq,
      ),
    ),
  )
  const resolvedIsForkSourceTitleLoading = computed(() => Boolean(props.isForkSourceTitleLoading))
  const forkSourceDisplayText = computed(() =>
    formatTaskIdTitleLabel(
      props.task?.fork_from,
      props.forkSourceTitle,
      workspaceSeqOf(todosList(), props.task?.fork_from, props.forkSourceSeq),
    ),
  )

  return {
    taskIdCopied,
    copyTaskId,
    taskDisplayNo,
    assigneesDisplayText,
    assigneesDisplayTitle,
    ownerDisplayText,
    operatorDisplayText,
    parentDeliverableDisplayText,
    resolvedIsForkSourceTitleLoading,
    forkSourceDisplayText,
  }
}
