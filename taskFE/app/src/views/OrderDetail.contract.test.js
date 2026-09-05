// @vitest-environment jsdom
// 订单详情页展示下单资源；支付心跳在弹窗关闭/卸载后停止。
if (!process.env.VITEST) {
  console.log('[skip] OrderDetail.contract.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    renderQrToCanvas: vi.fn(),
    useRoute: () => ({
      params: { tenant: '873472655125147648', orderId: '1' },
      query: {},
      fullPath: '/',
      path: '/',
    }),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    extractErrorMessage: () => 'mock-err',
  }))
  vi.mock('../composables/useQrCodeCanvas.js', () => ({
    renderQrToCanvas: (...args) => mocks.renderQrToCanvas(...args),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => '' }))
  vi.mock('vue-router', () => ({ useRoute: () => mocks.useRoute() }))

  const { default: OrderDetail } = await import('./OrderDetail.vue')

  const ORDER = {
    id: '1',
    order_number: 'ORD-TEST-001',
    status: 'pending',
    total_yuan: '3.00',
    buyer_note: '请开通 team-foo',
    items: [
      { id: 'i1', resource_type: 'task_post', quantity: 2, unit_price_yuan: '1.00', subtotal_yuan: '2.00' },
      { id: 'i2', resource_type: 'gitlab_disk', quantity: 1, unit_price_yuan: '1.00', subtotal_yuan: '1.00', region: 'tencent-sh-1' },
    ],
  }

  function mockApi(overrides = {}) {
    mocks.apiFetch.mockImplementation((url, options = {}) => {
      const method = String(options.method || 'GET').toUpperCase()
      const key = `${method} ${url}`
      const hit = overrides[key]
      const body = hit !== undefined ? hit : null
      return Promise.resolve({
        ok: body !== null,
        status: body !== null ? 200 : 404,
        json: () => Promise.resolve(body ?? {}),
      })
    })
  }

  const pollGets = () =>
    mocks.apiFetch.mock.calls.filter(([url, options]) =>
      url.includes('/billing/orders/1/') &&
      String(options?.method || 'GET').toUpperCase() === 'GET' &&
      !url.includes('/pay/'))

  async function startPolling(wrapper) {
    const payBtn = wrapper.findAll('button').find((b) => b.text().includes('确认支付'))
    await payBtn.trigger('click')
    await flushPromises()
  }

  describe('OrderDetail 资源行项与支付轮询', () => {
    beforeEach(() => {
      vi.useFakeTimers()
      mocks.apiFetch.mockReset()
      mocks.renderQrToCanvas.mockReset()
      mockApi({
        'GET /api/tenant/873472655125147648/billing/phone-verification-status/': { required: false, has_phone: true, sms_verified: true },
        'GET /api/tenant/873472655125147648/billing/orders/1/': ORDER,
        'POST /api/tenant/873472655125147648/billing/orders/1/pay/': { code_url: 'mock-url', out_trade_no: 'MOCK-OUT', mode: 'mock' },
      })
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('展示用户下单的资源类型与数量', async () => {
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      const items = wrapper.find('[data-testid="order-detail-items"]')
      expect(items.exists()).toBe(true)
      expect(items.text()).toContain('任务帖')
      expect(items.text()).toContain('GitLab 磁盘')
      expect(items.text()).toContain('tencent-sh-1')
      expect(items.text()).toContain('2')
      expect(wrapper.find('[data-testid="order-buyer-note"]').text()).toContain('请开通 team-foo')
      wrapper.unmount()
      vi.useRealTimers()
    })

    it('支付确认后按间隔轮询订单状态', async () => {
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      const beforePay = pollGets().length
      await startPolling(wrapper)
      await vi.advanceTimersByTimeAsync(10000)
      expect(pollGets().length).toBe(beforePay + 1)
      await vi.advanceTimersByTimeAsync(10000)
      expect(pollGets().length).toBe(beforePay + 2)
      wrapper.unmount()
      vi.useRealTimers()
    })

    it('点击「关闭」后轮询立即停止', async () => {
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await startPolling(wrapper)
      await vi.advanceTimersByTimeAsync(10000)
      const beforeClose = pollGets().length
      expect(beforeClose).toBeGreaterThanOrEqual(1)
      const closeBtn = wrapper.findAll('button').find((b) => b.text().trim() === '关闭')
      await closeBtn.trigger('click')
      await vi.advanceTimersByTimeAsync(30_000)
      expect(pollGets().length).toBe(beforeClose)
      wrapper.unmount()
      vi.useRealTimers()
    })

    it('支付弹窗打开后把 code_url 绘到 canvas，且不展示模拟支付', async () => {
      mocks.renderQrToCanvas.mockResolvedValue({ success: true, method: 'qrcode' })
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await startPolling(wrapper)
      await flushPromises()
      expect(wrapper.text()).toContain('微信扫码支付')
      expect(mocks.renderQrToCanvas).toHaveBeenCalled()
      const [canvas, text] = mocks.renderQrToCanvas.mock.calls[0]
      expect(canvas).toBeTruthy()
      expect(canvas.tagName).toBe('CANVAS')
      expect(text).toBe('mock-url')
      expect(wrapper.text()).not.toContain('模拟支付成功')
      expect(wrapper.text()).not.toContain('当前为测试模式')
      wrapper.unmount()
      vi.useRealTimers()
    })

    it('组件卸载后轮询立即停止', async () => {
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await startPolling(wrapper)
      await vi.advanceTimersByTimeAsync(10000)
      const beforeUnmount = pollGets().length
      expect(beforeUnmount).toBeGreaterThanOrEqual(1)
      wrapper.unmount()
      await vi.advanceTimersByTimeAsync(30_000)
      expect(pollGets().length).toBe(beforeUnmount)
      vi.useRealTimers()
    })
  })
}
