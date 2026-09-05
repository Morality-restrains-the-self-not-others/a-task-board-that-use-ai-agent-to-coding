// @vitest-environment jsdom
/**
 * OPT-20260819-017 回归：拒绝退款必须走自定义模态框收集原因，
 * 禁止 window.prompt（前端「禁止浏览器原生弹窗」规范）。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.reject-modal.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    promptSpy: vi.fn(() => ''),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('../../utils/modalService.js', () => ({
    default: { confirm: vi.fn() },
  }))

  vi.stubGlobal('prompt', hoisted.promptSpy)

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

  describe('SystemAdminRefundPanel 拒绝退款模态框', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.promptSpy.mockReset()
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/') && (!opts.method || opts.method === 'GET')) {
          return jsonOk({ enabled: true })
        }
        if (u.includes('/refund-applications/') && opts.method === 'POST') {
          return jsonOk({ ...pendingRow, status: 'rejected' })
        }
        if (u.includes('/refund-applications/')) {
          return jsonOk({ results: [pendingRow] })
        }
        return jsonOk({})
      })
    })

    it('点击拒绝打开自定义模态框（不再 window.prompt）', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-reject-btn"]').trigger('click')
      await flushPromises()

      const modal = wrapper.get('[data-testid="refund-reject-modal"]')
      expect(modal.exists()).toBe(true)
      expect(modal.text()).toContain('拒绝退款')
      expect(hoisted.promptSpy).not.toHaveBeenCalled()
    })

    it('填写拒绝原因确认后 POST note', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-reject-btn"]').trigger('click')
      await flushPromises()

      const modal = wrapper.get('[data-testid="refund-reject-modal"]')
      await modal.get('[data-testid="refund-reject-note-input"]').setValue('凭证不符')
      await modal.get('[data-testid="refund-reject-confirm-btn"]').trigger('click')
      await flushPromises()

      const postCall = hoisted.apiFetch.mock.calls.find(
        ([url, opts]) =>
          String(url).includes('/refund-applications/877617449518792704/reject/') &&
          opts?.method === 'POST'
      )
      expect(postCall).toBeTruthy()
      expect(JSON.parse(postCall[1].body)).toEqual({ note: '凭证不符' })
      expect(hoisted.promptSpy).not.toHaveBeenCalled()
    })
  })
}
