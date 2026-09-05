// @vitest-environment jsdom
/**
 * 系统管理 · GitLab 资源：删除按钮须弹窗展示仓库地址后再确认停用。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGitlabResources.delete-confirm.test.js requires vitest runtime')
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
    gitlab_web_url: 'https://git.daydaymoney.com',
    gitlab_api_base: 'https://git.daydaymoney.com/api/v4',
    is_active: true,
    allocated_disk_gb: 0,
    total_disk_gb: 100,
    allocated_traffic_gb: 0,
    total_traffic_gb: 50,
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

  describe('SystemAdminGitlabResources 删除确认仓库地址', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.apiFetchMock.mockImplementation(async (url, opts = {}) => {
        const u = String(url || '')
        if (u.includes('/gitlab-regions/') && (!opts.method || opts.method === 'GET')) {
          return jsonOk({ regions: [region] })
        }
        if (u.includes('/gitlab-regions/') && opts.method === 'DELETE') {
          return jsonOk({ ok: true })
        }
        return jsonOk({})
      })
    })

    it('点击删除后弹窗展示仓库地址；确认才发 DELETE', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      const deleteBtn = wrapper.findAll('button').find((b) => b.text() === '删除')
      expect(deleteBtn).toBeTruthy()
      await deleteBtn.trigger('click')
      await flushPromises()

      const modal = wrapper.find('[data-testid="gitlab-region-delete-modal"]')
      expect(modal.exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-region-delete-repo-address"]').text()).toBe(
        'https://git.daydaymoney.com'
      )

      const deleteCallsBefore = hoistedMocks.apiFetchMock.mock.calls.filter(
        ([, opts]) => opts?.method === 'DELETE'
      )
      expect(deleteCallsBefore).toHaveLength(0)

      await wrapper.find('[data-testid="gitlab-region-delete-confirm"]').trigger('click')
      await flushPromises()

      const deleteCalls = hoistedMocks.apiFetchMock.mock.calls.filter(
        ([, opts]) => opts?.method === 'DELETE'
      )
      expect(deleteCalls).toHaveLength(1)
      expect(String(deleteCalls[0][0])).toContain('/gitlab-regions/tencent-sh-1/')
    })

    it('取消关闭弹窗且不发 DELETE', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      await wrapper.findAll('button').find((b) => b.text() === '删除').trigger('click')
      await flushPromises()
      await wrapper.find('[data-testid="gitlab-region-delete-cancel"]').trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="gitlab-region-delete-modal"]').exists()).toBe(false)
      const deleteCalls = hoistedMocks.apiFetchMock.mock.calls.filter(
        ([, opts]) => opts?.method === 'DELETE'
      )
      expect(deleteCalls).toHaveLength(0)
    })
  })
}
