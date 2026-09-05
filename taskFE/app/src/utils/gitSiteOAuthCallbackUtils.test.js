// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] gitSiteOAuthCallbackUtils.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const {
    applyOAuthCallbackFromRoute,
    resolveOAuthCallbackMessage,
    setupOAuthCallbackToastGuard,
    shouldSkipOAuthCallbackToast,
  } = await import('./gitSiteOAuthCallbackUtils.js')

  describe('resolveOAuthCallbackMessage', () => {
    it('gitlab profile_failed 应返回可读错误文案', () => {
      const result = resolveOAuthCallbackMessage('gitlab', 'profile_failed')
      expect(result.severity).toBe('error')
      expect(result.message).toContain('无法读取 GitLab 用户资料')
    })

    it('github ok 应为 success', () => {
      const result = resolveOAuthCallbackMessage('github', 'ok')
      expect(result.severity).toBe('success')
      expect(result.message).toContain('GitHub 授权成功')
    })
  })

  describe('shouldSkipOAuthCallbackToast', () => {
    it('git-site-oauth 设置页应跳过 Toast', () => {
      expect(shouldSkipOAuthCallbackToast('/user/1/profile/git-site-oauth/')).toBe(true)
      expect(shouldSkipOAuthCallbackToast('/tenant/t1/profile/git-site-oauth/')).toBe(true)
    })

    it('项目详情页不应跳过', () => {
      expect(
        shouldSkipOAuthCallbackToast('/tenant/827923618468040704/projects/846027310833254400/'),
      ).toBe(false)
    })
  })

  describe('applyOAuthCallbackFromRoute', () => {
    it('失败码应触发 onError 并清除 gitlab query', async () => {
      const replaceMock = vi.fn().mockResolvedValue(undefined)
      const onError = vi.fn()
      const route = {
        path: '/tenant/1/projects/2/',
        query: { gitlab: 'profile_failed', foo: 'bar', trace_id: 'oauth-tid-1' },
      }
      const router = { replace: replaceMock }

      const result = await applyOAuthCallbackFromRoute(route, router, { onError })

      expect(onError).toHaveBeenCalledWith('授权失败：无法读取 GitLab 用户资料', {
        traceId: 'oauth-tid-1',
      })
      expect(result?.code).toBe('profile_failed')
      expect(result?.traceId).toBe('oauth-tid-1')
      expect(replaceMock).toHaveBeenCalledWith({
        path: '/tenant/1/projects/2/',
        query: { foo: 'bar' },
      })
    })

    it('清除 gitlab 回调参数时保留 accessCode', async () => {
      const replaceMock = vi.fn().mockResolvedValue(undefined)
      const route = {
        path: '/tenant/877397588196749312/projects/proj_880498883115905024/',
        query: { gitlab: 'ok', accessCode: '9aaHjbryhL' },
      }
      const router = { replace: replaceMock }

      await applyOAuthCallbackFromRoute(route, router)

      expect(replaceMock).toHaveBeenCalledWith({
        path: '/tenant/877397588196749312/projects/proj_880498883115905024/',
        query: { accessCode: '9aaHjbryhL' },
      })
    })

    it('ok 不应触发 onError', async () => {
      const onError = vi.fn()
      const route = {
        path: '/tenant/1/projects/2/',
        query: { gitlab: 'ok' },
      }
      const router = { replace: vi.fn().mockResolvedValue(undefined) }

      await applyOAuthCallbackFromRoute(route, router, { onError })

      expect(onError).not.toHaveBeenCalled()
    })
  })

  describe('setupOAuthCallbackToastGuard', () => {
    it('项目详情带 gitlab=profile_failed 应调用 toastService.error', async () => {
      let afterEachHook
      const router = {
        afterEach: vi.fn((fn) => {
          afterEachHook = fn
        }),
        replace: vi.fn().mockResolvedValue(undefined),
      }
      const toastService = { error: vi.fn(), success: vi.fn() }

      setupOAuthCallbackToastGuard(router, toastService)
      expect(router.afterEach).toHaveBeenCalledOnce()

      afterEachHook({
        path: '/tenant/827923618468040704/projects/846027310833254400/',
        query: { gitlab: 'profile_failed', trace_id: 'toast-tid-9' },
      })

      await vi.waitFor(() => {
        expect(toastService.error).toHaveBeenCalledWith(
          '授权失败：无法读取 GitLab 用户资料',
          5000,
          { traceId: 'toast-tid-9' },
        )
      })
    })

    it('git-site-oauth 设置页不应调用 toastService.error', () => {
      let afterEachHook
      const router = {
        afterEach: vi.fn((fn) => {
          afterEachHook = fn
        }),
        replace: vi.fn().mockResolvedValue(undefined),
      }
      const toastService = { error: vi.fn(), success: vi.fn() }

      setupOAuthCallbackToastGuard(router, toastService)

      afterEachHook({
        path: '/user/1/profile/git-site-oauth/',
        query: { gitlab: 'profile_failed' },
      })

      expect(toastService.error).not.toHaveBeenCalled()
      expect(router.replace).not.toHaveBeenCalled()
    })
  })
}
