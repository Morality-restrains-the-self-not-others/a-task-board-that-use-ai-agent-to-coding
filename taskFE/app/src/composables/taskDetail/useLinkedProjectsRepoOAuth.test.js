// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useLinkedProjectsRepoOAuth.test.js requires vitest runtime')
} else {
  const { defineComponent } = await import('vue')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const GITLAB_URL = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad'

  const hoisted = vi.hoisted(() => ({
    fetchConnected: vi.fn(async () => false),
    resolveUserId: vi.fn(async () => 'u1'),
    replace: vi.fn(async () => {}),
    routeMock: {
      path: '/tenant/t1/workspace/w1/task-detail/42/',
      query: {},
      hash: '',
    },
  }))

  vi.mock('../../utils/gitOAuthUserAppConnection.js', () => ({
    fetchGitOAuthUserAppConnected: hoisted.fetchConnected,
  }))
  vi.mock('../../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: hoisted.resolveUserId,
  }))
  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('../../utils/githubAppReturnStorage.js', () => ({
    createGithubAppReturnKey: () => 'a'.repeat(32),
    setGithubAppReturnTarget: () => {},
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
    useRouter: () => ({ replace: hoisted.replace }),
  }))

  const { useLinkedProjectsRepoOAuth } = await import('./useLinkedProjectsRepoOAuth.js')

  const Host = defineComponent({
    props: {
      taskProjectsWithDetails: { type: Array, default: () => [] },
      gitOAuthCatalogVersion: { type: Number, default: 1 },
      getRepoBranchError: { type: Function, default: () => '' },
    },
    setup(props, { emit }) {
      return useLinkedProjectsRepoOAuth({ props, emit })
    },
    template: '<div />',
  })

  const gitlabProjectProps = {
    taskProjectsWithDetails: [
      { project: { git_repos: [GITLAB_URL] } },
    ],
    gitOAuthCatalogVersion: 1,
  }

  describe('useLinkedProjectsRepoOAuth', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoisted.routeMock.query = {}
      hoisted.fetchConnected.mockResolvedValue(false)
      hoisted.resolveUserId.mockResolvedValue('u1')
    })

    it('挂载后查询 user-app-connection；未绑定则 emit unbound', async () => {
      const wrapper = mount(Host, { props: gitlabProjectProps })
      await flushPromises()
      expect(hoisted.fetchConnected).toHaveBeenCalledWith(GITLAB_URL)
      const events = wrapper.emitted('repo-oauth-readiness') || []
      expect(events.length).toBeGreaterThan(0)
      const latest = events[events.length - 1][0]
      expect(latest.loading).toBe(false)
      expect(latest.allBound).toBe(false)
      expect(latest.unboundRepoUrls).toContain(GITLAB_URL)
    })

    it('连接已存在时 emit allBound，不把未检查当成未绑定', async () => {
      hoisted.fetchConnected.mockResolvedValue(true)
      const wrapper = mount(Host, { props: gitlabProjectProps })
      await flushPromises()
      const latest = (wrapper.emitted('repo-oauth-readiness') || []).at(-1)[0]
      expect(latest.loading).toBe(false)
      expect(latest.allBound).toBe(true)
      expect(latest.unboundRepoUrls).toEqual([])
    })

    it('无前端 userId 时仍查询连接（会话由网关 cookie 注入）', async () => {
      hoisted.resolveUserId.mockResolvedValue('')
      hoisted.fetchConnected.mockResolvedValue(true)
      const wrapper = mount(Host, { props: gitlabProjectProps })
      await flushPromises()
      expect(hoisted.fetchConnected).toHaveBeenCalledWith(GITLAB_URL)
      const latest = (wrapper.emitted('repo-oauth-readiness') || []).at(-1)[0]
      expect(latest.allBound).toBe(true)
      wrapper.unmount()
    })

    it('OAuth next 使用当前 task-detail path 与 accessCode', async () => {
      const { buildOauthReturnPath } = await import('../../utils/githubAppContinueNav.js')
      hoisted.routeMock.query = { accessCode: '9aaHjbryhL' }
      expect(
        buildOauthReturnPath({
          routePath: hoisted.routeMock.path,
          search: '',
          query: hoisted.routeMock.query,
        }),
      ).toBe('/tenant/t1/workspace/w1/task-detail/42/?accessCode=9aaHjbryhL')
    })

    it('gitlab=ok 回流后重试查询，一旦 connected 即 allBound 并去掉 query', async () => {
      vi.useFakeTimers()
      hoisted.routeMock.query = { gitlab: 'ok', accessCode: 'DR2AKvP9J9' }
      hoisted.fetchConnected
        .mockResolvedValueOnce(false)
        .mockResolvedValueOnce(false)
        .mockResolvedValue(true)

      const wrapper = mount(Host, { props: gitlabProjectProps })
      await flushPromises()
      await vi.runAllTimersAsync()
      await flushPromises()

      expect(hoisted.fetchConnected.mock.calls.length).toBeGreaterThanOrEqual(2)
      const latest = (wrapper.emitted('repo-oauth-readiness') || []).at(-1)[0]
      expect(latest.allBound).toBe(true)
      expect(hoisted.replace).toHaveBeenCalled()
      const replaced = hoisted.replace.mock.calls[0][0]
      expect(replaced.query.gitlab).toBeUndefined()
      expect(replaced.query.accessCode).toBe('DR2AKvP9J9')
      wrapper.unmount()
      vi.useRealTimers()
    })
  })
}
