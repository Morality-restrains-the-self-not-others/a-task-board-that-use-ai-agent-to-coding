/**
 * 导航栏公司切换：目标 URL 构造。
 *
 * 「工作面板」位置的公司下拉语义：切换公司后进入目标公司工作面板，
 * 不保留旧页面路径（避免 profile/pricing/人员页等同路径污染），
 * 并清除跨租户 query（如 workspace_id），仅可选保留 accessCode。
 */

/**
 * @param {unknown} companyId
 * @param {string} [currentHref] 完整当前 URL（用于读取 accessCode）
 * @returns {string|null} 目标 href；companyId 为空时返回 null
 */
export function buildCompanySwitchHref(companyId, currentHref = '') {
  const id = String(companyId ?? '').trim()
  if (!id) return null

  let accessCode = ''
  try {
    if (currentHref) {
      const u = new URL(currentHref, 'http://local.invalid')
      accessCode = String(u.searchParams.get('accessCode') || '').trim()
    }
  } catch (_) {
    accessCode = ''
  }

  let href = `/tenant/${id}/work-panel/`
  if (accessCode) {
    href += `?accessCode=${encodeURIComponent(accessCode)}`
  }
  return href
}
