/**
 * Edit-mode base-branch select / custom-input / datalist helpers for linked projects panel.
 */
import { ref } from 'vue'
import { createDatalistDismissController } from '../../utils/datalistDismiss.js'

/**
 * @param {object} deps
 * @param {{ repoCloneFieldId: Function, getRepoBranches: Function, getRepoBranchValue: Function }} deps.props
 * @param {(event: string, ...args: any[]) => void} deps.emit
 */
export function useLinkedProjectsEditBranches({ props, emit }) {
  const editBranchCustomInputByKey = ref({})

  const normalizeRepoUrlKey = (repoUrl) => String(repoUrl || '').trim()

  const editBranchFieldKey = (projectId, repoUrl) =>
    `${String(projectId || '').trim()}::${normalizeRepoUrlKey(repoUrl)}`

  const editBaseBranchDatalistId = (project, repoUrl) =>
    `task-edit-base-branch-options-${props.repoCloneFieldId(repoUrl)}`

  const isEditBranchCustomInput = (projectId, repoUrl) =>
    editBranchCustomInputByKey.value[editBranchFieldKey(projectId, repoUrl)] === true

  const shouldUseEditBaseBranchSelect = (project, repoUrl) => {
    const branches = props.getRepoBranches(project.project_id, repoUrl)
    if (!Array.isArray(branches) || branches.length === 0) return false
    if (isEditBranchCustomInput(project.project_id, repoUrl)) return false
    const current = String(props.getRepoBranchValue(project, repoUrl) || '').trim()
    return !current || branches.includes(current)
  }

  const getEditBaseBranchSelectValue = (project, repoUrl) => {
    const current = String(props.getRepoBranchValue(project, repoUrl) || '').trim()
    const branches = props.getRepoBranches(project.project_id, repoUrl)
    return branches.includes(current) ? current : ''
  }

  const setEditBranchCustomInput = (projectId, repoUrl, enabled) => {
    const key = editBranchFieldKey(projectId, repoUrl)
    editBranchCustomInputByKey.value = {
      ...editBranchCustomInputByKey.value,
      [key]: Boolean(enabled),
    }
  }

  const exitEditBranchCustomInput = (projectId, repoUrl) => {
    setEditBranchCustomInput(projectId, repoUrl, false)
  }

  const onEditBaseBranchSelect = (project, repoUrl, event) => {
    const value = String(event?.target?.value ?? '')
    if (value === '__custom__') {
      setEditBranchCustomInput(project.project_id, repoUrl, true)
      return
    }
    setEditBranchCustomInput(project.project_id, repoUrl, false)
    emit('set-repo-branch', project, repoUrl, value)
  }

  const {
    listAttr: datalistListAttr,
    dismissIfPicked: dismissDatalistIfPicked,
    restoreOnFocus: restoreDatalistOnFocus,
  } = createDatalistDismissController()

  const onEditBaseBranchInput = (project, repoUrl, event) => {
    const value = String(event?.target?.value ?? '')
    emit('set-repo-branch', project, repoUrl, value)
  }

  const onEditBaseBranchChange = (project, repoUrl, event) => {
    const inputEl = event?.target
    const value = String(inputEl?.value ?? '')
    emit('set-repo-branch', project, repoUrl, value)
    const branches = props.getRepoBranches(project.project_id, repoUrl)
    dismissDatalistIfPicked(inputEl, branches, value)
  }

  return {
    normalizeRepoUrlKey,
    editBaseBranchDatalistId,
    isEditBranchCustomInput,
    shouldUseEditBaseBranchSelect,
    getEditBaseBranchSelectValue,
    exitEditBranchCustomInput,
    onEditBaseBranchSelect,
    datalistListAttr,
    restoreDatalistOnFocus,
    onEditBaseBranchInput,
    onEditBaseBranchChange,
  }
}
