// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useAllowedCountryCodes.test.js requires vitest runtime')
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

describe('useAllowedCountryCodes defaultCountryPrefix (OPT-20260806-027)', () => {
  beforeEach(() => {
    apiFetch.mockReset()
    // 模块级缓存是跨用例共享的，每次重置以便独立断言
    vi.resetModules()
  })

  it('keeps +86 default when options include +86', async () => {
    const { useAllowedCountryCodes } = await import('./useAllowedCountryCodes.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { phone_country_options: [{ code: '+86', name: '中国' }, { code: '+852', name: '中国香港' }] } }),
    })
    const { fetchAllowedCodes, defaultCountryPrefix } = useAllowedCountryCodes()
    await fetchAllowedCodes()
    expect(defaultCountryPrefix.value).toBe('+86')
  })

  it('falls back to first option when whitelist excludes +86', async () => {
    const { useAllowedCountryCodes } = await import('./useAllowedCountryCodes.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { phone_country_options: [{ code: '+852', name: '中国香港' }, { code: '+853', name: '中国澳门' }] } }),
    })
    const { fetchAllowedCodes, defaultCountryPrefix } = useAllowedCountryCodes()
    await fetchAllowedCodes()
    expect(defaultCountryPrefix.value).toBe('+852')
  })

  it('syncPrefixWithAllowedOptions corrects a stale prefix ref to +852', async () => {
    const { useAllowedCountryCodes } = await import('./useAllowedCountryCodes.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { phone_country_options: [{ code: '+852', name: '中国香港' }] } }),
    })
    const { fetchAllowedCodes, syncPrefixWithAllowedOptions } = useAllowedCountryCodes()
    const prefix = ref('+86') // 消费者初始硬编码 +86
    syncPrefixWithAllowedOptions(prefix)
    await fetchAllowedCodes()
    // watch immediate + options 更新后应校正为白名单首项
    await new Promise((r) => setTimeout(r, 0))
    expect(prefix.value).toBe('+852')
  })

  it('keeps prefix unchanged when options still include it', async () => {
    const { useAllowedCountryCodes } = await import('./useAllowedCountryCodes.js')
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { phone_country_options: [{ code: '+86', name: '中国' }, { code: '+852', name: '中国香港' }] } }),
    })
    const { fetchAllowedCodes, syncPrefixWithAllowedOptions } = useAllowedCountryCodes()
    const prefix = ref('+86')
    syncPrefixWithAllowedOptions(prefix)
    await fetchAllowedCodes()
    await new Promise((r) => setTimeout(r, 0))
    expect(prefix.value).toBe('+86')
  })

  it('does not touch prefix when API fails (fallback full list)', async () => {
    const { useAllowedCountryCodes } = await import('./useAllowedCountryCodes.js')
    apiFetch.mockRejectedValueOnce(new Error('network'))
    const { fetchAllowedCodes, syncPrefixWithAllowedOptions } = useAllowedCountryCodes()
    const prefix = ref('+86')
    syncPrefixWithAllowedOptions(prefix)
    await fetchAllowedCodes()
    await new Promise((r) => setTimeout(r, 0))
    expect(prefix.value).toBe('+86')
  })
})

}
