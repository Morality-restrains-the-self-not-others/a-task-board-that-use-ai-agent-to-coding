/**
 * 租户访问校验失败时的导航确认（禁止静默回弹 profile，避免循环跳转）。
 *
 * WorkPanel / useTenantAccessGuard 等在 /me/ 返回 403/404 时调用：
 * - 弹窗让用户选择是否离开
 * - 确认 → 整页跳转目标（默认个人资料）
 * - 取消 → 留在当前页，继续降级初始化
 */

/**
 * @param {object} options
 * @param {string} options.targetHref - 用户确认后前往的地址
 * @param {string} [options.message]
 * @param {string} [options.title]
 * @param {string} [options.confirmText]
 * @param {string} [options.cancelText]
 * @param {(msg: string, title?: string, confirmText?: string, cancelText?: string) => Promise<unknown>} [options.confirm]
 * @param {(href: string) => void} [options.navigate]
 * @returns {Promise<'navigated'|'stayed'>}
 */
export async function promptNavigateOnTenantAccessDenied(options = {}) {
  const targetHref = String(options.targetHref || '').trim()
  if (!targetHref) return 'stayed'

  const message =
    options.message ||
    '无法验证当前租户的访问权限（可能是会话异常、无权访问或租户已失效）。是否离开本页并前往个人资料？\n选择「留在本页」可避免与导航来源之间循环跳转。'
  const title = options.title || '无法访问工作面板'
  const confirmText = options.confirmText || '前往个人资料'
  const cancelText = options.cancelText || '留在本页'

  const confirmFn =
    options.confirm ||
    (async (msg, t, ok, cancel) => {
      const { default: modalService } = await import('./modalService.js')
      return modalService.confirm(msg, t, ok, cancel)
    })

  const navigate =
    options.navigate ||
    ((href) => {
      if (typeof window !== 'undefined') {
        window.location.href = href
      }
    })

  try {
    await confirmFn(message, title, confirmText, cancelText)
    navigate(targetHref)
    return 'navigated'
  } catch {
    return 'stayed'
  }
}

/**
 * 由 userId 构造个人资料 href。
 * @param {string|null|undefined} userId
 * @returns {string}
 */
export function buildProfileHref(userId) {
  const uid = String(userId ?? '').trim()
  return uid ? `/user/${uid}/profile/` : '/profile/'
}
