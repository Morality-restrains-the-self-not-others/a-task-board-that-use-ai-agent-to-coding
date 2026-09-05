/** Provider-agnostic labels for create-task project repo access status. */

const ACCESS_STATUS_LABELS = {
  accessible: '可访问',
  needs_auth: '未授权',
  not_accessible: '不可访问',
  not_configured: '未设置',
  uncheckable: '无法检查',
}

/**
 * @param {{ is_accessible?: boolean, access_status?: string, message?: string } | null | undefined} data
 * @returns {'可访问'|'未授权'|'不可访问'|'未设置'|'无法检查'}
 */
export function mapRepoAccessApiToLabel(data) {
  if (!data || typeof data !== 'object') {
    return '不可访问'
  }
  const status = String(data.access_status || '').trim()
  if (status && ACCESS_STATUS_LABELS[status]) {
    return ACCESS_STATUS_LABELS[status]
  }
  if (data.is_accessible) {
    return '可访问'
  }
  const message = String(data.message || '')
  if (message.includes('未设置Git仓库')) {
    return '未设置'
  }
  if (message.includes('SSH') || message.includes('无法自动验证')) {
    return '无法检查'
  }
  if (/未绑定|未授权|OAuth|凭据|authorization/i.test(message)) {
    return '未授权'
  }
  return '不可访问'
}
