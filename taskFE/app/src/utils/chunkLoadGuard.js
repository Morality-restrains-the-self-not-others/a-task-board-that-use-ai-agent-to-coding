// 懒加载 chunk 获取失败兜底（OPT-20260807-056）
//
// 背景：线上报 "Failed to fetch dynamically imported module"（发布后旧 index 引用
// 已删除的旧哈希 chunk / CDN 缓存不一致 / 瞬时网络失败）。全站 ~40+ 路由懒加载
// （router.js），此前无任何 chunk 失败兜底。
//
// 策略：
// - 路由导航失败 → 重试 1 次（force 重导航会重跑 lazy import）→ 仍失败则整页
//   刷新 1 次（拿到新 index.html → 新 chunk 映射，收敛）。
// - 点击路径（如提交评论的 import()）失败 → 展示刷新文案、默认不 reload，避免
//   丢掉输入框草稿；瞬时网络失败可用 importWithChunkGuard 重试 1 次。
// 每页面生命周期内 1 次导航重试 + 1 次 reload 封顶，不会循环。

// 跨浏览器错误签名（大小写不敏感）：
// - Chrome/Edge: "Failed to fetch dynamically imported module"
// - Firefox:     "error loading dynamically imported module"
// - Safari:      "Importing a module script failed"
// - webpack:     "Loading chunk N failed"
// - Vite preload: "Unable to preload CSS"
const CHUNK_LOAD_ERROR_RE =
  /Failed to fetch dynamically imported module|error loading dynamically imported module|Importing a module script failed|Loading chunk .* failed|Unable to preload CSS/i

export const CHUNK_STALE_RELOAD_MESSAGE = '页面已更新，请刷新后重试'

const chunkLoadRecoveryState = { pageReloaded: false }

export function resetChunkLoadGuardForTests() {
  chunkLoadRecoveryState.pageReloaded = false
}

export const isChunkLoadError = (error) => {
  if (!error) return false
  const message = typeof error === 'string' ? error : error.message || String(error)
  return CHUNK_LOAD_ERROR_RE.test(message)
}

function defaultReload() {
  if (typeof window !== 'undefined' && window.location && typeof window.location.reload === 'function') {
    window.location.reload()
  }
}

/**
 * 路由级兜底：确认是 chunk 失败后整页刷新 1 次。
 * @param {unknown} error
 * @param {{ reload?: () => void }} [options]
 * @returns {boolean} 是否已按 chunk 失败处理
 */
export function recoverFromChunkLoadError(error, options = {}) {
  if (!isChunkLoadError(error)) return false
  const reloadPage = options.reload || defaultReload
  if (chunkLoadRecoveryState.pageReloaded) return true
  chunkLoadRecoveryState.pageReloaded = true
  reloadPage()
  return true
}

/**
 * 点击/提交路径：chunk 失败时提示用户刷新，默认不 reload（保留表单）。
 * @param {unknown} error
 * @param {{ showError?: (message: string) => void }} [options]
 * @returns {boolean}
 */
export function handleActionChunkLoadError(error, options = {}) {
  if (!isChunkLoadError(error)) return false
  if (typeof options.showError === 'function') {
    options.showError(CHUNK_STALE_RELOAD_MESSAGE)
  }
  return true
}

/**
 * 包装动态 import：瞬时失败重试 1 次；仍失败则 reload 并抛出。
 * @template T
 * @param {() => Promise<T>} importer
 * @param {{ reload?: () => void }} [options]
 * @returns {Promise<T>}
 */
export async function importWithChunkGuard(importer, options = {}) {
  try {
    return await importer()
  } catch (error) {
    if (!isChunkLoadError(error)) throw error
    try {
      return await importer()
    } catch (retryError) {
      recoverFromChunkLoadError(retryError, options)
      throw retryError
    }
  }
}

/**
 * 安装路由级 chunk 失败兜底。依赖注入 options 便于单测：
 * @param {import('vue-router').Router} router
 * @param {{ reload?: () => void }} [options]
 */
export function installRouterChunkGuard(router, options = {}) {
  if (!router || typeof router.onError !== 'function') return

  let navigationRetried = false

  router.onError((error, to) => {
    if (!isChunkLoadError(error)) return

    if (!navigationRetried) {
      // 首次：force 重导航重跑 lazy import（瞬时网络失败即可恢复）
      navigationRetried = true
      router
        .replace({ path: to.fullPath, force: true })
        // 重试仍失败（chunk 仍不可达）→ 整页刷新拿新 index.html
        .catch(() => recoverFromChunkLoadError(error, options))
      return
    }
    recoverFromChunkLoadError(error, options)
  })
}
