// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminReferralManagement.policy.test.js requires vitest runtime')
} else {
const { describe, it, expect, vi, beforeEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

const { apiFetch } = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch,
}))
// OPT-20260825-013: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
// 断言 POST 携带 Idempotency-Key 头（与 SystemAdminLoginPaymentPolicy 同构）。
vi.mock('../utils/clickGuard.js', () => ({
  createClickGuard: () => ({
    run: async (fn) => fn({ idempotencyKey: 'ik-test-1' }),
  }),
  mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
}))

const { default: View } = await import('./SystemAdminReferralManagement.vue')

function jsonOk(body) {
  return {
    ok: true,
    json: async () => body,
    headers: { get: () => 'application/json' },
  }
}

describe('SystemAdminReferralManagement 保存策略/入账天数 clickGuard 接线', () => {
  beforeEach(() => {
    apiFetch.mockReset()
    apiFetch.mockImplementation(async (url, opts) => {
      if (opts?.method === 'POST') {
        if (url === '/api/system-admin/referral/policy/') return jsonOk({ data: { mode: 'approval', message: '' } })
        if (url === '/api/system-admin/referral/config/update/') return jsonOk({ data: { settle_delay_days: 15 } })
      }
      if (url === '/api/system-admin/referral/policy/') return jsonOk({ data: { mode: 'approval', message: '' } })
      if (url === '/api/system-admin/referral/config/') return jsonOk({ data: { settle_delay_days: 15 } })
      return jsonOk({})
    })
  })

  it('点击保存策略发起一次 POST 且携带 Idempotency-Key', async () => {
    const wrapper = mount(View)
    await flushPromises()
    const btn = wrapper.findAll('button').find((b) => b.text() === '保存策略')
    expect(btn).toBeTruthy()
    await btn.trigger('click')
    await flushPromises()

    const postCall = apiFetch.mock.calls.find(([url, opts]) => opts?.method === 'POST' && url === '/api/system-admin/referral/policy/')
    expect(postCall).toBeTruthy()
    expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    expect(postCall[1].body).toContain('"mode":"approval"')
  })

  it('点击保存入账天数发起一次 POST 且携带 Idempotency-Key', async () => {
    const wrapper = mount(View)
    await flushPromises()
    const btn = wrapper.findAll('button').find((b) => b.text() === '保存')
    expect(btn).toBeTruthy()
    await btn.trigger('click')
    await flushPromises()

    const postCall = apiFetch.mock.calls.find(([url, opts]) => opts?.method === 'POST' && url === '/api/system-admin/referral/config/update/')
    expect(postCall).toBeTruthy()
    expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    expect(postCall[1].body).toContain('"settle_delay_days"')
  })

  it('保存策略在网关 502 HTML 时展示重试文案与 data-traceId', async () => {
    apiFetch.mockImplementation(async (url, opts) => {
      if (opts?.method === 'POST' && url === '/api/system-admin/referral/policy/') {
        return {
          ok: false,
          status: 502,
          traceId: 'tr-policy-502',
          headers: { get: () => null },
          _errorData: {
            _rawErrorText: '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>',
          },
          json: async () => { throw new Error('body is html') },
        }
      }
      if (url === '/api/system-admin/referral/policy/') return jsonOk({ data: { mode: 'approval', message: '' } })
      if (url === '/api/system-admin/referral/config/') return jsonOk({ data: { settle_delay_days: 15 } })
      return jsonOk({})
    })
    const wrapper = mount(View)
    await flushPromises()
    const btn = wrapper.findAll('button').find((b) => b.text() === '保存策略')
    expect(btn).toBeTruthy()
    await btn.trigger('click')
    await flushPromises()

    const err = wrapper.find('.bg-red-50')
    expect(err.exists()).toBe(true)
    expect(err.text()).toContain('服务暂时不可用，请稍后重试')
    const attrs = err.attributes()
    expect(attrs['data-trace-id'] || attrs['data-traceid']).toBe('tr-policy-502')
  })
})
}
