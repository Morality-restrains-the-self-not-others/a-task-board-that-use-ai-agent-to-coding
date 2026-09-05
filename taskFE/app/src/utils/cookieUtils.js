/**
 * Cookie 工具函数模块
 * 提供获取和设置 Cookie 的功能
 */

import { resolveSharedCookieDomain } from './ssoCookieDomain.js'

function configuredSsoCookieDomain() {
  const fromEnv = import.meta.env.VITE_SSO_COOKIE_DOMAIN
  if (typeof fromEnv === 'string' && fromEnv.trim()) {
    return fromEnv.trim()
  }
  if (typeof window !== 'undefined') {
    const fromConfig = window.config?.SSO_COOKIE_DOMAIN
    if (typeof fromConfig === 'string' && fromConfig.trim()) {
      return fromConfig.trim()
    }
  }
  return ''
}

/**
 * @returns {string} Domain attribute segment including leading "; domain=..." or empty
 */
function cookieDomainAttribute() {
  if (typeof window === 'undefined' || !window.location?.hostname) {
    return ''
  }
  const domain = resolveSharedCookieDomain(
    window.location.hostname,
    configuredSsoCookieDomain(),
  )
  return domain ? `; domain=${domain}` : ''
}

function cookieSecureAttribute() {
  if (typeof window === 'undefined') return ''
  return window.location?.protocol === 'https:' ? '; Secure' : ''
}

/**
 * 获取指定名称的 Cookie 值
 * @param {string} name - Cookie 名称
 * @returns {string} - Cookie 值，如果不存在则返回空字符串
 */
export function getCookie(name) {
  const cookieValue = document.cookie
    .split('; ')
    .find(row => row.startsWith(name + '='))
    ?.split('=')[1];
  return cookieValue ? decodeURIComponent(cookieValue) : '';
}

/**
 * 设置指定名称的 Cookie 值
 * @param {string} name - Cookie 名称
 * @param {string} value - Cookie 值
 * @param {number} days - 过期天数
 */
export function setCookie(name, value, days) {
  const date = new Date();
  date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
  const expires = "expires=" + date.toUTCString();
  // SameSite=Lax so top-level OIDC navigations to api.* still send the cookie.
  document.cookie =
    name +
    '=' +
    encodeURIComponent(value) +
    ';' +
    expires +
    ';path=/' +
    cookieDomainAttribute() +
    '; SameSite=Lax' +
    cookieSecureAttribute();
}

/**
 * 删除指定名称的 Cookie
 * @param {string} name - Cookie 名称
 */
export function clearCookie(name) {
  if (typeof document === 'undefined') return;
  const domainAttr = cookieDomainAttribute()
  document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/${domainAttr}; SameSite=Lax`
  // Also clear host-only variant left by older builds
  document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; SameSite=Lax`
}

/**
 * 将工具函数挂载到 window 对象（向后兼容）
 */
if (typeof window !== 'undefined') {
  window.utils = window.utils || {};
  window.utils.getCookie = getCookie;
  window.utils.setCookie = setCookie;
}
