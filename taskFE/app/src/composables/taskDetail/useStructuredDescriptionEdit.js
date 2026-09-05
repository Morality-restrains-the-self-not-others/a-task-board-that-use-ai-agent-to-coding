/**
 * 结构化任务描述编辑逻辑 — 从 TaskDetailTaskIdentityPanel 提取以控制组件行数。
 */
import { ref, watch } from 'vue'
import {
  CREATE_TASK_STRUCTURED_FIELDS,
  parseCreateTaskDescription,
  composeCreateTaskDescription,
} from '../../utils/createTaskDescriptionCompose.js'

/**
 * @param {import('vue').Ref<boolean>} isEditingRef
 * @param {import('vue').Ref<object|null>} editingTaskRef
 */
export function useStructuredDescriptionEdit(isEditingRef, editingTaskRef) {
  const structuredTextFields = CREATE_TASK_STRUCTURED_FIELDS.filter((f) => f.kind !== 'deps')
  const structuredEditMode = ref('structured')

  function hydrateStructuredFields() {
    const task = editingTaskRef?.value
    if (!task || typeof task !== 'object') return
    const parsed = parseCreateTaskDescription(task.description)
    task.description = parsed.description
    for (const def of CREATE_TASK_STRUCTURED_FIELDS) {
      if (task[def.key] == null) {
        task[def.key] = parsed[def.key]
      }
    }
  }

  function composeDescriptionFromStructuredFields() {
    const task = editingTaskRef?.value
    if (!task) return
    task.description = composeCreateTaskDescription(task)
  }

  function switchToStructuredEdit() {
    structuredEditMode.value = 'structured'
    hydrateStructuredFields()
  }

  function switchToRawDescription() {
    structuredEditMode.value = 'raw'
  }

  watch(
    () => isEditingRef?.value && editingTaskRef?.value,
    (active) => {
      if (active) {
        hydrateStructuredFields()
        const parsed = parseCreateTaskDescription(editingTaskRef?.value?.description || '')
        structuredEditMode.value = CREATE_TASK_STRUCTURED_FIELDS.some((f) => parsed[f.key])
          ? 'structured'
          : 'raw'
      }
    },
    { immediate: true },
  )

  return {
    structuredTextFields,
    structuredEditMode,
    hydrateStructuredFields,
    composeDescriptionFromStructuredFields,
    switchToStructuredEdit,
    switchToRawDescription,
  }
}
