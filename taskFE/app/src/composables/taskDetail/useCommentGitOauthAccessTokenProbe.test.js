// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useCommentGitOauthAccessTokenProbe.test.js requires vitest runtime')
} else {
  const { defineComponent, computed } = await import('vue')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const githubUrl = 'https://github.com/acme/demo.git'

  const hoisted = vi.hoisted(() => ({
    probeOnce: vi.fn(),
    showRequestError: vi.fn(),
  }))

  vi.mock('../../utils/probeCommentGitOauthAccessTokensOnce.js', () => ({
    probeCommentGitOauthAccessTokensOnce: (...args) => hoisted.probeOnce(...args),
  }))
  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: (...args) => hoisted.showRequestError(...args),
  }))

  const { useCommentGitOauthAccessTokenProbe } = await import('./useCommentGitOauthAccessTokenProbe.js')

  const Host = defineComponent({
    props: {
      displayComments: { type: Array, default: () => [] },
      fallbackRepoIdentities: { type: Array, default: () => [] },
      repoOAuthReadiness: { type: Object, default: null },
    },
    setup(props, { emit }) {
      return useCommentGitOauthAccessTokenProbe({
        displayComments: computed(() => props.displayComments),
        fallbackRepoIdentities: computed(() => props.fallbackRepoIdentities),
        repoOAuthReadiness: computed(() => props.repoOAuthReadiness),
        emit,
      })
    },
    template: '<div />',
  })

  describe('useCommentGitOauthAccessTokenProbe', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoisted.probeOnce.mockResolvedValue({
        probed: [],
        invalidRepoUrls: [],
        unreachableRepoUrls: [],
        checkFailedRepoUrls: [],
      })
    })

    it('does not open an error modal when AccessToken probe times out; surfaces check_failed on readiness', async () => {
      hoisted.probeOnce.mockResolvedValue({
        probed: [githubUrl],
        invalidRepoUrls: [],
        unreachableRepoUrls: [],
        checkFailedRepoUrls: [githubUrl],
        probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
        probeTraceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
      })
      const wrapper = mount(Host, {
        props: {
          displayComments: [{ repo_identities: [{ repo_url: githubUrl }] }],
          repoOAuthReadiness: {
            loading: false,
            hasOAuthRepos: true,
            allBound: true,
            unboundRepoUrls: [],
          },
        },
      })
      await flushPromises()
      expect(hoisted.showRequestError).not.toHaveBeenCalled()
      const events = wrapper.emitted('repo-oauth-readiness') || []
      expect(events.length).toBeGreaterThan(0)
      const latest = events[events.length - 1][0]
      expect(latest.checkFailedRepoUrls).toEqual([githubUrl])
      expect(latest.probeError).toBe('检查 OAuth 绑定状态超时，请确认服务可用后重试')
      expect(latest.probeTraceId).toBe('e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb')
      expect(latest.startBlocked).toBe(true)
    })

    it('retry re-probes a check_failed URL on demand without opening an error modal (OPT-20260902-011)', async () => {
      hoisted.probeOnce.mockResolvedValue({
        probed: [githubUrl],
        invalidRepoUrls: [],
        unreachableRepoUrls: [],
        checkFailedRepoUrls: [githubUrl],
        probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
        probeTraceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
      })
      const wrapper = mount(Host, {
        props: {
          displayComments: [{ repo_identities: [{ repo_url: githubUrl }] }],
          repoOAuthReadiness: {
            loading: false,
            hasOAuthRepos: true,
            allBound: true,
            unboundRepoUrls: [],
          },
        },
      })
      await flushPromises()
      expect(hoisted.probeOnce).toHaveBeenCalledTimes(1)
      // 重试后仍失败：再次探测、更新徽标，但不得弹错误窗
      hoisted.probeOnce.mockResolvedValue({
        probed: [githubUrl],
        invalidRepoUrls: [],
        unreachableRepoUrls: [],
        checkFailedRepoUrls: [githubUrl],
        probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
        probeTraceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
      })
      expect(typeof wrapper.vm.retryProbeFor).toBe('function')
      wrapper.vm.retryProbeFor([githubUrl])
      await flushPromises()
      expect(hoisted.probeOnce).toHaveBeenCalledTimes(2)
      expect(hoisted.probeOnce.mock.calls[1][0]).toMatchObject({
        force: true,
        urls: [githubUrl],
      })
      expect(hoisted.showRequestError).not.toHaveBeenCalled()
      const events = wrapper.emitted('repo-oauth-readiness') || []
      const last = events[events.length - 1][0]
      expect(last.checkFailedRepoUrls).toEqual([githubUrl])
      expect(last.startBlocked).toBe(true)
    })
  })
}
