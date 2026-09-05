// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminReferralManagement.ratios.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))

  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))

  const { default: Page } = await import('./SystemAdminReferralManagement.vue')

  function okJson(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => null },
    }
  }

  describe('SystemAdminReferralManagement ratios', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('/referral/policy/')) {
          return okJson({ data: { mode: 'approval', message: '' } })
        }
        if (String(url).includes('/referral/config/')) {
          return okJson({
            data: {
              settle_delay_days: 15,
              profit_sharing_ratio_percent: 12,
              referral_rate_percent: 12,
            },
            ratio_percent_min: 5,
            ratio_percent_max: 30,
          })
        }
        return okJson({})
      })
    })

    it('shows a fixed 5% ratio and does not bind config', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      const card = wrapper.get('[data-testid="system-admin-referral-ratio-card"]')
      expect(card.get('[data-testid="referral-rate-fixed"]').text()).toBe('5%')
      expect(wrapper.find('[data-testid="referral-rate-input"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="referral-profit-sharing-ratio-input"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="referral-ratio-save"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="referral-ratio-range"]').exists()).toBe(false)
      expect(card.text()).toContain('分成比例')
      expect(card.text()).toContain('固定为 5%')
      expect(card.text()).not.toContain('5%~30%')
    })

    it('does not POST referral_rate_percent', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      const updateCalls = apiFetch.mock.calls.filter((c) => String(c[0]).includes('/referral/config/update/'))
      expect(updateCalls).toHaveLength(0)
      expect(wrapper.find('[data-testid="referral-ratio-save"]').exists()).toBe(false)
    })
  })
}
