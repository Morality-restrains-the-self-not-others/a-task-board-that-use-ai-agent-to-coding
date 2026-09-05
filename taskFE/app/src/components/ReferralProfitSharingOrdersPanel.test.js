// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ReferralProfitSharingOrdersPanel.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
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

  function jsonError(status, body) {
    return Promise.resolve({ ok: false, status, json: async () => body })
  }

  const channelsFixture = [
    {
      channel_code: 'CH-A',
      period_from: '2026-08-04T12:00:00Z',
      period_to: '2026-08-19T12:00:00Z',
      order_amount_yuan_cents: 3500,
      frozen_amount_yuan_cents: 50,
      failed_amount_yuan_cents: 80,
      shareable_amount_yuan_cents: 125,
      shareable: true,
    },
    {
      channel_code: 'CH-B',
      period_from: '2026-08-06T00:00:00Z',
      period_to: '2026-08-06T00:00:00Z',
      order_amount_yuan_cents: 600,
      frozen_amount_yuan_cents: 0,
      failed_amount_yuan_cents: 0,
      shareable_amount_yuan_cents: 0,
      shareable: false,
    },
  ]

  beforeEach(() => {
    hoisted.apiFetchMock.mockReset()
    hoisted.apiFetchMock.mockImplementation(() => jsonOk({ channels: channelsFixture }))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('ReferralProfitSharingOrdersPanel', () => {
    it('loads channel aggregates from the billing API on mount', async () => {
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      const urls = hoisted.apiFetchMock.mock.calls.map((c) => String(c[0]))
      expect(urls).toContain('/api/billing/profit-sharing/referrer-orders/')
      const text = wrapper.get('[data-testid="referral-ps-orders-table"]').text()
      expect(text).toContain('CH-A')
      expect(text).toContain('渠道号')
      expect(text).not.toContain('ORD-')
      expect(text).not.toContain('订单号')
    })

    it('renders period and amount columns without referred order numbers', async () => {
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      const text = wrapper.get('[data-testid="referral-ps-orders-table"]').text()
      expect(text).toContain('2026-08-04 ~ 2026-08-19')
      expect(text).toContain('¥35.00')
      expect(text).toContain('¥0.50')
      expect(text).toContain('¥1.25')
      expect(text).toContain('¥6.00')
    })

    it('renders the 失败金额 column from failed_amount_yuan_cents (OPT-20260824-086)', async () => {
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      const tableText = wrapper.get('[data-testid="referral-ps-orders-table"]').text()
      expect(tableText).toContain('失败金额')
      const failedCells = wrapper.findAll('[data-testid="referral-ps-failed-amount"]')
      expect(failedCells).toHaveLength(2)
      expect(failedCells[0].text()).toBe('¥0.80')
      expect(failedCells[1].text()).toBe('¥0.00')
    })

    it('shows the 分账 button only for shareable channels', async () => {
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      const buttons = wrapper.findAll('[data-testid="referral-ps-share-btn"]')
      expect(buttons).toHaveLength(1)
      expect(buttons[0].text()).toBe('分账')
    })

    it('shows the empty state when no channels exist', async () => {
      hoisted.apiFetchMock.mockImplementation(() => jsonOk({ channels: [] }))
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-orders-empty"]').text()).toContain('暂无渠道分账数据')
      expect(wrapper.find('[data-testid="referral-ps-orders-table"]').exists()).toBe(false)
    })

    it('surfaces load errors with trace id', async () => {
      hoisted.apiFetchMock.mockImplementation(() => jsonError(500, { error: 'boom' }))
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-orders-error"]').text()).toContain('boom')
    })

    it('posts to share-channel and reloads the list on success', async () => {
      const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
      hoisted.apiFetchMock
        .mockImplementationOnce(() => jsonOk({ channels: channelsFixture }))
        .mockImplementationOnce(() => jsonOk({ status: 'ok', state: 'shared' }))
        .mockImplementationOnce(() => jsonOk({ channels: channelsFixture }))
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-share-btn"]').trigger('click')
      await flushPromises()
      expect(confirmSpy).toHaveBeenCalledTimes(1)
      expect(String(confirmSpy.mock.calls[0][0])).toContain('CH-A')
      expect(String(confirmSpy.mock.calls[0][0])).not.toContain('ORD-')
      const postCall = hoisted.apiFetchMock.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(String(postCall[0])).toBe('/api/billing/profit-sharing/referrer-orders/share-channel/')
      expect(String(postCall[1]?.headers?.['Idempotency-Key'] || '')).not.toBe('')
      const body = JSON.parse(String(postCall[1]?.body || '{}'))
      expect(body.channel_code).toBe('CH-A')
      expect(hoisted.apiFetchMock.mock.calls.length).toBe(3)
    })

    it('shows the share error when the backend rejects the request', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      hoisted.apiFetchMock
        .mockImplementationOnce(() => jsonOk({ channels: channelsFixture }))
        .mockImplementationOnce(() => jsonError(409, { error: '当前状态不可分账' }))
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-share-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-share-error"]').text()).toContain('当前状态不可分账')
    })

    it('maps empty receiver PARAM_ERROR to bind-wechat copy', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      hoisted.apiFetchMock
        .mockImplementationOnce(() => jsonOk({ channels: channelsFixture }))
        .mockImplementationOnce(() => jsonError(400, {
          error: 'create profit sharing failed: HTTP 400, {"code":"PARAM_ERROR","message":"输入源“/body/receivers/0/account”映射到值字段“分账接收方帐号”字符串规则校验失败，字符数 0，小于最小值 1"}',
        }))
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-share-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-share-error"]').text()).toBe('推荐人未绑定微信收款账号')
    })

    it('does not POST when the user cancels the confirm dialog', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(false)
      hoisted.apiFetchMock.mockImplementation(() => jsonOk({ channels: channelsFixture }))
      const { default: Panel } = await import('./ReferralProfitSharingOrdersPanel.vue')
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-share-btn"]').trigger('click')
      await flushPromises()
      const postCalls = hoisted.apiFetchMock.mock.calls.filter((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(postCalls).toHaveLength(0)
    })
  })
}
