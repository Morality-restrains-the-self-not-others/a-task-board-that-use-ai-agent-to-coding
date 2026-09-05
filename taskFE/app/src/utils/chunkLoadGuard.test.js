import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  CHUNK_STALE_RELOAD_MESSAGE,
  handleActionChunkLoadError,
  importWithChunkGuard,
  isChunkLoadError,
  installRouterChunkGuard,
  recoverFromChunkLoadError,
  resetChunkLoadGuardForTests,
} from './chunkLoadGuard.js'

const createFakeRouter = () => {
  let errorHandler = null
  const router = {
    replace: vi.fn(),
    onError(handler) {
      errorHandler = handler
    },
    /** 测试助手：触发一次导航错误 */
    fireError(error, to = { fullPath: '/profile/' }) {
      errorHandler(error, to)
    },
  }
  return router
}

describe('isChunkLoadError', () => {
  it('命中 Chrome 签名', () => {
    expect(
      isChunkLoadError(
        new TypeError('Failed to fetch dynamically imported module: https://x/assets/UserProfile-abc.js'),
      ),
    ).toBe(true)
  })

  it('命中 Firefox / Safari / webpack / Vite preload 签名', () => {
    expect(isChunkLoadError(new TypeError('error loading dynamically imported module: https://x/a.js'))).toBe(true)
    expect(isChunkLoadError(new TypeError('Importing a module script failed: https://x/a.js'))).toBe(true)
    expect(isChunkLoadError(new Error('Loading chunk 12 failed.'))).toBe(true)
    expect(isChunkLoadError(new Error('Unable to preload CSS: https://x/a.css'))).toBe(true)
  })

  it('普通错误 / 空值不命中', () => {
    expect(isChunkLoadError(new TypeError('x is not a function'))).toBe(false)
    expect(isChunkLoadError(new Error('Network Error'))).toBe(false)
    expect(isChunkLoadError(null)).toBe(false)
    expect(isChunkLoadError(undefined)).toBe(false)
  })
})

describe('handleActionChunkLoadError', () => {
  beforeEach(() => {
    resetChunkLoadGuardForTests()
  })

  it('命中点击路径 chunk 失败 → 展示刷新文案且默认不 reload（保留表单）', () => {
    const showError = vi.fn()
    const reload = vi.fn()
    const error = new TypeError(
      'Failed to fetch dynamically imported module: https://www.daydaymoney.com/static/assets/gitOauthPushPrecheck-C15U_Qut.js',
    )
    expect(handleActionChunkLoadError(error, { showError, reload })).toBe(true)
    expect(showError).toHaveBeenCalledWith(CHUNK_STALE_RELOAD_MESSAGE)
    expect(reload).not.toHaveBeenCalled()
  })

  it('非 chunk 错误不处理', () => {
    const showError = vi.fn()
    expect(handleActionChunkLoadError(new Error('提交失败'), { showError })).toBe(false)
    expect(showError).not.toHaveBeenCalled()
  })
})

describe('recoverFromChunkLoadError', () => {
  beforeEach(() => {
    resetChunkLoadGuardForTests()
  })

  it('chunk 错误且允许 reload 时只刷新 1 次', () => {
    const reload = vi.fn()
    const error = new TypeError('Failed to fetch dynamically imported module: https://x/a.js')
    expect(recoverFromChunkLoadError(error, { reload })).toBe(true)
    expect(recoverFromChunkLoadError(error, { reload })).toBe(true)
    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('普通错误不刷新', () => {
    const reload = vi.fn()
    expect(recoverFromChunkLoadError(new Error('boom'), { reload })).toBe(false)
    expect(reload).not.toHaveBeenCalled()
  })
})

describe('importWithChunkGuard', () => {
  beforeEach(() => {
    resetChunkLoadGuardForTests()
  })

  it('首次失败后重试成功则返回模块', async () => {
    const reload = vi.fn()
    const importer = vi.fn()
      .mockRejectedValueOnce(new TypeError('Failed to fetch dynamically imported module: https://x/a.js'))
      .mockResolvedValueOnce({ ok: true })
    await expect(importWithChunkGuard(importer, { reload })).resolves.toEqual({ ok: true })
    expect(importer).toHaveBeenCalledTimes(2)
    expect(reload).not.toHaveBeenCalled()
  })

  it('两次都失败则 reload 并抛出', async () => {
    const reload = vi.fn()
    const error = new TypeError('Failed to fetch dynamically imported module: https://x/a.js')
    const importer = vi.fn().mockRejectedValue(error)
    await expect(importWithChunkGuard(importer, { reload })).rejects.toBe(error)
    expect(importer).toHaveBeenCalledTimes(2)
    expect(reload).toHaveBeenCalledTimes(1)
  })
})

describe('installRouterChunkGuard', () => {
  beforeEach(() => {
    resetChunkLoadGuardForTests()
  })

  it('首次 chunk 错误 → force 重导航重试 1 次', () => {
    const router = createFakeRouter()
    const reload = vi.fn()
    router.replace.mockResolvedValue(undefined)
    installRouterChunkGuard(router, { reload })

    router.fireError(new TypeError('Failed to fetch dynamically imported module: https://x/UserProfile-abc.js'))

    expect(router.replace).toHaveBeenCalledTimes(1)
    expect(router.replace).toHaveBeenCalledWith({ path: '/profile/', force: true })
    expect(reload).not.toHaveBeenCalled()
  })

  it('重试仍失败 → reload 兜底 1 次', async () => {
    const router = createFakeRouter()
    const reload = vi.fn()
    router.replace.mockRejectedValue(new TypeError('Failed to fetch dynamically imported module: https://x/a.js'))
    installRouterChunkGuard(router, { reload })

    router.fireError(new TypeError('Failed to fetch dynamically imported module: https://x/a.js'))
    await Promise.resolve() // 等待 catch 链

    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('重试后再次 chunk 错误 → reload 封顶 1 次（防循环）', async () => {
    const router = createFakeRouter()
    const reload = vi.fn()
    router.replace.mockResolvedValue(undefined)
    installRouterChunkGuard(router, { reload })

    // 第一次：重试成功，无 reload
    router.fireError(new TypeError('Failed to fetch dynamically imported module: https://x/a.js'))
    await Promise.resolve()
    expect(reload).not.toHaveBeenCalled()

    // 第二次 + 第三次：直接 reload，且仅 1 次
    router.fireError(new TypeError('Failed to fetch dynamically imported module: https://x/b.js'))
    router.fireError(new TypeError('Failed to fetch dynamically imported module: https://x/c.js'))

    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('非 chunk 错误不触发重试 / reload', () => {
    const router = createFakeRouter()
    const reload = vi.fn()
    installRouterChunkGuard(router, { reload })

    router.fireError(new TypeError('x is not a function'))

    expect(router.replace).not.toHaveBeenCalled()
    expect(reload).not.toHaveBeenCalled()
  })
})
