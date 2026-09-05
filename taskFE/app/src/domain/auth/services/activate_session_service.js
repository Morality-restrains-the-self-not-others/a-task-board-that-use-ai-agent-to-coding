import { clearCookie } from '../../../utils/cookieUtils.js'
import { storeUserId, clearStoredUserId } from '../../../utils/sessionUserIdUtils.js'
import { setCachedAuthToken, clearCachedAuthToken, apiFetch as defaultApiFetch } from '../../../utils/apiUtils.js'
import {
  setActiveAccount,
  removeSavedAccount,
  getSavedAccount,
  getActiveToken,
} from './saved_accounts_store.js'

const ACTIVATE_PATH = '/api/accounts/users/activate-session/'

function clearStaleSessionCookie() {
  clearCookie('sessionid')
}

/**
 * 调用 activate-session，写入激活凭据并 upsert 账号槽。
 */
export async function activateSavedAccountSession({
  userId,
  token,
  apiFetch,
  username = '',
  avatarUrl = null,
}) {
  const uid = String(userId ?? '').trim()
  const tok = String(token ?? '').trim()
  if (!uid) throw new Error('activateSavedAccountSession: userId is required')
  if (!tok) throw new Error('activateSavedAccountSession: token is required')
  // apiFetch 可注入（测试），缺省使用全局 apiFetch
  const doFetch = typeof apiFetch === 'function' ? apiFetch : defaultApiFetch

  // getActiveToken 读取本机账号槽，失败时 previousToken 留空；仅用于失败时恢复
  // 缓存 token，缺失不影响 activate-session 主路径（cookie 兜底）。
  let previousToken = null
  try {
    previousToken = await getActiveToken()
  } catch {
    /* previousToken 留空即可 */
  }
  setCachedAuthToken(tok)

  const response = await doFetch(ACTIVATE_PATH, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ user_id: uid }),
  })

  if (!response.ok) {
    let detail = `activate-session failed: ${response.status}`
    try {
      const errBody = await response.json()
      detail = errBody.error || errBody.detail || detail
    } catch { /* keep status-based detail */ }
    if (response.status === 401) {
      try { await removeSavedAccount(uid) } catch { /* ignore */ }
    }
    if (previousToken && previousToken !== tok) {
      setCachedAuthToken(previousToken)
    } else if (response.status === 401) {
      clearCachedAuthToken()
    }
    const err = new Error(detail)
    err.status = response.status
    throw err
  }

  const data = await response.json()
  const returnedId = String(data?.user?.id ?? uid).trim()
  const returnedToken = String(data?.token ?? tok).trim()
  if (!returnedId) throw new Error('activate-session response missing user.id')
  if (!returnedToken) throw new Error('activate-session response missing token')

  setCachedAuthToken(returnedToken)
  clearStaleSessionCookie()

  const nameFromResp = String(data?.user?.username ?? username ?? '').trim()
  const avatarFromResp = data?.user?.avatar_url ?? data?.user?.avatarUrl ?? avatarUrl ?? null
  await setActiveAccount({
    userId: returnedId,
    username: nameFromResp,
    avatarUrl: avatarFromResp,
    token: returnedToken,
  })

  return {
    userId: returnedId,
    token: returnedToken,
    user: data.user,
    redirectUrl: data.redirect_url || null,
  }
}

/**
 * 将当前登录结果写入激活凭据与槽。
 */
export async function persistLoginAccountSlot({ userId, token, username = '', avatarUrl = null }) {
  const uid = String(userId ?? '').trim()
  const tok = String(token ?? '').trim()
  if (!uid || !tok) throw new Error('persistLoginAccountSlot requires userId and token')
  storeUserId(uid)
  setCachedAuthToken(tok)
  return setActiveAccount({ userId: uid, token: tok, username, avatarUrl })
}

/**
 * 登录 API 成功后的凭据落盘。
 *
 * 主路径：调用 activate-session 让服务端 Set-Cookie 落 HttpOnly userId+token
 * 会话 cookie（浏览器导航场景的认证凭据；后端 forward-auth 的 userId cookie
 * 回退要求存在有效 auth_customtoken 行 + 会话 cookie，见 OPT-20260807-067）。
 * 失败（网络/服务不可达）静默回退到账号槽写盘，不阻断登录跳转。
 */
export async function persistLoginSuccessCredentials(data) {
  const user = data?.user
  const token = data?.token
  if (user?.id && token) {
    const uid = String(user.id)
    const username = String(user.username || user.email || '')
    try {
      const activated = await activateSavedAccountSession({
        userId: uid,
        token: String(token),
        username,
        avatarUrl: user.avatar_url ?? user.avatarUrl ?? null,
      })
      // activate-session 成功 = 服务端已确认身份；将服务端返回的权威 userId 同步
      // 到本地主存储（currentUserId）。此前仅回退分支写 storeUserId，导致模拟登录
      // （/api/system-admin/users/{uid}/impersonate/ → persist(data)）或账号切换后
      // localStorage 残留旧账号 ID —— 模拟会话下认证头已是目标用户，前端却以残留
      // 模拟者 ID 构造 API 路径（如 /api/git-identities/user/{id}/），后端
      // authUserID != urlUserID 判定 403 → 页面报「获取身份列表失败」。
      // 以服务端返回为准（returnedId = data.user.id ?? uid），与回退分支语义一致。
      storeUserId(activated?.userId || uid)
      return
    } catch {
      // activate-session 失败 → 回退既有行为（账号槽 / 内存 + userId cookie）
    }
  }
  if (user?.id && token) {
    try {
      await persistLoginAccountSlot({
        userId: String(user.id),
        token: String(token),
        username: String(user.username || user.email || ''),
        avatarUrl: user.avatar_url ?? user.avatarUrl ?? null,
      })
      return
    } catch {
      storeUserId(user.id)
      setCachedAuthToken(token)
      return
    }
  }
  if (user?.id) storeUserId(user.id)
  if (token) setCachedAuthToken(token)
}

export function resolveSwitchHref(currentPath, newUserId) {
  const uid = String(newUserId ?? '').trim()
  const raw = String(currentPath ?? '/')
  if (!uid) return raw

  const qIndex = raw.indexOf('?')
  let path = qIndex >= 0 ? raw.slice(0, qIndex) : raw
  const query = qIndex >= 0 ? raw.slice(qIndex + 1) : ''

  if (/\/tenant\//.test(path)) {
    return `/user/${uid}/profile/`
  }

  path = path.replace(/\/user\/[^/]+\//, `/user/${uid}/`)

  if (!query) return path
  const params = new URLSearchParams(query)
  params.delete('workspace_id')
  const qs = params.toString()
  return qs ? `${path}?${qs}` : path
}

export { getSavedAccount, removeSavedAccount }
