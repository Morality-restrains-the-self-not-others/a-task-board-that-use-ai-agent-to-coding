// @vitest-environment jsdom
/**
 * 系统管理 · GitLab 资源：区域卡片须展示是否带宽共享及总/剩余带宽。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGitlabResources.bandwidth.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../components/system-admin/SystemAdminGitlabRegionForm.vue', () => ({
    default: { name: 'SystemAdminGitlabRegionForm', template: '<div />' },
  }))

  vi.mock('../components/system-admin/SystemAdminGitlabTenantPanel.vue', () => ({
    default: { name: 'SystemAdminGitlabTenantPanel', template: '<div />' },
  }))

  const Page = (await import('./SystemAdminGitlabResources.vue')).default

  const region = {
    name: '腾讯上海一区',
    slug: 'tencent-sh-1',
    gitlab_web_url: 'https://gitlab-tencent-sh-1.daydaymoney.com',
    is_active: true,
    allocated_disk_gb: 0,
    total_disk_gb: 100,
    allocated_traffic_gb: 0,
    total_traffic_gb: 50,
    bandwidth_shared: true,
    total_bandwidth_mbps: 200,
    remaining_bandwidth_mbps: 80,
  }

  function jsonOk(body) {
    return {
      ok: true,
      status: 200,
      headers: { get: () => 'application/json' },
      json: async () => body,
      clone: function () {
        return this
      },
    }
  }

  describe('SystemAdminGitlabResources 带宽共享展示', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.apiFetchMock.mockImplementation(async () => jsonOk({ regions: [region] }))
    })

    it('卡片展示带宽共享分区、总带宽与剩余带宽', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-share"]').text()).toBe('带宽共享分区')
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-total"]').text()).toContain('200')
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-remaining"]').text()).toBe('80')
    })
  })
}
