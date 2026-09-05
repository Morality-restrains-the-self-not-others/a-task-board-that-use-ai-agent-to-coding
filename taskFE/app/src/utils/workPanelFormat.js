/**
 * WorkPanel：公司昵称解析与 API 错误结构扁平化。
 */

const API_FIELD_LABELS = Object.freeze({
  title: '标题',
  description: '描述',
  projects: '关联项目',
  owner: '负责人',
  assignees: '协作者',
  branch_strategy: '分支策略',
  priority: '优先级',
  non_field_errors: '请求错误',
})

const GO_TASK_ERROR_MESSAGES = Object.freeze({
  forbidden: '您没有权限修改此任务',
  'not found': '任务不存在或已被删除',
  'invalid json': '请求数据格式无效，请刷新页面后重试',
  'projects: one task can only link one project': '一个任务只能关联一个项目',
})

export function humanizeGoTaskErrorMessage(errorText) {
  const raw = String(errorText || '').trim()
  if (!raw) return ''
  if (GO_TASK_ERROR_MESSAGES[raw]) return GO_TASK_ERROR_MESSAGES[raw]
  const baseBranchMatch = raw.match(/^projects\[(\d+)\] base_branch required$/)
  if (baseBranchMatch) {
    const index = Number(baseBranchMatch[1]) + 1
    return `请为关联项目第 ${index} 个仓库配置基准分支`
  }
  const projectObjectMatch = raw.match(/^projects\[(\d+)\] must be object$/)
  if (projectObjectMatch) {
    const index = Number(projectObjectMatch[1]) + 1
    return `关联项目第 ${index} 项数据格式无效`
  }
  return raw
}

/**
 * 从 API 错误响应体解析用户可读文案（兼容 Go task 服务的 error 与 Django/DRF 的 detail）。
 */
export function resolveApiErrorMessage(data, { fallback = '操作失败', httpStatus = '' } = {}) {
  if (data == null || data === '') {
    return httpStatus ? `${fallback}（HTTP ${httpStatus}）` : fallback
  }
  if (typeof data === 'string') {
    const text = data.trim()
    return text || (httpStatus ? `${fallback}（HTTP ${httpStatus}）` : fallback)
  }
  if (typeof data.detail === 'string' && data.detail.trim()) {
    return data.detail.trim()
  }
  if (Array.isArray(data.detail) && data.detail.length > 0) {
    const formatted = formatApiErrorMessage(data.detail)
    if (formatted) return formatted
  }
  if (typeof data.error === 'string' && data.error.trim()) {
    return humanizeGoTaskErrorMessage(data.error)
  }
  if (typeof data.message === 'string' && data.message.trim()) {
    return data.message.trim()
  }
  const formattedBody = formatApiErrorMessage(
    Object.fromEntries(
      Object.entries(data).map(([key, value]) => [API_FIELD_LABELS[key] || key, value]),
    ),
  )
  if (formattedBody) return formattedBody
  return httpStatus ? `${fallback}（HTTP ${httpStatus}）` : fallback
}

export function extractCompanyNickname(profileData, tenantIdValue) {
  const profileItems = Array.isArray(profileData?.company_nicknames) ? profileData.company_nicknames : []
  const matchedItem = profileItems.find((item) => String(item.company_id) === String(tenantIdValue))
  if (matchedItem?.member_name) {
    return String(matchedItem.member_name).trim()
  }
  return ''
}

export function formatApiErrorMessage(errorData, parentKey = '') {
  if (!errorData) return ''

  if (typeof errorData === 'string') {
    return parentKey ? `${parentKey}: ${errorData}` : errorData
  }

  if (Array.isArray(errorData)) {
    const items = errorData.map((item) => formatApiErrorMessage(item, parentKey)).filter(Boolean)
    return items.join('\n')
  }

  if (typeof errorData === 'object') {
    const entries = Object.entries(errorData)
      .map(([key, value]) => {
        const normalizedKey = key === 'non_field_errors' ? '请求错误' : key
        return formatApiErrorMessage(value, normalizedKey)
      })
      .filter(Boolean)
    return entries.join('\n')
  }

  return String(errorData)
}
