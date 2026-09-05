// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] GitlabSelfHostedReachability.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  const { default: GitlabSelfHostedReachability } = await import('./GitlabSelfHostedReachability.vue')

  describe('GitlabSelfHostedReachability', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('does not probe when not configured', async () => {
      const wrapper = mount(GitlabSelfHostedReachability, {
        props: { tenantId: '123', configured: false, intranet: false },
      })
      await flushPromises()
      expect(apiFetchMock).not.toHaveBeenCalled()
      expect(wrapper.find('[data-testid="gitlab-self-hosted-reachability"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="gitlab-intranet-checkbox"]').exists()).toBe(true)
    })

    it('skips probe and does not show down alarm when intranet', async () => {
      const wrapper = mount(GitlabSelfHostedReachability, {
        props: { tenantId: '123', configured: true, intranet: true },
      })
      await flushPromises()
      expect(apiFetchMock).not.toHaveBeenCalled()
      const box = wrapper.find('[data-testid="gitlab-self-hosted-reachability"]')
      expect(box.attributes('data-status')).toBe('skipped_intranet')
      expect(wrapper.find('[data-testid="gitlab-reachability-intranet"]').text()).toContain('内网')
      expect(wrapper.find('[data-testid="gitlab-reachability-down"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="gitlab-reachability-retry"]').exists()).toBe(false)
    })

    it('shows unreachable when public GitLab cannot be reached', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          configured: true,
          intranet: false,
          status: 'unreachable',
          reachable: false,
          reason: 'timeout',
        }),
        traceId: '',
      })
      const wrapper = mount(GitlabSelfHostedReachability, {
        props: { tenantId: '123', configured: true, intranet: false },
      })
      await flushPromises()
      expect(String(apiFetchMock.mock.calls[0][0])).toContain(
        '/api/git-oauth/tenant-connection/tenant_id/123/reachability/',
      )
      const box = wrapper.find('[data-testid="gitlab-self-hosted-reachability"]')
      expect(box.attributes('data-status')).toBe('unreachable')
      expect(wrapper.find('[data-testid="gitlab-reachability-down"]').text()).toContain('无法连接')
    })

    it('emits intranet toggle', async () => {
      const wrapper = mount(GitlabSelfHostedReachability, {
        props: { tenantId: '123', configured: true, intranet: false },
      })
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ status: 'reachable', reachable: true }),
        traceId: '',
      })
      await wrapper.find('[data-testid="gitlab-intranet-checkbox"]').setValue(true)
      expect(wrapper.emitted('update:intranet')?.[0]).toEqual([true])
    })
  })
}
