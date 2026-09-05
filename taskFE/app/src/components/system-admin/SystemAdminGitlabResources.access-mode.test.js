// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGitlabResources.access-mode.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  vi.mock('../../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'k' }),
    }),
    mergeIdempotencyHeaders: (h) => h,
  }))

  const { default: View } = await import('../../views/SystemAdminGitlabResources.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminGitlabResources 区域模式徽章', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue(jsonOk({
        regions: [{
          slug: 'tencent-sh-1',
          name: '腾讯上海一区',
          gitlab_web_url: 'https://gitlab-tencent-sh-1.daydaymoney.com',
          is_active: true,
          access_mode: 'development',
        }],
        total: 1,
      }))
    })

    it('renders 开发模式 for development regions', async () => {
      const wrapper = mount(View, {
        global: {
          stubs: {
            SystemAdminGitlabRegionForm: true,
            SystemAdminGitlabRegionDeleteModal: true,
            SystemAdminGitlabRegionDeployPaths: true,
            SystemAdminGitlabRegionCapacity: true,
            SystemAdminGitlabTenantPanel: true,
          },
        },
      })
      await flushPromises()
      expect(wrapper.get('[data-testid="gitlab-region-access-mode"]').text()).toBe('开发模式')
    })
  })
}
