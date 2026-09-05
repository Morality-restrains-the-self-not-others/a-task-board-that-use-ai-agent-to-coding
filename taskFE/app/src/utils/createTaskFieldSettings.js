/** 创建任务可选字段显隐：与 taskProjectService knownCreateTaskFieldKeys 对齐。 */

export const CREATE_TASK_FIELD_SETTING_KEYS = Object.freeze([
  'description',
  'task_kind',
  'code_lang',
  'structured_fields',
  'project_branch',
  'feature_params',
  'priority',
  'due_date',
  'auto_run',
  'operator',
  'owner',
  'assignees',
])

export const CREATE_TASK_FIELD_SETTING_LABELS = Object.freeze({
  description: '任务描述',
  task_kind: '任务类型',
  code_lang: '主要编程语言',
  structured_fields: '结构化任务说明',
  project_branch: '项目 / 分支策略',
  feature_params: '智能体资源配置',
  priority: '优先级',
  due_date: '截止日期',
  auto_run: '自动运行',
  operator: '操作员',
  owner: '负责人',
  assignees: '协作者',
})

/** 无工作区配置时：主要编程语言、结构化任务说明默认关闭，其余可选字段默认开启。 */
const DEFAULT_DISABLED_CREATE_TASK_FIELD_KEYS = Object.freeze([
  'code_lang',
  'structured_fields',
])

export function defaultCreateTaskFieldSettings() {
  const out = {}
  for (const key of CREATE_TASK_FIELD_SETTING_KEYS) {
    out[key] = !DEFAULT_DISABLED_CREATE_TASK_FIELD_KEYS.includes(key)
  }
  return out
}

/**
 * @param {unknown} raw
 * @returns {Record<string, boolean>}
 */
export function normalizeCreateTaskFieldSettings(raw) {
  const out = defaultCreateTaskFieldSettings()
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return out
  }
  for (const key of CREATE_TASK_FIELD_SETTING_KEYS) {
    if (!(key in raw)) continue
    const v = raw[key]
    if (typeof v === 'boolean') {
      out[key] = v
      continue
    }
    if (typeof v === 'string') {
      const s = v.trim().toLowerCase()
      out[key] = !(s === 'false' || s === '0' || s === 'no' || s === 'off')
      continue
    }
    if (typeof v === 'number') {
      out[key] = v !== 0
      continue
    }
    out[key] = true
  }
  return out
}

/**
 * @param {unknown} body
 * @returns {Record<string, boolean>}
 */
export function createTaskFieldSettingsFromResponse(body) {
  if (body && typeof body === 'object' && body.fields && typeof body.fields === 'object') {
    return normalizeCreateTaskFieldSettings(body.fields)
  }
  return defaultCreateTaskFieldSettings()
}

/**
 * @param {Record<string, boolean>|null|undefined} settings
 * @param {string} key
 */
export function isCreateTaskFieldEnabled(settings, key) {
  if (!key) return true
  if (!settings || typeof settings !== 'object') return true
  if (!(key in settings)) return true
  return settings[key] !== false
}
