// @vitest-environment jsdom
/**
 * 退款审批：列表展示申请原因；批准时弹窗展示申请原因且默认退款原因为「订单退款」。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.approve-reason.test.js requires vitest runtime')
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

  const pendingRow = {
    id: '877617449518792704',
    tenant_id: '877397588196749312',
    order_id: '8775960076914851840',
    frozen_points: 55,
    status: 'pending',
    reason: '体验不满意，申请全额退款',
    created_at: '2026-08-18T18:18:00Z',
  }

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

  describe('SystemAdminRefundPanel 申请原因与批准退款原因', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.confirm.mockReset()
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/') && (!opts.method || opts.method === 'GET')) {
          return jsonOk({ enabled: true })
        }
        if (u.includes('/refund-applications/') && opts.method === 'POST') {
          return jsonOk({ ...pendingRow, status: 'approved' })
        }
        if (u.includes('/refund-applications/')) {
          return jsonOk({ results: [pendingRow] })
        }
        return jsonOk({})
      })
    })

    it('列表展示申请原因列', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      expect(wrapper.text()).toContain('申请原因')
      expect(wrapper.text()).toContain('体验不满意，申请全额退款')
    })

    it('批准弹窗展示申请原因且默认退款原因为「订单退款」', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-approve-btn"]').trigger('click')
      await flushPromises()

      const modal = wrapper.get('[data-testid="refund-approve-modal"]')
      expect(modal.text()).toContain('体验不满意，申请全额退款')
      const reasonInput = modal.get('[data-testid="refund-approve-reason-input"]')
      expect(reasonInput.element.value).toBe('订单退款')
    })

    it('确认批准时 POST note 为管理员填写的退款原因', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-approve-btn"]').trigger('click')
      await flushPromises()

      const modal = wrapper.get('[data-testid="refund-approve-modal"]')
      const reasonInput = modal.get('[data-testid="refund-approve-reason-input"]')
      await reasonInput.setValue('协商部分退款')
      await modal.get('[data-testid="refund-approve-confirm-btn"]').trigger('click')
      await flushPromises()

      const postCall = hoisted.apiFetch.mock.calls.find(
        ([url, opts]) =>
          String(url).includes('/refund-applications/877617449518792704/approve/') &&
          opts?.method === 'POST'
      )
      expect(postCall).toBeTruthy()
      expect(JSON.parse(postCall[1].body)).toEqual({ note: '协商部分退款' })
    })
  })
}
