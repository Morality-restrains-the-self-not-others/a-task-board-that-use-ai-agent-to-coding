// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PhoneVerificationGate.click-guard.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('../utils/cookieUtils', () => ({
    getCookie: () => 'csrf-test',
  }))

  vi.mock('../utils/config.js', () => ({
    getApiUrl: (p) => `http://test${p}`,
  }))

  vi.mock('../composables/auth/useAllowedCountryCodes.js', () => ({
    useAllowedCountryCodes: () => ({
      filteredCountryOptions: [],
      defaultCountryPrefix: { value: '+86' },
      syncPrefixWithAllowedOptions: () => {},
      fetchAllowedCodes: () => Promise.resolve(),
      loading: { value: false },
      error: { value: '' },
    }),
  }))

  const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => apiFetch(...args),
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言发验证码/验证手机 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-phone-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: PhoneVerificationGate } = await import('./PhoneVerificationGate.vue')

  function jsonResponse(body, { ok = true } = {}) {
    return { ok, json: async () => body }
  }

  describe('PhoneVerificationGate 发验证码/验证手机实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('发送短信验证码 POST 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('send_verification_code')) return jsonResponse({})
        return jsonResponse({})
      })
      const wrapper = mount(PhoneVerificationGate, {
        props: {
          tenantId: 't-1',
          active: true,
          phoneStatus: {
            has_phone: true,
            phone_masked: '138****8000',
            phone_e164: '+8613800138000',
            sms_verified: false,
            required: true,
          },
        },
      })
      await flushPromises()
      await wrapper.findAll('button').find((b) => b.text().includes('发送验证码')).trigger('click')
      await flushPromises()

      const sendCall = apiFetch.mock.calls.find(([url]) => String(url).includes('send_verification_code'))
      expect(sendCall).toBeTruthy()
      expect(sendCall[1].method).toBe('POST')
      expect(sendCall[1].headers['Idempotency-Key']).toBe('ik-phone-1')
      expect(sendCall[1].body).toContain('"+8613800138000"')
      wrapper.unmount()
    })

    it('验证手机 POST 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('send_verification_code')) return jsonResponse({})
        return jsonResponse({ phone_masked: '138****8000' })
      })
      const wrapper = mount(PhoneVerificationGate, {
        props: {
          tenantId: 't-1',
          active: true,
          phoneStatus: {
            has_phone: true,
            phone_masked: '138****8000',
            phone_e164: '+8613800138000',
            sms_verified: false,
            required: true,
          },
        },
      })
      await flushPromises()
      await wrapper.find('input[placeholder="6 位短信验证码"]').setValue('123456')
      await wrapper.findAll('button').find((b) => b.text().trim() === '验证').trigger('click')
      await flushPromises()

      const verifyCall = apiFetch.mock.calls.find(([url]) => String(url).includes('verify-phone-code'))
      expect(verifyCall).toBeTruthy()
      expect(verifyCall[1].method).toBe('POST')
      expect(verifyCall[1].headers['Idempotency-Key']).toBe('ik-phone-1')
      expect(verifyCall[1].body).toContain('"code":"123456"')
      wrapper.unmount()
    })
  })
}
