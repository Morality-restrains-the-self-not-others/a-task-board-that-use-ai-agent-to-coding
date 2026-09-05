import { clearCookie, getCookie, setCookie } from './cookieUtils.js'
import { apiFetch } from './apiUtils.js'

const PROFILE_PATH = '/api/accounts/users/profile/'
const USER_ID_COOKIE_DAYS = 30
const USER_ID_STORAGE_KEY = 'currentUserId'

let inflightResolve = null

// OPT-20260824-064: 本地 currentUserId 服务端权威校验回填。
const VALIDATE_USER_ID_TTL_MS = 60_000
// 与 useImpersonateUser.js 的 IMPERSONATOR_BACKUP_KEY 同值；此处不 import 以避免
// sessionUserIdUtils ← useImpersonateUser 循环依赖。
const IMPERSONATOR_BACKUP_KEY = 'impersonatorAccountBackup'
let lastValidatedAt = 0
let validationInflight = null

function isImpersonationSession() {
  try {
    return !!(
      typeof sessionStorage !== 'undefined' && sessionStorage.getItem(IMPERSONATOR_BACKUP_KEY)
    )
  } catch {
    return false
  }
}

/**
 * 用 profile API 的权威 user_id 校验本地 currentUserId（OPT-20260824-064）。
 * 命中不一致时回填 storeUserId 并 console.warn；返回服务端 user_id（失败/401 返回 ''）。
 * 模拟会话由 status 端点自愈接管，调用方负责在 isImpersonationSession() 时跳过。
 */
export async function validateStoredUserIdAgainstServer(stored) {
  if (!stored) return ''
  if (validationInflight) return validationInflight
  validationInflight = (async () => {
    try {
      const response = await apiFetch(PROFILE_PATH, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (!response.ok) return ''
      const profile = await response.json()
      const serverId = extractUserIdFromProfile(profile)
      if (serverId && serverId !== stored) {
        storeUserId(serverId)
        console.warn(
          `[sessionUserId] stored currentUserId=${stored} diverged from server=${serverId}; refilled`,
        )
      }
      return serverId
    } catch {
      return ''
    } finally {
      validationInflight = null
    }
  })()
  return validationInflight
}

/** 校验 TTL 缓存重置（测试用；避免用例间 lastValidatedAt 污染）。 */
export function resetUserIdValidationCache() {
  lastValidatedAt = 0
  validationInflight = null
}

function maybeValidateStoredUserId(stored) {
  if (!stored || isImpersonationSession()) return
  const now = Date.now()
  if (now - lastValidatedAt < VALIDATE_USER_ID_TTL_MS) return
  lastValidatedAt = now
  void validateStoredUserIdAgainstServer(stored)
}

export function extractUserIdFromProfile(profile) {
  return String(profile?.user_id ?? profile?.user?.id ?? '').trim()
}

/**
 * 从 localStorage 读取 currentUserId（主存储，跨 Tab 不共享）。
 * 回退到 cookie（OIDC SSO 跨端口桥接所需，由 activate-session 设置为 HttpOnly）。
 */
export function getStoredUserId() {
  try {
    const fromStorage = localStorage.getItem(USER_ID_STORAGE_KEY)
    if (fromStorage) return String(fromStorage).trim()
  } catch {}
  return String(getCookie('userId') || '').trim()
}

/**
 * 异步获取当前 userId（本机账号槽活跃账号）。
 */
export async function getStoredUserIdAsync() {
  const { getCurrentUserId } = await import('../domain/auth/services/saved_accounts_store.js')
  return String(await getCurrentUserId() || '').trim()
}

/**
 * 写入 userId 到 localStorage（主）和 cookie（回退，兼容旧代码）。
 * OPT-20260807-004 后语义统一为「提示性 + 服务端为准」：服务端仅接受
 * activate-session 落下的 HttpOnly 签名 cookie（<id>.<ts>.<hmac>，user_id_cookie.go），
 * 本函数写入的 JS cookie 不可被服务端用于认证（未被签名即被拒），仅作前端
 * localStorage 之外的弱回退展示；HttpOnly cookie 存在时浏览器会忽略同名 JS 写入。
 */
export function storeUserId(userId) {
  const uid = String(userId || '').trim()
  if (!uid) return
  try { localStorage.setItem(USER_ID_STORAGE_KEY, uid) } catch {}
  setCookie('userId', uid, USER_ID_COOKIE_DAYS)
}

/**
 * 清除 userId 存储。
 */
export function clearStoredUserId() {
  try { localStorage.removeItem(USER_ID_STORAGE_KEY) } catch {}
  // HttpOnly cookie 只能由服务端清除，此处兜底清除 JS 可写的两种变体：
  // 域变体（Domain=.daydaymoney.com，跨子域共享）+ host-only 旧形态，避免
  // 残留 userId 在任意子域误导前端登录态展示（clearCookie 双发，见 cookieUtils）。
  try { clearCookie('userId') } catch {}
}

/**
 * profile 校验成功时回填 userId 存储，避免 session 有效但缓存过期/缺失导致业务不可用。
 * 优先写入 localStorage，cookie 作为兼容回退（activate-session 会覆盖为 HttpOnly）。
 */
export function syncUserIdCookieFromProfile(profile) {
  const userId = extractUserIdFromProfile(profile)
  if (!userId) return false
  storeUserId(userId)
  return true
}

/**
 * 微信回调凭据补全（修复整页跳转后 401，如 onboarding 设置公司名时报「服务器错误 (401)」）：
 * 微信回调 URL 仅携带 wechat_token，无 user 信息 — token 只写入内存缓存
 * （persistLoginSuccessCredentials({token}) 无 user 分支），随整页跳转全部丢失；
 * 账号槽未写入时取不到 token，请求不带 Authorization → 网关
 * forward-auth 无任何有效凭据 → 401。
 * 此处解析用户身份并落盘：
 *  - 显式携带 Authorization: Token <wechat_token>（不依赖内存缓存时序 —
 *    此前仅靠 setCachedAuthToken 的模块内存态，异步链路任一环节被打断即丢凭据，
 *    导致跳转 /onboarding/ 后全线 401；显式头保证请求确定性）
 *  - userId cookie + localStorage（网关 forward-auth 的 userId cookie 兜底认证）
 *  - 同步 active 账号槽（token 用回调携带的 wechat_token，避免跳转后
 *    账号槽返回旧 token）
 * 失败重试 1 次（网络抖动/瞬时错误），仍失败则不抛错（不阻断登录跳转，
 * 与修复前行为一致，仅凭据未持久化 — 由 Onboarding 提交 401 恢复跳转兜底）。
 *
 * @param {string} wechatToken 微信回调携带的登录 token
 * @returns {Promise<boolean>} 是否成功解析并持久化凭据
 */
export async function resolveWechatSessionIdentity(wechatToken) {
  const fetchProfile = async () => {
    const response = await apiFetch(PROFILE_PATH, {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        // 显式携带回调 token：profile 请求不依赖 apiFetch 内存缓存，杜绝
        // 「token 未及时 setCachedAuthToken → 请求无凭据 → 401 → 凭据永不落盘」死锁
        Authorization: `Token ${String(wechatToken ?? '')}`,
      },
    })
    return response
  }

  let response
  try {
    response = await fetchProfile()
    if (!response.ok) {
      // 瞬时错误重试 1 次（间隔 300ms），兜底网络抖动
      await new Promise((resolve) => setTimeout(resolve, 300))
      response = await fetchProfile()
    }
  } catch (firstErr) {
    await new Promise((resolve) => setTimeout(resolve, 300))
    try {
      response = await fetchProfile()
    } catch {
      return { ok: false, profile: null }
    }
  }
  if (!response?.ok) return { ok: false, profile: null }

  try {
    const profile = await response.json()
    const userId = extractUserIdFromProfile(profile)
    if (!userId) return { ok: false, profile: null }
    storeUserId(userId)
    // OPT-20260807-005: 补 activate-session — 微信登录用户此前仅落 userId cookie
    // + 账号槽，HttpOnly token cookie 缺失，依赖 userId cookie 兜底路径；这里调用
    // activateSavedAccountSession 让服务端 Set-Cookie 落 HttpOnly userId+token
    // 会话 cookie，与主登录流（persistLoginSuccessCredentials）同构。
    // 失败静默回退：回退到账号槽同步（setActiveAccount），不阻断登录跳转。
    try {
      const { activateSavedAccountSession } = await import(
        '../domain/auth/services/activate_session_service.js'
      )
      await activateSavedAccountSession({
        userId,
        token: String(wechatToken ?? ''),
        // taskAuth profile 真实字段 personal_nickname（兼容 user.username 形态）
        username: String(
          profile?.personal_nickname ?? profile?.username ?? profile?.user?.username ?? '',
        ),
        avatarUrl: profile?.avatar_url ?? profile?.user?.avatarUrl ?? null,
      })
    } catch {
      try {
        const { setActiveAccount } = await import(
          '../domain/auth/services/saved_accounts_store.js'
        )
        await setActiveAccount({
          userId,
          username: String(
            profile?.personal_nickname ?? profile?.username ?? profile?.user?.username ?? '',
          ),
          avatarUrl: profile?.avatar_url ?? profile?.user?.avatarUrl ?? null,
          token: String(wechatToken ?? ''),
        })
      } catch {
        /* 账号槽写入失败：userId cookie 已兜底认证 */
      }
    }
    return { ok: true, profile }
  } catch {
    return { ok: false, profile: null }
  }
}

/**
 * 解析当前登录用户的 numeric user id。
 * 优先读 localStorage，回退 cookie，再回退 profile API。
 */
export async function resolveAuthenticatedUserId() {
  const stored = getStoredUserId()
  if (stored) {
    // OPT-20260824-064: 命中本地缓存时后台做一次服务端权威校验（TTL 限频、
    // 模拟会话跳过），不一致回填 —— 防任何历史来源（模拟会话残留等）的陈旧
    // currentUserId 造成后续 API 403。返回仍用本地值，不阻塞调用方。
    maybeValidateStoredUserId(stored)
    return stored
  }

  if (inflightResolve) return inflightResolve

  inflightResolve = (async () => {
    try {
      const response = await apiFetch(PROFILE_PATH, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (!response.ok) return ''
      const profile = await response.json()
      syncUserIdCookieFromProfile(profile)
      return extractUserIdFromProfile(profile)
    } catch {
      return ''
    } finally {
      inflightResolve = null
    }
  })()

  return inflightResolve
}
