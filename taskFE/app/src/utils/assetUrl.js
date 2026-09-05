/**
 * 静态资源 URL 缓存击穿：为裸 URL 附加内容 hash 查询参数（元规则 18）。
 * Vite 模块图内 import 由 vite-plugin-asset-cache-bust 自动加 ?h=；本模块供模板拼接与动态路径使用。
 */

export const ASSET_HASH_QUERY_PARAM = 'h'

/**
 * @param {string} url
 * @param {string} contentHash
 */
export function appendAssetHashQuery(url, contentHash) {
  const base = String(url || '').trim()
  const hash = String(contentHash || '').trim()
  if (!base || !hash) return base
  if (new RegExp(`[?&]${ASSET_HASH_QUERY_PARAM}=`).test(base)) return base
  const sep = base.includes('?') ? '&' : '?'
  return `${base}${sep}${ASSET_HASH_QUERY_PARAM}=${hash}`
}

/**
 * @param {string} path 站点根相对路径，如 /utils/foo.js
 * @param {string} contentHash
 */
export function assetUrl(path, contentHash) {
  const normalized = String(path || '').trim()
  if (!normalized) return ''
  const withSlash = normalized.startsWith('/') ? normalized : `/${normalized}`
  return appendAssetHashQuery(withSlash, contentHash)
}

