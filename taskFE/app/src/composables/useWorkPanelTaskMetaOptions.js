import { useWorkPanelTaskKindOptions } from './useWorkPanelTaskKindOptions.js'
import { useWorkPanelCodeLangOptions } from './useWorkPanelCodeLangOptions.js'
import { useWorkPanelCreateTaskFieldSettings } from './useWorkPanelCreateTaskFieldSettings.js'

/** @param {unknown} task @param {{ trim?: boolean }} [opts] */
export function pickTaskMetaFields(task, opts = {}) {
  const trim = opts.trim !== false
  const kind = task?.task_kind != null ? String(task.task_kind) : ''
  const lang = task?.code_lang != null ? String(task.code_lang) : ''
  return {
    task_kind: trim ? kind.trim() : kind,
    code_lang: trim ? lang.trim() : lang,
  }
}

/**
 * 创建任务元数据：任务类型 / 编程语言可选值 + 可选字段显隐。
 * @param {Parameters<typeof useWorkPanelTaskKindOptions>[0]} deps
 */
export function useWorkPanelTaskMetaOptions(deps) {
  const taskKind = useWorkPanelTaskKindOptions(deps)
  const codeLang = useWorkPanelCodeLangOptions(deps)
  const fieldSettings = useWorkPanelCreateTaskFieldSettings(deps)

  const fetchTaskMetaOptions = async () => {
    await Promise.all([
      taskKind.fetchTaskKindOptions(),
      codeLang.fetchCodeLangOptions(),
      fieldSettings.fetchCreateTaskFieldSettings(),
    ])
  }

  const resetTaskMetaOptions = () => {
    taskKind.resetTaskKindOptions()
    codeLang.resetCodeLangOptions()
    fieldSettings.resetCreateTaskFieldSettings()
  }

  return {
    taskKindOptions: taskKind.taskKindOptions,
    codeLangOptions: codeLang.codeLangOptions,
    createTaskFieldSettings: fieldSettings.createTaskFieldSettings,
    fetchTaskMetaOptions,
    resetTaskMetaOptions,
  }
}
