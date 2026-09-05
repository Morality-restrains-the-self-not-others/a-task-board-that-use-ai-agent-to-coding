// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useLoginFeaturePolicy.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { ref } = await import('vue')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

describe('useLoginFeaturePolicy', () => {
  beforeEach(() => {
    apiFetch.mockReset()
    localStorage.clear()
  })

  // 2026-08-24：手机号+验证码登录已移除，isPhoneLoginMethod 仅剩 phonePassword
  it('disables phone login when public policy fetch fails', async () => {
    const { useLoginFeaturePolicy } = await import('./useLoginFeaturePolicy.js')
    apiFetch.mockRejectedValueOnce(new Error('network'))
    const loginMethod = ref('phonePassword')
    const phoneFields = { resetPhoneFields: vi.fn() }
    const { allowPhoneLogin, loadPublicFeaturePolicy } = useLoginFeaturePolicy({
      loginMethod,
      phoneFields,
    })
    await loadPublicFeaturePolicy()
    expect(allowPhoneLogin.value).toBe(false)
    expect(loginMethod.value).toBe('emailPassword')
  })

  it('enables phone login when policy says so', async () => {
    const { useLoginFeaturePolicy } = await import('./useLoginFeaturePolicy.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ enable_phone_login: true }),
    })
    const loginMethod = ref('emailPassword')
    const { allowPhoneLogin, loadPublicFeaturePolicy } = useLoginFeaturePolicy({
      loginMethod,
      phoneFields: { resetPhoneFields: vi.fn() },
    })
    await loadPublicFeaturePolicy()
    expect(allowPhoneLogin.value).toBe(true)
  })

  it('disables wechat login by default (missing flags, fail-closed)', async () => {
    const { useLoginFeaturePolicy } = await import('./useLoginFeaturePolicy.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    })
    const loginMethod = ref('emailPassword')
    const { allowWechatLogin, loadPublicFeaturePolicy } = useLoginFeaturePolicy({
      loginMethod,
      phoneFields: { resetPhoneFields: vi.fn() },
    })
    await loadPublicFeaturePolicy()
    expect(allowWechatLogin.value).toBe(false)
  })

  it('disables wechat login when policy fetch fails', async () => {
    const { useLoginFeaturePolicy } = await import('./useLoginFeaturePolicy.js')
    apiFetch.mockRejectedValueOnce(new Error('network'))
    const loginMethod = ref('emailPassword')
    const { allowWechatLogin, loadPublicFeaturePolicy } = useLoginFeaturePolicy({
      loginMethod,
      phoneFields: { resetPhoneFields: vi.fn() },
    })
    await loadPublicFeaturePolicy()
    expect(allowWechatLogin.value).toBe(false)
  })

  it('enables wechat login when wechat_login_available is true', async () => {
    const { useLoginFeaturePolicy } = await import('./useLoginFeaturePolicy.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ enable_wechat_login: true, wechat_login_available: true }),
    })
    const loginMethod = ref('emailPassword')
    const { allowWechatLogin, loadPublicFeaturePolicy } = useLoginFeaturePolicy({
      loginMethod,
      phoneFields: { resetPhoneFields: vi.fn() },
    })
    await loadPublicFeaturePolicy()
    expect(allowWechatLogin.value).toBe(true)
  })

  it('falls back to enable_wechat_login when availability field is missing', async () => {
    const { useLoginFeaturePolicy } = await import('./useLoginFeaturePolicy.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ enable_wechat_login: true }),
    })
    const loginMethod = ref('emailPassword')
    const { allowWechatLogin, loadPublicFeaturePolicy } = useLoginFeaturePolicy({
      loginMethod,
      phoneFields: { resetPhoneFields: vi.fn() },
    })
    await loadPublicFeaturePolicy()
    expect(allowWechatLogin.value).toBe(true)
  })

  it('keeps wechat login disabled when policy off even if availability field claims otherwise', async () => {
    const { useLoginFeaturePolicy } = await import('./useLoginFeaturePolicy.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ enable_wechat_login: false, wechat_login_available: false }),
    })
    const loginMethod = ref('emailPassword')
    const { allowWechatLogin, loadPublicFeaturePolicy } = useLoginFeaturePolicy({
      loginMethod,
      phoneFields: { resetPhoneFields: vi.fn() },
    })
    await loadPublicFeaturePolicy()
    expect(allowWechatLogin.value).toBe(false)
  })
})

}
