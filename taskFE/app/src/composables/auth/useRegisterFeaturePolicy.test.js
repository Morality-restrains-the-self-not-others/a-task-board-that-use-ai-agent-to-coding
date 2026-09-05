// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] useRegisterFeaturePolicy.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

const apiFetch = vi.hoisted(() => vi.fn())

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: (...args) => apiFetch(...args),
}))

  const { ref } = await import('vue')

describe('useRegisterFeaturePolicy', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('disables email register when public policy fetch fails', async () => {
    const { useRegisterFeaturePolicy } = await import('./useRegisterFeaturePolicy.js')
    apiFetch.mockRejectedValueOnce(new Error('network'))
    const currentRegisterType = ref('email')
    const { allowEmailRegister, loadPublicFeaturePolicy } = useRegisterFeaturePolicy({
      currentRegisterType,
    })
    await loadPublicFeaturePolicy()
    expect(allowEmailRegister.value).toBe(false)
    expect(currentRegisterType.value).toBe('phone')
  })

  it('keeps email register when policy enables it', async () => {
    const { useRegisterFeaturePolicy } = await import('./useRegisterFeaturePolicy.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { enable_email_register: true } }),
    })
    const currentRegisterType = ref('email')
    const { allowEmailRegister, loadPublicFeaturePolicy } = useRegisterFeaturePolicy({
      currentRegisterType,
    })
    await loadPublicFeaturePolicy()
    expect(allowEmailRegister.value).toBe(true)
    expect(currentRegisterType.value).toBe('email')
  })

  it('switches to phone when policy disables email register', async () => {
    const { useRegisterFeaturePolicy } = await import('./useRegisterFeaturePolicy.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { enable_email_register: false } }),
    })
    const currentRegisterType = ref('email')
    const { allowEmailRegister, loadPublicFeaturePolicy } = useRegisterFeaturePolicy({
      currentRegisterType,
    })
    await loadPublicFeaturePolicy()
    expect(allowEmailRegister.value).toBe(false)
    expect(currentRegisterType.value).toBe('phone')
  })
})
}
