// @vitest-environment jsdom
/**
 * 用户充值抽屉：points / amount_points 为分，须按元展示。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUserRechargeDrawer.formatYuan.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  const Page = (await import('./SystemAdminUserRechargeDrawer.vue')).default

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

  describe('SystemAdminUserRechargeDrawer points 分转元', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
    })

    it('将 amount_points=55 显示为 0.55', async () => {
      hoisted.apiFetch.mockResolvedValue(
        jsonOk({
          recharges: [
            {
              id: 'r1',
              amount_yuan: '1.00',
              amount_points: 55,
              points_source_type: 'user_recharge_wechat',
              created_at: '2026-08-19T08:00:00Z',
            },
          ],
          consents: [],
        })
      )

      const wrapper = mount(Page, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      expect(wrapper.text()).toContain('金额 0.55')
    })
  })
}
