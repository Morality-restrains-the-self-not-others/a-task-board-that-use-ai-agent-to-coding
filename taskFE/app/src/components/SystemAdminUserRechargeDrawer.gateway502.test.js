// @vitest-environment jsdom
/**
 * 回归：网关 502 HTML（taskBill 重启窗口 connection refused）不得展示无信息的
 * 「加载支付记录失败」，须可读文案 + data-traceId + 可重试。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUserRechargeDrawer.gateway502.test.js requires vitest runtime')
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

  const TRACE = '2fb383c5-baf3-41d3-b869-2de4761b044f'
  const GATEWAY_HTML =
    '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>'

  function gateway502() {
    return {
      ok: false,
      status: 502,
      traceId: TRACE,
      headers: { get: (n) => (String(n).toLowerCase() === 'x-trace-id' ? TRACE : null) },
      _errorData: { _rawErrorText: GATEWAY_HTML },
      json: async () => {
        throw new SyntaxError('Unexpected token <')
      },
      clone: function clone() {
        return this
      },
    }
  }

  function jsonOk(body) {
    return {
      ok: true,
      status: 200,
      headers: { get: () => 'application/json' },
      json: async () => body,
      clone: function clone() {
        return this
      },
    }
  }

  describe('SystemAdminUserRechargeDrawer gateway 502', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
    })

    it('shows retryable 服务暂时不可用 and keeps data-traceId on APISIX 502 HTML', async () => {
      hoisted.apiFetch.mockResolvedValue(gateway502())

      const wrapper = mount(Page, {
        props: { visible: false, userId: '9900000000000000001' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()

      expect(wrapper.text()).toContain('服务暂时不可用，请稍后重试')
      expect(wrapper.text()).not.toContain('加载支付记录失败')
      const errEl = wrapper.get('p.text-red-600')
      const tid = errEl.attributes('data-traceid') || errEl.attributes('data-traceId')
      expect(tid).toBe(TRACE)
      expect(wrapper.get('[data-testid="recharge-load-retry"]').exists()).toBe(true)
    })

    it('retry reloads payment records after gateway recovers', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(gateway502())
        .mockResolvedValueOnce(jsonOk({ recharges: [], consents: [] }))

      const wrapper = mount(Page, {
        props: { visible: false, userId: '9900000000000000001' },
      })
      await wrapper.setProps({ visible: true })
      await flushPromises()
      expect(wrapper.text()).toContain('服务暂时不可用，请稍后重试')

      await wrapper.get('[data-testid="recharge-load-retry"]').trigger('click')
      await flushPromises()

      expect(wrapper.text()).toContain('暂无支付记录')
      expect(wrapper.text()).not.toContain('服务暂时不可用')
      expect(hoisted.apiFetch).toHaveBeenCalledTimes(2)
    })
  })
}
