// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ReferralStatsPanel.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  function jsonOk(body) {
    return Promise.resolve({ ok: true, status: 200, json: async () => body })
  }

  describe('ReferralStatsPanel', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      hoisted.apiFetchMock.mockImplementation(() => jsonOk({
        referral_count: 2,
        referral_rate_display: '12%',
        monthly_earnings: [],
        channels: [{ name: '微信', channel_code: 'WX', referral_count: 2, consumption: '10.00', commission: '0.50' }],
      }))
    })

    it('sends channel and date filters on stats GET', async () => {
      const { default: Panel } = await import('./ReferralStatsPanel.vue')
      const wrapper = mount(Panel, {
        props: {
          userId: 'user-1',
          channels: [{ code: 'WX', name: '微信' }],
        },
      })
      await flushPromises()
      await wrapper.get('[data-testid="referral-stats-channel"]').setValue('WX')
      await wrapper.get('[data-testid="referral-stats-from"]').setValue('2026-07-01')
      await wrapper.get('[data-testid="referral-stats-to"]').setValue('2026-07-31')
      await wrapper.get('[data-testid="referral-stats-apply"]').trigger('click')
      await flushPromises()
      const urls = hoisted.apiFetchMock.mock.calls.map((c) => String(c[0]))
      expect(urls.some((u) => u.includes('channel_code=WX') && u.includes('from=2026-07-01') && u.includes('to=2026-07-31'))).toBe(true)
      expect(wrapper.get('[data-testid="referral-channel-stats"]').text()).toContain('微信')
    })

    it('does not substitute 5% when stats omit referral_rate_display', async () => {
      hoisted.apiFetchMock.mockImplementation(() => jsonOk({
        referral_count: 0,
        referral_rate_display: '',
        monthly_earnings: [],
        channels: [],
      }))
      const { default: Panel } = await import('./ReferralStatsPanel.vue')
      const wrapper = mount(Panel, {
        props: { userId: 'user-1' },
      })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-rate-display"]').text()).not.toBe('5%')
      expect(wrapper.get('[data-testid="referral-rate-display"]').text()).toBe('—')
    })

    it('labels the rate card as 当前的分账比例 and keeps the API value prominent', async () => {
      const { default: Panel } = await import('./ReferralStatsPanel.vue')
      const wrapper = mount(Panel, {
        props: { userId: 'user-1' },
      })
      await flushPromises()
      const label = wrapper.get('[data-testid="referral-rate-label"]')
      expect(label.text()).toBe('当前的分账比例')
      expect(label.classes()).toContain('font-semibold')
      expect(label.classes()).not.toContain('text-text-light')
      expect(wrapper.get('[data-testid="referral-rate-display"]').text()).toBe('12%')
      expect(wrapper.text()).toContain('当前的分账比例为')
      expect(wrapper.text()).toContain('12%')
    })

    it('renders referral_rate_display from stats API instead of a hardcoded 5%', async () => {
      const { default: Panel } = await import('./ReferralStatsPanel.vue')
      const wrapper = mount(Panel, {
        props: { userId: 'user-1' },
      })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-rate-display"]').text()).toBe('12%')
      expect(wrapper.text()).toContain('当前的分账比例为')
      expect(wrapper.text()).toContain('12%')
      expect(wrapper.get('[data-testid="referral-rate-display"]').text()).not.toBe('5%')
      const notices = wrapper.findAll('[data-testid="referral-rate-change-notice"]')
      expect(notices.length).toBeGreaterThanOrEqual(1)
      expect(notices[0].text()).toContain('固定为 5%')
    })

    it('states commission eligibility is checked at order time, not bind time', async () => {
      const { default: Panel } = await import('./ReferralStatsPanel.vue')
      const wrapper = mount(Panel, { props: { userId: 'user-1' } })
      await flushPromises()
      const rule = wrapper.get('[data-testid="referral-earnings-rule"]')
      expect(rule.text()).toContain('下单时')
      expect(rule.text()).toContain('后续下单')
      expect(rule.text()).not.toContain('绑边时')
    })

    it('warns ineligible users that later qualification still shares later orders', async () => {
      const { default: Panel } = await import('./ReferralStatsPanel.vue')
      const wrapper = mount(Panel, {
        props: { userId: 'user-1', hasActiveQualification: false },
      })
      await flushPromises()
      const notice = wrapper.get('[data-testid="referral-stats-ineligible-notice"]')
      expect(notice.text()).toContain('下单时')
      expect(notice.text()).toContain('后续下单')
      expect(notice.text()).not.toContain('但消费不分账')
    })
  })
}
