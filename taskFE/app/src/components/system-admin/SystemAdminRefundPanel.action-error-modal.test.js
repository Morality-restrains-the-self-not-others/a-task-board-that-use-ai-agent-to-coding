// @vitest-environment jsdom
/**
 * 超管批准/拒绝退款失败时，错误必须出现在弹窗内（含 data-traceId），
 * 不能只写在被遮罩挡住的页面横幅；渠道 dump 不得含签名头。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.action-error-modal.test.js requires vitest runtime')
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

  const WECHAT_DUMP = `error http response:[StatusCode: 403 Code: "NOT_ENOUGH"
Message: 基本账户余额不足，请充值后重新发起
Header:
 - Wechatpay-Signature=[NVkk5cJ3rdfE9Yl2OJ0s1RpF0cRBqqXxO8y6SrL3ESVtkFULLrg68e68jWJwDLMykeIhCbCqIgTXubZgfG+lGzRLCvgWosXUiNUl6mziU4LQ4G56qHwgfcBKVOt20CbFwMVaDHkgxVX3T+YRecFpApL/RtCbObtUn618D2oCOybQSO/ZmDRBzfz4bro9BgXXYlqIKD5S1fPvTFKVxahKvOy9pG2imaTYy05LVXGYNd/bvcQEHaRJFW2rwkf+DlssF2SpdNIYa+4ke8+XXPrbrbp+uvqlVd63+dHr0paBlikuzF25ZO9CSHGmCgrJ9+s5cwuZ88vDBLZQC4R+eQSAGw==]
]`

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

  function jsonFail(body, status, traceId) {
    return {
      ok: false,
      status,
      headers: {
        get: (name) => {
          const k = String(name).toLowerCase()
          if (k === 'x-trace-id') return traceId
          if (k === 'content-type') return 'application/json'
          return ''
        },
      },
      json: async () => body,
      clone: function () {
        return this
      },
    }
  }

  describe('SystemAdminRefundPanel 审批失败错误在弹窗内', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.confirm.mockReset()
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/') && (!opts.method || opts.method === 'GET')) {
          return jsonOk({ enabled: true })
        }
        if (u.includes('/refund-applications/') && opts.method === 'POST') {
          return jsonFail(
            { error: WECHAT_DUMP, trace_id: '06d29f3d-f970-4697-95d9-9795a2a016e4' },
            500,
            '06d29f3d-f970-4697-95d9-9795a2a016e4',
          )
        }
        if (u.includes('/refund-applications/')) {
          return jsonOk({ results: [pendingRow] })
        }
        return jsonOk({})
      })
    })

    it('批准失败：弹窗展示可读错误与 data-traceId，不含签名，弹窗保持打开', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-approve-btn"]').trigger('click')
      await flushPromises()

      const modal = wrapper.get('[data-testid="refund-approve-modal"]')
      await modal.get('[data-testid="refund-approve-confirm-btn"]').trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="refund-approve-modal"]').exists()).toBe(true)
      const errEl = wrapper.get('[data-testid="refund-approve-action-error"]')
      expect(errEl.text()).toContain('余额不足')
      expect(errEl.text()).not.toContain('Wechatpay-Signature')
      expect(errEl.element.getAttribute('data-traceId')).toBe(
        '06d29f3d-f970-4697-95d9-9795a2a016e4',
      )
    })

    it('拒绝失败：弹窗同样展示错误与 data-traceId', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-reject-btn"]').trigger('click')
      await flushPromises()

      const modal = wrapper.get('[data-testid="refund-reject-modal"]')
      await modal.get('[data-testid="refund-reject-confirm-btn"]').trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="refund-reject-modal"]').exists()).toBe(true)
      const errEl = wrapper.get('[data-testid="refund-reject-action-error"]')
      expect(errEl.text()).toContain('余额不足')
      expect(errEl.element.getAttribute('data-traceId')).toBe(
        '06d29f3d-f970-4697-95d9-9795a2a016e4',
      )
    })
  })
}
