// 全局配置文件
// 默认 `API_BASE_URL` 为空：请求走当前页同源（生产经 Nginx 反代到 task2app；开发经 Vite proxy 到 Django，与 monorepo conf/（YAML） 中 django 端口一致）。
// 构建时可在 `taskFE/app` 下通过 `.env` 或环境变量设置 `VITE_API_BASE_URL` 覆盖；留空或勿设置即同源。
// `TASK_GATEWAY_PUBLIC_BASE` 必须是 OIDC issuer / authorize 公网 origin（通常为 api.*），
// 不可与空的同源 API_BASE_URL 混用。

function normalizeApiBaseUrl(value) {
  if (value == null || typeof value !== 'string') return ''
  return value.trim().replace(/\/+$/, '')
}

const _viteApi = normalizeApiBaseUrl(import.meta.env.VITE_API_BASE_URL)
const _viteGateway = normalizeApiBaseUrl(import.meta.env.VITE_TASK_GATEWAY_PUBLIC_BASE)
const _viteSsoCookieDomain = normalizeApiBaseUrl(import.meta.env.VITE_SSO_COOKIE_DOMAIN)

export const config = {
  API_BASE_URL: _viteApi,
  TASK_GATEWAY_PUBLIC_BASE: _viteGateway,
  SSO_COOKIE_DOMAIN: _viteSsoCookieDomain,

  // 其他配置项
  APP_NAME: 'SaaS 项目管理系统',
  DEBUG: true
}

/**
 * 可选：在加载 main.js 之前注入 `window.__TASK2APP_API_BASE_URL__`（字符串），用于在不重新构建的情况下覆盖 API 根路径。
 * 同源反代到 task2app 时通常设为 `''`。
 */
export function applyRuntimeApiBaseFromWindow() {
  if (typeof window === 'undefined') return
  if (!Object.prototype.hasOwnProperty.call(window, '__TASK2APP_API_BASE_URL__')) return
  const v = window.__TASK2APP_API_BASE_URL__
  config.API_BASE_URL = normalizeApiBaseUrl(typeof v === 'string' ? v : '')
}

/**
 * 可选：注入 `window.__TASK2APP_TASK_GATEWAY_PUBLIC_BASE__` 覆盖 OIDC gateway origin。
 */
export function applyRuntimeGatewayBaseFromWindow() {
  if (typeof window === 'undefined') return
  if (!Object.prototype.hasOwnProperty.call(window, '__TASK2APP_TASK_GATEWAY_PUBLIC_BASE__')) return
  const v = window.__TASK2APP_TASK_GATEWAY_PUBLIC_BASE__
  config.TASK_GATEWAY_PUBLIC_BASE = normalizeApiBaseUrl(typeof v === 'string' ? v : '')
}

// 导出一个便捷的函数来构建完整的 API URL
export function getApiUrl(endpoint) {
  // 如果 endpoint 已经是完整的 URL，则直接返回
  if (endpoint.startsWith('http://') || endpoint.startsWith('https://')) {
    return endpoint;
  }
  
  // 确保 endpoint 以 '/' 开头
  if (!endpoint.startsWith('/')) {
    endpoint = '/' + endpoint;
  }
  
  // 拼接基础 URL 和 endpoint
  return config.API_BASE_URL + endpoint;
}
