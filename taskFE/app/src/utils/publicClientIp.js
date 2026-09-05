import { apiFetch } from './apiUtils.js'

let cachedIp = ''
let inflight = null

/**
 * 获取当前浏览器请求在边缘看到的公网客户端 IP（taskAuth 回显）。
 * @param {{ force?: boolean }} [opts] force=true 时忽略模块缓存，创建/启动时强制查询最新 IP
 * @returns {Promise<string>} 非空 IP；失败时抛错
 */
export async function fetchPublicClientIp(opts = {}) {
  const force = Boolean(opts && opts.force)
  if (!force && cachedIp) {
    return cachedIp
  }
  if (!force && inflight) {
    return inflight
  }
  const run = (async () => {
    const res = await apiFetch('/api/accounts/users/client-ip/', {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!res.ok) {
      throw new Error(`client-ip http ${res.status}`)
    }
    const data = await res.json().catch(() => ({}))
    const ip = String(data?.ip || '').trim()
    if (!ip) {
      throw new Error('client-ip empty')
    }
    cachedIp = ip
    return ip
  })()
  if (!force) {
    inflight = run
  }
  try {
    return await run
  } finally {
    if (!force && inflight === run) {
      inflight = null
    }
  }
}

/**
 * 自动创建安全组场景：创建时强制查询用户公网 IP，供写入 start-vm 请求体。
 * @returns {Promise<string>} 成功返回 IP；失败返回空串（由服务端 XFF 兜底）
 */
export async function queryClientPublicIpForAutoSg() {
  try {
    return String(await fetchPublicClientIp({ force: true }) || '').trim()
  } catch (e) {
    console.warn('[publicClientIp] 创建时查询公网 IP 失败，将由服务端请求头解析:', e)
    return ''
  }
}

/** @internal 仅测试用 */
export function __resetPublicClientIpCacheForTests() {
  cachedIp = ''
  inflight = null
}
