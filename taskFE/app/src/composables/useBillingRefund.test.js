// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBillingRefund.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { tenant: '850256677331562496' } }),
}))

vi.mock('../utils/cookieUtils', () => ({
  getCookie: () => 'csrf-test',
}))

vi.mock('../utils/config.js', () => ({
  getApiUrl: (p) => `http://test${p}`,
}))

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { useBillingRefund } = await import('./useBillingRefund.js')

function jsonResponse(body, { ok = true, status = 200, headers = {} } = {}) {
  return {
    ok,
    status,
    headers: {
      get: (k) => headers[k] || headers[String(k).toLowerCase()] || null,
    },
    json: async () => body,
  }
}

describe('useBillingRefund', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('loads pending application from list results', async () => {
    apiFetch.mockImplementation(async (url) => {
      if (String(url).includes('/balance/')) {
        return jsonResponse({ cents: 0, frozen_balance: 500 })
      }
      return jsonResponse({
        results: [{ id: '1', status: 'pending', frozen_cents: 500 }],
      })
    })
    const c = useBillingRefund()
    await c.refresh()
    expect(c.frozenBalance.value).toBe(500)
    expect(c.pendingApplication.value?.status).toBe('pending')
    expect(c.applications.value).toHaveLength(1)
  })

  it('sets applyError and data-traceId source on failed apply', async () => {
    apiFetch.mockImplementation(async (url, opts) => {
      if (opts?.method === 'POST') {
        return jsonResponse(
          { error: '该租户已有待审批的退款申请' },
          { ok: false, status: 409, headers: { 'X-Trace-Id': 'tr-refund-1' } },
        )
      }
      return jsonResponse({ results: [] })
    })
    const c = useBillingRefund()
    const ok = await c.applyRefund('unused', 'order-1')
    expect(ok).toBe(false)
    expect(c.applyError.value).toContain('待审批')
    expect(c.applyErrorTraceId.value).toBe('tr-refund-1')
  })

  it('passes order_id in request body when provided', async () => {
    let capturedBody = null
    apiFetch.mockImplementation(async (url, opts) => {
      if (String(url).includes('/balance/')) {
        return jsonResponse({ cents: 500, frozen_balance: 0 })
      }
      if (opts?.method === 'POST') {
        capturedBody = JSON.parse(opts.body || '{}')
        return jsonResponse({ id: '9003', status: 'pending', frozen_points: 500, reason: 'test', order_id: '123456' })
      }
      return jsonResponse({ results: [] })
    })
    const c = useBillingRefund()
    const ok = await c.applyRefund('test reason', '123456')
    expect(ok).toBe(true)
    expect(capturedBody).not.toBeNull()
    expect(capturedBody.order_id).toBe('123456')
    expect(capturedBody.reason).toBe('test reason')
  })

  it('sends Idempotency-Key when provided', async () => {
    let capturedHeaders = null
    apiFetch.mockImplementation(async (url, opts) => {
      if (String(url).includes('/balance/')) {
        return jsonResponse({ cents: 500, frozen_balance: 0 })
      }
      if (opts?.method === 'POST') {
        capturedHeaders = opts.headers || {}
        return jsonResponse({ id: '9005', status: 'pending', frozen_points: 500, reason: 'k', order_id: '1' })
      }
      return jsonResponse({ results: [] })
    })
    const c = useBillingRefund()
    const ok = await c.applyRefund('k', '1', 'ik-abc-1')
    expect(ok).toBe(true)
    expect(capturedHeaders['Idempotency-Key']).toBe('ik-abc-1')
  })

  it('always includes order_id in request body', async () => {
    let capturedBody = null
    apiFetch.mockImplementation(async (url, opts) => {
      if (String(url).includes('/balance/')) {
        return jsonResponse({ cents: 500, frozen_balance: 0 })
      }
      if (opts?.method === 'POST') {
        capturedBody = JSON.parse(opts.body || '{}')
        return jsonResponse({ id: '9004', status: 'pending', frozen_points: 500, reason: 'with order', order_id: '789' })
      }
      return jsonResponse({ results: [] })
    })
    const c = useBillingRefund()
    const ok = await c.applyRefund('with order', '789')
    expect(ok).toBe(true)
    expect(capturedBody).not.toBeNull()
    expect(capturedBody.order_id).toBe('789')
    expect(capturedBody.reason).toBe('with order')
  })

  it('syncs refundEnabled=false from list and surfaces REFUND_DISABLED on apply', async () => {
    apiFetch.mockImplementation(async (url, opts) => {
      if (String(url).includes('/balance/')) {
        return jsonResponse({ cents: 100, frozen_balance: 0, refund_enabled: false })
      }
      if (opts?.method === 'POST') {
        return jsonResponse(
          { error: '平台未开启退款申请', code: 'REFUND_DISABLED' },
          { ok: false, status: 403, headers: { 'X-Trace-Id': 'tr-refund-off' } },
        )
      }
      return jsonResponse({ enabled: false, results: [] })
    })
    const c = useBillingRefund()
    await c.refresh()
    expect(c.refundEnabled.value).toBe(false)
    const ok = await c.applyRefund('should fail', 'order-2')
    expect(ok).toBe(false)
    expect(c.applyError.value).toContain('未开启')
    expect(c.applyErrorTraceId.value).toBe('tr-refund-off')
  })
})

}
