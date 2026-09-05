// 租户访问守卫 composable
// 在租户范围视图中调用 /me/ API；403/404 时弹窗确认是否前往个人资料（禁止静默回弹）。
// 当前供租户范围视图选用；见 OPT-20260726-035 / 2026-08-12 工作面板回弹修复。

import { apiFetch } from './apiUtils'
import { getCookie } from './cookieUtils'
import {
  buildProfileHref,
  promptNavigateOnTenantAccessDenied,
} from './tenantAccessDeniedPrompt.js'

/**
 * 解析响应 JSON，失败返回 null。
 * @param {Response} r
 * @returns {Promise<object|null>}
 */
async function parseJsonSafe(r) {
  try {
    return await r.json()
  } catch {
    return null
  }
}

/**
 * 调用 /me/ API 验证当前用户对指定租户的访问权限。
 * 若 API 返回 403/404，弹窗询问是否前往个人资料；确认则整页跳转并返回 null，
 * 取消则返回 null 且不导航（调用方继续降级）。禁止静默 router.replace(profile)。
 * 若 API 不可达或其他错误，返回 null（调用方自行处理降级）。
 *
 * @param {object} options
 * @param {string} options.userId - 当前用户 ID（来自 cookie）
 * @param {string|null} options.tenantId - URL 中的租户 ID（可选）
 * @param {import('vue-router').Router} [options.router] - 保留参数兼容旧调用方（不再用于静默回弹）
 * @returns {Promise<object|null>} 用户数据对象，失败/null 时返回 null
 */
export async function verifyTenantAccess({ userId, tenantId, router: _router }) {
  if (!userId) return null

  const tenantParam = tenantId ? `?tenant_id=${encodeURIComponent(tenantId)}` : ''
  const r = await apiFetch(`/api/accounts/users/me/${tenantParam}`, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })

  if (r.ok) {
    const data = await parseJsonSafe(r)
    return data // null if JSON parse failed
  }

  if (r.status === 403 || r.status === 404) {
    console.warn('[useTenantAccessGuard] /me/ 返回 %d，征求用户是否离开本页', r.status)
    const uid = getCookie('userId') || userId
    await promptNavigateOnTenantAccessDenied({
      targetHref: buildProfileHref(uid),
    })
    return null
  }

  return null
}
