// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] SystemAdminReferralPerformanceDrawer.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(async () => ({
      ok: true,
      json: async () => ({
        referral_count: 1,
        own_recharge_total: 100,
        referred_recharge_total: 50,
        own_recharge_count: 1,
        referred_users: [],
        commission: {
          pending_points: 10,
          settled_points: 40,
          settle_delay_days: 7,
          commission_rate_display: '12%',
          items: [
            {
              id: 'itm_1',
              referred_user_id: 'u_123456789012345678',
              resource_type: 'task_post',
              consumption_points: 50,
              commission_points: 5,
              status: 'settled',
              consumed_at: '2026-08-10T09:30:00Z',
            },
          ],
        },
      }),
      headers: { get: () => null },
    })),
  }))

  const { default: ReferralDrawer } = await import('./SystemAdminReferralPerformanceDrawer.vue')

  describe('SystemAdminReferralPerformanceDrawer', () => {
    it('renders 佣金明细 消费时间 via formatDate without TypeError', async () => {
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      // 消费时间列应渲染为可读日期，而非调用未定义 formatDate 抛错导致空白
      const cells = wrapper.findAll('tbody tr td')
      const timeCell = cells.find((c) => c.text().includes('2026'))
      expect(timeCell, '消费时间应渲染出格式化日期').toBeDefined()
      expect(timeCell.text()).toContain('2026')
      expect(wrapper.get('[data-testid="referral-pending-rate"]').text()).toBe('待入账佣金（12%）')
      expect(wrapper.get('[data-testid="referral-settled-rate"]').text()).toBe('已入账佣金（12%）')
      expect(wrapper.text()).not.toContain('待入账佣金（5%）')
      expect(wrapper.text()).not.toContain('已入账佣金（5%）')
    })

    it('does not substitute 5% when commission_rate_display is empty', async () => {
      const { apiFetch } = await import('../utils/apiUtils.js')
      apiFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          referral_count: 0,
          own_recharge_total: 0,
          referred_recharge_total: 0,
          referred_users: [],
          commission: {
            pending_points: 0,
            settled_points: 0,
            settle_delay_days: 7,
            commission_rate_display: '',
            items: [],
          },
        }),
        headers: { get: () => null },
      })
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-pending-rate"]').text()).not.toContain('5%')
      expect(wrapper.get('[data-testid="referral-settled-rate"]').text()).not.toContain('5%')
      expect(wrapper.get('[data-testid="referral-pending-rate"]').text()).toContain('—')
    })

    it('佣金明细为空时不渲染消费时间列', async () => {
      // 数据无 commission.items —— 该区块不渲染，不应抛错
      const { apiFetch } = await import('../utils/apiUtils.js')
      apiFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ referral_count: 0, own_recharge_total: 0, referred_recharge_total: 0, referred_users: [] }),
        headers: { get: () => null },
      })
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      expect(wrapper.find('table').exists()).toBe(false)
    })

    it('自身支付有流水时渲染 own_payments 表格（时间/金额/来源）', async () => {
      const { apiFetch } = await import('../utils/apiUtils.js')
      apiFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          referral_count: 1,
          own_recharge_total: '1.55',
          own_recharge_count: 2,
          referred_recharge_total: '0.00',
          referred_users: [],
          own_payments: [
            { paid_at: '2026-08-21T10:00:00Z', amount: '0.55', points_source_type: 'resource_purchase' },
            { paid_at: '2026-08-21T09:00:00Z', amount: '1.50', points_source_type: 'user_recharge_wechat' },
          ],
          commission: {
            pending_points: 0,
            settled_points: 0,
            settle_delay_days: 7,
            commission_rate_display: '12%',
            items: [],
          },
        }),
        headers: { get: () => null },
      })
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      const table = wrapper.get('[data-testid="referral-own-payments-table"]')
      expect(table.exists()).toBe(true)
      expect(wrapper.findAll('[data-testid="referral-own-payments-table"] tbody tr').length).toBe(2)
      expect(wrapper.text()).toContain('0.55')
      expect(wrapper.text()).toContain('1.5')
      expect(wrapper.text()).toContain('微信支付')
      expect(wrapper.text()).not.toContain('暂无自身支付记录')
    })

    it('自身支付无流水时展示空状态', async () => {
      const { apiFetch } = await import('../utils/apiUtils.js')
      apiFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          referral_count: 0,
          own_recharge_total: '0.00',
          own_recharge_count: 0,
          referred_recharge_total: '0.00',
          referred_users: [],
          commission: { pending_points: 0, settled_points: 0, settle_delay_days: 7, commission_rate_display: '12%', items: [] },
        }),
        headers: { get: () => null },
      })
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      expect(wrapper.text()).toContain('暂无自身支付记录')
      expect(wrapper.find('[data-testid="referral-own-payments-table"]').exists()).toBe(false)
    })

    it('被推荐人 as_referred 时展示自身支付明细而非空状态', async () => {
      const { apiFetch } = await import('../utils/apiUtils.js')
      apiFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          referral_count: 0,
          own_recharge_total: '0.55',
          own_recharge_count: 1,
          referred_recharge_total: '0.00',
          referred_users: [],
          as_referred: {
            referrer_user_id: 'referrer-c',
            recharge_count: 1,
            recharge_total: '0.55',
          },
        }),
        headers: { get: () => null },
      })
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'referred-c' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      expect(wrapper.text()).toContain('推荐用户支付明细（1 人）')
      expect(wrapper.text()).toContain('该用户由 referrer-c 推荐')
      expect(wrapper.text()).not.toContain('暂无推荐用户支付记录')
      expect(wrapper.text()).toContain('referred-c')
      expect(wrapper.text()).toContain('0.55')
    })

    it('默认展示支付明细且不请求 profit-sharing', async () => {
      const { apiFetch } = await import('../utils/apiUtils.js')
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-tab-payments"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('支付明细')
      expect(wrapper.find('[data-testid="referral-wechat-profit-sharing-tab"]').exists()).toBe(false)
      const urls = apiFetch.mock.calls.map((c) => String(c[0]))
      expect(urls.some((u) => u.includes('profit-sharing'))).toBe(false)
    })

    it('点击微信分账 Tab 才请求 referrer_user_id', async () => {
      const { apiFetch } = await import('../utils/apiUtils.js')
      apiFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          referral_count: 0,
          own_recharge_total: 0,
          referred_recharge_total: 0,
          referred_users: [],
          commission: { pending_points: 0, settled_points: 0, settle_delay_days: 7, commission_rate_display: '12%', items: [] },
        }),
        headers: { get: () => null },
      })
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'drawer-user' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      apiFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ items: [], total: 0, receiver_registration_status: '' }),
        headers: { get: () => null },
      })
      await wrapper.get('[data-testid="referral-tab-wechat-ps"]').trigger('click')
      await flushPromises()
      const last = String(apiFetch.mock.calls.at(-1)[0])
      expect(last).toContain('/api/system-admin/profit-sharing/')
      expect(last).toContain('referrer_user_id=drawer-user')
      expect(last).toContain('status=all')
    })

    it('maps APISIX 502 HTML to retryable 服务暂时不可用', async () => {
      const { apiFetch } = await import('../utils/apiUtils.js')
      apiFetch.mockResolvedValueOnce({
        ok: false,
        status: 502,
        traceId: '2fb383c5-baf3-41d3-b869-2de4761b044f',
        _errorData: {
          _rawErrorText:
            '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>',
        },
        json: async () => {
          throw new SyntaxError('Unexpected token <')
        },
        headers: { get: () => '2fb383c5-baf3-41d3-b869-2de4761b044f' },
      })
      const wrapper = mount(ReferralDrawer, {
        props: { visible: false, userId: 'u1' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      expect(wrapper.text()).toContain('服务暂时不可用，请稍后重试')
      expect(wrapper.text()).not.toMatch(/加载失败$/)
      expect(wrapper.get('[data-testid="referral-load-retry"]').exists()).toBe(true)
    })
  })
}
