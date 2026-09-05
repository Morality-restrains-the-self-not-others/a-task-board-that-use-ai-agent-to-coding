// @vitest-environment jsdom
/**
 * 注销阻断「前往处理」必须落到真实页面；错误路径会被 router catch-all 踢回首页。
 */
if (!process.env.VITEST) {
  console.log('[skip] UserProfileAccountDeletionPanel.actionUrl.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  const Panel = (await import('./UserProfileAccountDeletionPanel.vue')).default

  function jsonOk(body) {
    return {
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => body,
    }
  }

  describe('UserProfileAccountDeletionPanel 前往处理链接', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/account-deletion/status/')) {
          return jsonOk({ status: 'none' })
        }
        if (u.includes('/account-deletion/precheck/')) {
          return jsonOk({
            cooldown_days: 15,
            blockers: [
              {
                code: 'BILLING_GITLAB_RESOURCE_ACTIVE',
                blocking: true,
                message: '租户 877397588196749312 仍有有效 GitLab 资源订阅',
                action_url: '/tenant/877397588196749312/billing/gitlab-resources/',
                tenant_id: '877397588196749312',
              },
              {
                code: 'BILLING_PAYMENT_PENDING',
                blocking: true,
                message: '有待完成的支付（WX877596018835750912）',
                action_url: '/profile/',
                tenant_id: '877397588196749312',
              },
            ],
          })
        }
        return jsonOk({})
      })
    })

    it('GitLab 订阅与待支付链接指向已注册页面，而非会踢回首页的假路径', async () => {
      const wrapper = mount(Panel)
      await flushPromises()
      const links = wrapper.findAll('[data-testid="account-deletion-blocker-action"]')
      expect(links).toHaveLength(2)
      expect(links[0].attributes('href')).toBe(
        '/tenant/877397588196749312/settings/gitlab-connection/',
      )
      expect(links[1].attributes('href')).toBe('/tenant/877397588196749312/billing/')
      expect(links[0].attributes('href')).not.toContain('/billing/gitlab-resources/')
      expect(links[1].attributes('href')).not.toBe('/profile/')
    })
  })
}
