/* @alias:util-referral-access-code */
import { apiFetch } from './apiUtils.js'
import { getReferralCode, setReferralCode } from './referralUtils.js'

const STORAGE_KEY = 'referral_access_code'
// 自身默认推荐码会话级缓存：登录后地址栏 accessCode 归一化时避免每页重复请求状态接口
const OWN_CODE_STORAGE_KEY = 'referral_own_access_code'
const OWN_CODE_STATUS_URL = '/api/accounts/users/referral-codes/status/'

export function normalizeReferralAccessCode(code) {
  return String(code || '').trim()
}

export function readStoredReferralAccessCode() {
  try {
    const v = sessionStorage.getItem(STORAGE_KEY)
    return typeof v === 'string' ? v.trim() : ''
  } catch {
    return ''
  }
}

export function writeStoredReferralAccessCode(code) {
  const normalized = normalizeReferralAccessCode(code)
  try {
    if (normalized) {
      sessionStorage.setItem(STORAGE_KEY, normalized)
    } else {
      sessionStorage.removeItem(STORAGE_KEY)
    }
  } catch {
    /* ignore quota / private mode */
  }
  return normalized
}

export function captureReferralAccessCodeFromSearch(search) {
  const raw = String(search || '')
  const params = new URLSearchParams(raw.startsWith('?') ? raw.slice(1) : raw)
  const fromQuery = normalizeReferralAccessCode(params.get('accessCode') || params.get('access_code'))
  if (fromQuery) {
    return writeStoredReferralAccessCode(fromQuery)
  }
  return readStoredReferralAccessCode()
}

export function readCachedOwnReferralAccessCode() {
  try {
    const v = sessionStorage.getItem(OWN_CODE_STORAGE_KEY)
    return typeof v === 'string' ? v.trim() : ''
  } catch {
    return ''
  }
}

export function writeCachedOwnReferralAccessCode(code) {
  const normalized = normalizeReferralAccessCode(code)
  try {
    if (normalized) {
      sessionStorage.setItem(OWN_CODE_STORAGE_KEY, normalized)
    } else {
      sessionStorage.removeItem(OWN_CODE_STORAGE_KEY)
    }
  } catch {
    /* ignore quota / private mode */
  }
  return normalized
}

/**
 * 获取当前登录用户自己的默认推荐码（状态接口 access_code 字段，后端 ensureUserShareCode 保证存在）。
 * fetchImpl 可注入以便单测；默认走 apiFetch（带 API 前缀与 in-flight GET 去重）。
 * @returns {Promise<{ok: boolean, code: string}>} ok=false 表示网络/服务异常（调用方应保持 URL 原样）
 */
export async function fetchOwnReferralAccessCode(fetchImpl) {
  const doFetch = fetchImpl || apiFetch
  try {
    const response = await doFetch(OWN_CODE_STATUS_URL, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!response.ok) return { ok: false, code: '' }
    const data = await response.json().catch(() => ({}))
    return { ok: true, code: String(data.access_code || '').trim() }
  } catch (error) {
    console.warn('[referral] 获取自身推荐码失败，地址栏 accessCode 保持原样:', error)
    return { ok: false, code: '' }
  }
}

/**
 * 登录态下把地址栏 ?accessCode= 归一化为自己的默认推荐码（OPT-20260824-005 闭环）。
 *
 * 语义：地址栏 accessCode 是分享链接溯源参数（他人推荐码），与本页/本账号「自己的推荐码」不同；
 * 登录后继续展示会造成复制地址栏分享时归属错误。本函数：
 * 1. 无 URL 参数且旧机制 localStorage 'referralCode' 为空 → 直接返回（零开销）
 * 2. 解析自己的码：ownCode 参数 > 会话缓存 > 状态接口（成功后写缓存）
 * 3. 同步旧机制 localStorage 'referralCode'（initReferralCode 点击处理器按它给同源链接附加
 *    accessCode，不同步会让他人码在下次点击时卷土重来）
 * 4. URL 参数与自己的码不同 → history.replaceState 重写（无历史记录）；API 确认无码 → 移除参数；
 *    API 不可达 → 保持 URL 原样（不破坏溯源）
 *
 * @param {{ownCode?: string, fetchImpl?: Function}} [opts]
 * @returns {Promise<boolean>} URL 是否被重写
 */
export async function normalizeUrlAccessCodeToOwn({ ownCode, fetchImpl } = {}) {
  if (typeof window === 'undefined') return false
  // OPT-20260827-014: 项目分享链接（/projects/ 及 /tenant/:tid/projects/:id/）的
  // ?accessCode= 是入站分享/授权码，登录后不得用「自己的推荐码」覆盖，否则入站
  // 分享溯源归属丢失。其他页面（如 /profile/referral/）仍保持归一化行为。
  if (/\/projects\//.test(window.location.pathname)) return false
  const params = new URLSearchParams(window.location.search)
  const key = params.has('accessCode') ? 'accessCode' : params.has('access_code') ? 'access_code' : ''
  const current = key ? String(params.get(key) || '').trim() : ''
  const stored = getReferralCode()
  if (!current && !stored) return false

  let own = normalizeReferralAccessCode(ownCode)
  let resolvedFromApi = false
  if (!own) own = readCachedOwnReferralAccessCode()
  if (!own) {
    const res = await fetchOwnReferralAccessCode(fetchImpl)
    resolvedFromApi = res.ok
    own = res.code
    if (own) writeCachedOwnReferralAccessCode(own)
  }

  if (own && stored !== own) setReferralCode(own)
  if (!current) return false
  if (own && current === own) return false
  if (!own && !resolvedFromApi) return false

  if (own) {
    params.set(key, own)
  } else {
    params.delete(key)
  }
  const search = params.toString()
  window.history.replaceState(history.state, '', window.location.pathname + (search ? `?${search}` : ''))
  return true
}
