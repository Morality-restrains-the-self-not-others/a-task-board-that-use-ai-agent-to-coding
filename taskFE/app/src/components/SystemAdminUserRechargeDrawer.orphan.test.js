// @vitest-environment jsdom
/**
 * 「支付与签署」抽屉：未挂到支付的协议签署须用运营可读文案，禁止「未绑定流水」行话。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUserRechargeDrawer.orphan.test.js requires vitest runtime')
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

  describe('SystemAdminUserRechargeDrawer 其他协议签署', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
    })

    it('注册服务协议未挂支付时展示「其他协议签署」而非「未绑定流水的签署」', async () => {
      hoisted.apiFetch.mockResolvedValue(
        jsonOk({
          recharges: [
            {
              id: 'r1',
              amount_yuan: '1.00',
              amount_points: 100,
              points_source_type: 'user_recharge_wechat',
              channel: 'wechat',
              created_at: '2026-08-19T08:00:00Z',
              consent: { id: 'c-pay', agreement_version: '0.9', document_kind: 'recharge_cents' },
            },
          ],
          consents: [
            {
              id: 'c-pay',
              agreement_version: '0.9',
              document_kind: 'recharge_cents',
              consented_at: '2026-08-19T07:00:00Z',
              content_snapshot: '支付条款',
            },
            {
              id: 'c-signup',
              agreement_version: '1.0',
              document_kind: 'service',
              consented_at: '2026-08-01T08:00:00Z',
              content_snapshot: '服务协议正文',
            },
          ],
        })
      )

      const wrapper = mount(Page, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()

      const heading = wrapper.get('[data-testid="orphan-consent-heading"]')
      expect(heading.text()).toBe('其他协议签署')
      expect(wrapper.text()).not.toContain('未绑定流水的签署')
      expect(wrapper.text()).toContain('不表示签署缺失')
      expect(wrapper.text()).toContain('服务协议')
      expect(wrapper.text()).toContain('v1.0')
    })

    it('全部签署已挂在支付上时不渲染其他协议区块', async () => {
      hoisted.apiFetch.mockResolvedValue(
        jsonOk({
          recharges: [
            {
              id: 'r1',
              amount_points: 100,
              points_source_type: 'user_recharge_wechat',
              consent: { id: 'c-pay' },
            },
          ],
          consents: [{ id: 'c-pay', document_kind: 'recharge_cents', agreement_version: '0.9' }],
        })
      )

      const wrapper = mount(Page, {
        props: { visible: false, userId: 'u2' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()

      expect(wrapper.find('[data-testid="orphan-consent-heading"]').exists()).toBe(false)
    })
  })
}
