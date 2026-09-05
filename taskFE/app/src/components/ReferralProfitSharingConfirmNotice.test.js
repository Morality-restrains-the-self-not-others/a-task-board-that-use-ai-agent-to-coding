// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ReferralProfitSharingConfirmNotice.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { flushPromises, mount } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  const { default: ReferralProfitSharingConfirmNotice } = await import('./ReferralProfitSharingConfirmNotice.vue')

  function jsonOk(body) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => body,
    })
  }

  describe('ReferralProfitSharingConfirmNotice', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
    })

    it('tells the user to confirm WeChat profit sharing within the notice validity window', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonOk({ pending_count: 0 }))
      const wrapper = mount(ReferralProfitSharingConfirmNotice)
      await flushPromises()
      const el = wrapper.get('[data-testid="referral-profit-sharing-confirm-notice"]')
      expect(el.text()).toContain('有效期内')
      expect(el.text()).toContain('点击确认')
      expect(el.text()).toContain('无法补分')
      expect(el.text()).not.toContain('待确认')
    })

    it('shows pending count and deadline when WeChat receivers are PENDING（OPT-20260823-043）', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonOk({ pending_count: 2, deadline: '2026-09-01' }))
      const wrapper = mount(ReferralProfitSharingConfirmNotice)
      await flushPromises()
      const el = wrapper.get('[data-testid="referral-profit-sharing-confirm-notice"]')
      expect(el.text()).toContain('待确认 2 笔')
      expect(el.text()).toContain('2026-09-01')
      expect(el.attributes('data-pending-count')).toBe('2')
    })

    it('falls back to static notice when pending fetch returns no deadline', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonOk({ pending_count: 1 }))
      const wrapper = mount(ReferralProfitSharingConfirmNotice)
      await flushPromises()
      const el = wrapper.get('[data-testid="referral-profit-sharing-confirm-notice"]')
      expect(el.text()).toContain('待确认 1 笔')
      expect(el.text()).toContain('无法补分')
      expect(el.text()).not.toContain('最晚')
    })
  })
}
