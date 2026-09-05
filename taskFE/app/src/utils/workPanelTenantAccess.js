/**
 * WorkPanel 租户成员资格判定（纯函数）。
 * 返回非 null 时表示「建议离开」的目标 href；调用方须经
 * `promptNavigateOnTenantAccessDenied` 弹窗确认后再导航，
 * **禁止**静默 router.replace（见 49_no_link_click_interception）。
 */

/**
 * @param {{
 *   tenantIdFromUrl: string|null|undefined,
 *   userData: object|null|undefined,
 *   userId: string|null|undefined,
 * }} opts
 * @returns {{ href: string, reason: string } | null} 需跳转时返回目标；可继续访问时返回 null
 */
export function resolveUnauthorizedTenantRedirect({ tenantIdFromUrl, userData, userId }) {
  const tenantId = String(tenantIdFromUrl ?? '').trim()
  if (!tenantId) {
    return null
  }

  // 提取 userId（优先参数传入，其次 userData，最后为空）
  const uid = String(userId ?? userData?.id ?? userData?.user_id ?? '').trim()
  const profileHref = uid ? `/user/${uid}/profile/` : '/profile/'

  // userData 不可用时（profile API 失败等），无法校验租户成员资格，但仍有 userId
  // 时重定向到个人资料页作为安全回退，避免用户卡在无法访问的页面上
  if (!userData || typeof userData !== 'object') {
    if (uid) {
      return { href: profileHref, reason: '无法获取用户公司列表，回退到个人资料页' }
    }
    return null
  }

  const companies = Array.isArray(userData.companies) ? userData.companies : []
  const companyIds = companies
    .map((c) => String(c?.id ?? '').trim())
    .filter(Boolean)

  // 无公司列表但已认证时，回退到个人资料页而非放行到可能无权限的页面
  if (companyIds.length === 0) {
    if (uid) {
      return { href: profileHref, reason: '用户公司列表为空，回退到个人资料页' }
    }
    return null
  }

  if (companyIds.includes(tenantId)) {
    return null
  }

  const currentCompanyId = String(userData.current_company?.id ?? '').trim()
  if (currentCompanyId && currentCompanyId === tenantId) {
    return null
  }

  return {
    href: profileHref,
    reason: '当前账号无权访问该租户工作面板',
  }
}
