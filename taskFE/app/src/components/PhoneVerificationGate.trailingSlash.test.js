// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PhoneVerificationGate.trailingSlash.test.js requires vitest runtime')
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

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../utils/apiUtils', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { default: PhoneVerificationGate } = await import('./PhoneVerificationGate.vue')

function jsonResponse(body, { ok = true, status = 200 } = {}) {
  return { ok, status, json: async () => body }
}

describe('PhoneVerificationGate 验证 URL 尾部斜杠（回归：not found 事故）', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('点击验证发起 billing/verify-phone-code/ 请求且带尾部斜杠', async () => {
    const urls = []
    apiFetch.mockImplementation(async (url) => {
      urls.push(String(url))
      if (String(url).includes('send_verification_code')) return jsonResponse({})
      return jsonResponse({ phone_masked: '138****8000' })
    })

    const wrapper = mount(PhoneVerificationGate, {
      props: {
        tenantId: '850256677331562496',
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

    // 1. 发送验证码（走 accounts send_verification_code，非 billing）
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('验证码已发送')

    // 2. 输入 6 位验证码并点击「验证」
    await wrapper.find('input[placeholder="6 位短信验证码"]').setValue('123456')
    await wrapper.findAll('button')[1].trigger('click')
    await flushPromises()

    const verifyUrl = urls.find((u) => u.includes('verify-phone-code'))
    expect(verifyUrl).toBe('/api/tenant/850256677331562496/billing/verify-phone-code/')
    // 验证成功后门禁自身隐藏（visible 由 sms_verified 控制），通过事件确认成功
    expect(wrapper.emitted('verified')).toHaveLength(1)

    wrapper.unmount()
  })
})

}
