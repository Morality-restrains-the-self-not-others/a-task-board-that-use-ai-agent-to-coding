// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminLoginPaymentPolicy.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头（与 OrderCreate 一致）。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./SystemAdminLoginPaymentPolicy.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminLoginPaymentPolicy 保存策略 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue(jsonOk({ data: { message: '保存成功' } }))
    })

    it('点击保存策略发起一次 POST 且携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET') {
          return jsonOk({
            data: {
              enable_phone_login: true,
              enable_recharge_phone_verification: true,
              enable_email_register: true,
              enable_wechat_login: true,
              allowed_phone_country_codes: ['86'],
            },
          })
        }
        return jsonOk({ data: { message: '保存成功' } })
      })
      const wrapper = mount(View, {
        global: {
          stubs: {
            'router-link': { template: '<a><slot /></a>' },
            'router-view': true,
          },
        },
      })
      await flushPromises()
      await wrapper.find('button').trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/system-admin/system-feature-policy/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
      expect(postCall[1].body).toContain('"enable_phone_login":true')
    })

    it('GET 加载成功后页面出现保存策略按钮', async () => {
      apiFetch.mockResolvedValue(jsonOk({
        data: {
          enable_phone_login: false,
          enable_recharge_phone_verification: false,
          enable_email_register: true,
          enable_wechat_login: false,
          allowed_phone_country_codes: [],
        },
      }))
      const wrapper = mount(View, {
        global: {
          stubs: {
            'router-link': { template: '<a><slot /></a>' },
            'router-view': true,
          },
        },
      })
      await flushPromises()
      expect(wrapper.text()).toContain('保存策略')
    })
  })
}
