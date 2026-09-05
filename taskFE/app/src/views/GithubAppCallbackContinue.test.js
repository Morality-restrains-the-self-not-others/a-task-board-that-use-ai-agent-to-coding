// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] GithubAppCallbackContinue.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    replace: vi.fn(),
    consume: vi.fn(),
    routeQuery: {
      returnKey: 'ab'.repeat(16),
      provider: 'gitlab',
      gitlab: 'ok',
    },
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({ query: hoisted.routeQuery }),
    useRouter: () => ({ replace: hoisted.replace }),
  }))

  vi.mock('../utils/githubAppReturnStorage', () => ({
    consumeGithubAppReturnTarget: (...args) => hoisted.consume(...args),
  }))

  const { default: GithubAppCallbackContinue } = await import('./GithubAppCallbackContinue.vue')

  describe('GithubAppCallbackContinue', () => {
    beforeEach(() => {
      hoisted.replace.mockReset()
      hoisted.consume.mockReset()
      hoisted.routeQuery.returnKey = 'ab'.repeat(16)
      hoisted.routeQuery.provider = 'gitlab'
      hoisted.routeQuery.gitlab = 'ok'
    })

    it('consumes returnKey and replace 回项目详情（保留 accessCode）', async () => {
      hoisted.consume.mockReturnValue(
        '/tenant/877397588196749312/projects/proj_880498883115905024/?accessCode=9aaHjbryhL',
      )
      mount(GithubAppCallbackContinue)
      await flushPromises()
      expect(hoisted.consume).toHaveBeenCalledWith('ab'.repeat(16))
      expect(hoisted.replace).toHaveBeenCalledWith({
        path: '/tenant/877397588196749312/projects/proj_880498883115905024/',
        query: { accessCode: '9aaHjbryhL', gitlab: 'ok' },
      })
    })

    it('returnKey 缺失时落到 /profile/', async () => {
      hoisted.routeQuery.returnKey = ''
      mount(GithubAppCallbackContinue)
      await flushPromises()
      expect(hoisted.replace).toHaveBeenCalledWith({ path: '/profile/' })
    })

    it('localStorage 无目标时落到 git-site-oauth', async () => {
      hoisted.consume.mockReturnValue(null)
      mount(GithubAppCallbackContinue)
      await flushPromises()
      expect(hoisted.replace).toHaveBeenCalledWith({
        path: '/profile/git-site-oauth/',
        query: { provider: 'gitlab', gitlab: 'return_expired' },
      })
    })
  })
}
