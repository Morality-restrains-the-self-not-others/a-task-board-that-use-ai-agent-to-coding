// @vitest-environment jsdom
/**
 * SystemAdminUserKycDrawer：标题区须展示各 KYC 等级说明，便于超管理解 T0/T1/T2。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUserKycDrawer.tier-help.unit.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
  }))
  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))

  const { default: SystemAdminUserKycDrawer } = await import('./SystemAdminUserKycDrawer.vue')
  const { KYC_TIER_HELP } = await import('../utils/kycTierHelp.js')

  describe('SystemAdminUserKycDrawer tier help', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          profile: { tier: 'T1_basic', status: 'approved' },
          audit: [],
          latest_aml: null,
          limit_policies: [
            { tier: 'T0_unverified', max_single_yuan: 100, max_daily_yuan: 300 },
            { tier: 'T1_basic', max_single_yuan: 1000, max_daily_yuan: 3000 },
            { tier: 'T2_enhanced', max_single_yuan: 5000, max_daily_yuan: 15000 },
          ],
        }),
      })
    })

    it('exposes help copy for T0/T1/T2', () => {
      expect(KYC_TIER_HELP.map((r) => r.tier)).toEqual([
        'T0_unverified',
        'T1_basic',
        'T2_enhanced',
      ])
      expect(KYC_TIER_HELP.every((r) => r.label && r.description)).toBe(true)
    })

    it('renders KYC 身份等级 heading with tier help panel', async () => {
      const wrapper = mount(SystemAdminUserKycDrawer, {
        props: { visible: true, userId: '1001' },
      })
      await flushPromises()

      expect(wrapper.text()).toContain('KYC 身份等级')
      const help = wrapper.get('[data-testid="kyc-tier-help"]')
      expect(help.text()).toContain('各等级说明')
      expect(help.text()).toContain('T0 未验证')
      expect(help.text()).toContain('T1 基础')
      expect(help.text()).toContain('T2 增强')
      expect(help.text()).toContain('手机验证')
      expect(help.text()).toContain('人工审核')

      wrapper.unmount()
    })

    it('各档限额金额来自 API limit_policies 而非硬编码', async () => {
      const wrapper = mount(SystemAdminUserKycDrawer, {
        props: { visible: false, userId: '1001' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()

      const help = wrapper.get('[data-testid="kyc-tier-help"]')
      expect(help.text()).toContain('单笔≤100元')
      expect(help.text()).toContain('日累计≤300元')
      expect(help.text()).toContain('单笔≤1,000元')
      expect(help.text()).toContain('单笔≤5,000元')

      wrapper.unmount()
    })
  })
}
