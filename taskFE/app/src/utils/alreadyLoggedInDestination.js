import { sanitizeOidcResumeNext } from './oidcResumeUrl.js'
import { PostLoginReturnUrl } from '../domain/auth/value_objects/post_login_return_url_value_object.js'

export const WORK_PANEL_LABEL = '工作面板'
export const SYSTEM_ADMIN_LABEL = '系统管理'

function firstQueryValue(raw) {
  if (Array.isArray(raw)) {
    return typeof raw[0] === 'string' ? raw[0] : ''
  }
  return typeof raw === 'string' ? raw : ''
}

/**
 * 已登录用户命中登录页时的跳转目标解析（OPT-20260810-044）。
 * 优先级与登录成功一致：OIDC 回跳 next → 同源业务 next → 工作面板 → 系统管理。
 *
 * @param {string | string[] | null | undefined} nextRaw 登录页 next 查询参数
 * @param {{ current_company?: { id?: string }, companies?: Array<{ id?: string }> }} [userInfo] /me/ 载荷
 * @returns {{ href: string, label: string }}
 */
export function resolveAlreadyLoggedInDestination(nextRaw, userInfo = {}) {
  const raw = firstQueryValue(nextRaw)

  // 1) OIDC 回跳 next（与登录成功一致）
  const resumeNext = sanitizeOidcResumeNext(raw)
  if (resumeNext) {
    return { href: resumeNext, label: resumeNext }
  }

  // 2) 同源业务 next
  const businessNext = PostLoginReturnUrl.normalize(raw)
  if (businessNext) {
    return { href: businessNext, label: businessNext }
  }

  // 3) 工作面板：优先 current_company.id，兜底 companies[0].id（与 Navbar currentTenant 解析一致）
  const currentCompanyId = String(userInfo?.current_company?.id ?? '').trim()
  const companies = Array.isArray(userInfo?.companies) ? userInfo.companies : []
  const tenantId = currentCompanyId || String(companies[0]?.id ?? '').trim()
  if (tenantId) {
    return { href: `/tenant/${tenantId}/work-panel/`, label: WORK_PANEL_LABEL }
  }

  // 4) 兜底：与登录成功默认落点一致
  return { href: '/system-admin/', label: SYSTEM_ADMIN_LABEL }
}
