// @vitest-environment jsdom
/**
 * 超管「开启退款申请」开关：切换后须二次确认；取消则回滚且不发 PUT。
 * 面板挂载于订单与退款页的 refund Tab。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.policy-confirm.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    confirm: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('../../utils/modalService.js', () => ({
    default: {
      confirm: (...args) => hoisted.confirm(...args),
    },
  }))

  const Page = (await import('./SystemAdminRefundPanel.vue')).default

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

  describe('SystemAdminRefundPanel 退款策略开关二次确认', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.confirm.mockReset()
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/') && (!opts.method || opts.method === 'GET')) {
          return jsonOk({ enabled: true })
        }
        if (u.includes('/refund-policy/') && opts.method === 'PUT') {
          const body = JSON.parse(opts.body || '{}')
          return jsonOk({ enabled: body.enabled === true })
        }
        if (u.includes('/refund-applications/')) {
          return jsonOk({ results: [] })
        }
        return jsonOk({})
      })
    })

    it('确认后才 PUT 关闭策略', async () => {
      hoisted.confirm.mockResolvedValue(true)
      const wrapper = mount(Page)
      await flushPromises()

      const toggle = wrapper.get('[data-testid="enable-refund-applications-switch"]')
      expect(toggle.element.checked).toBe(true)

      await toggle.setValue(false)
      await flushPromises()

      expect(hoisted.confirm).toHaveBeenCalled()
      const putCalls = hoisted.apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'PUT')
      expect(putCalls).toHaveLength(1)
      expect(JSON.parse(putCalls[0][1].body)).toEqual({ enabled: false })
      expect(toggle.element.checked).toBe(false)
    })

    it('取消确认时回滚开关且不发 PUT', async () => {
      hoisted.confirm.mockRejectedValue(false)
      const wrapper = mount(Page)
      await flushPromises()

      const toggle = wrapper.get('[data-testid="enable-refund-applications-switch"]')
      expect(toggle.element.checked).toBe(true)

      await toggle.setValue(false)
      await flushPromises()

      expect(hoisted.confirm).toHaveBeenCalled()
      const putCalls = hoisted.apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'PUT')
      expect(putCalls).toHaveLength(0)
      expect(toggle.element.checked).toBe(true)
      expect(wrapper.text()).toContain('已开启')
    })
  })
}
