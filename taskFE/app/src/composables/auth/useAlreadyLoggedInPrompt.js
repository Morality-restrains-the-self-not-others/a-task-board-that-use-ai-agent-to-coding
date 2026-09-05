import { apiFetch } from '../../utils/apiUtils.js'
import { resolveAlreadyLoggedInDestination } from '../../utils/alreadyLoggedInDestination.js'

const ME_PATH = '/api/accounts/users/me/'

function inProgressFlowParams(search) {
  const q = new URLSearchParams(search)
  return Boolean(
    // OIDC 回跳（code/state）由 resumeOidcFlowIfReady 处理，勿叠加弹窗
    (q.get('code') && q.get('state')) ||
      // 微信回调（token/error/bound）由 handleWeChatCallback 处理
      q.get('wechat_token') ||
      q.get('wechat_error') ||
      q.get('wechat_bound') ||
      // 激活结果弹窗
      q.get('activated'),
  )
}

/**
 * 已登录用户命中登录页时，弹窗询问是否跳转（OPT-20260810-044）。
 * 不自动 router.replace —— 确认才跳转，取消留在登录页。
 *
 * 依赖均可注入以便单测：
 * - getCurrentUser：同步取当前会话（默认 window.currentUser，由 Navbar 填充）
 * - fetchMe：拉取 /me/（默认 apiFetch，与 Navbar 的并发请求经 GET 去重合并）
 * - modalConfirm：确认弹窗（默认 modalService.confirm，resolve=确认 / reject=取消）
 * - navigate：跳转（默认 window.location.href）
 */
export function useAlreadyLoggedInPrompt({
  getCurrentUser = () => window.currentUser,
  fetchMe = async () => {
    const resp = await apiFetch(ME_PATH, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    return resp.ok ? resp.json() : null
  },
  modalConfirm = (message) =>
    import('../../utils/modalService.js').then((m) => m.default.confirm(message, '提示', '前往', '留在本页')),
  navigate = (href) => {
    window.location.href = href
  },
  getSearch = () => window.location.search,
} = {}) {
  async function resolveAuthenticatedUser() {
    const current = getCurrentUser()
    if (current && current.isAuthenticated) return current
    return fetchMe()
  }

  async function maybePromptAlreadyLoggedIn() {
    if (inProgressFlowParams(getSearch())) return false

    let userInfo
    try {
      userInfo = await resolveAuthenticatedUser()
    } catch {
      return false
    }
    if (!userInfo) return false

    // window.currentUser 带 isAuthenticated 标记；/me/ 载荷无该字段，用 userId 判定
    const authenticated =
      userInfo.isAuthenticated === true || Boolean(String(userInfo.userId ?? '').trim())
    if (!authenticated) return false

    const nextRaw = new URLSearchParams(getSearch()).get('next')
    const { href, label } = resolveAlreadyLoggedInDestination(nextRaw, userInfo)
    try {
      await modalConfirm(`您已登录，是否前往 ${label}？`)
      navigate(href)
      return true
    } catch {
      // 用户取消：留在登录页
      return false
    }
  }

  return { maybePromptAlreadyLoggedIn }
}
