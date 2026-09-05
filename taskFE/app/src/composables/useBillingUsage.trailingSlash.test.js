// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBillingUsage.trailingSlash.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

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
  parseCompanyMembersResponse: () => ({ members: [] }),
}))

const { useBillingUsage } = await import('./useBillingUsage.js')

function jsonResponse(body, { ok = true, status = 200 } = {}) {
  return { ok, status, json: async () => body }
}

describe('useBillingUsage billing URL 尾部斜杠（回归：not found 事故）', () => {
  beforeEach(() => {
    apiFetch.mockReset()
    // useBillingUsage.js 显式 import apiFetch（OPT-20260807-039），经模块 mock 拦截。
  })

  it('onMounted 加载计费单元类型请求 billing/units/ 带尾部斜杠', async () => {
    const urls = []
    apiFetch.mockImplementation(async (url) => {
      urls.push(String(url))
      if (String(url).includes('/billing/units/')) return jsonResponse([])
      return jsonResponse({ results: [] })
    })

    const wrapper = mount({ setup() { useBillingUsage(); return () => null } })
    await flushPromises()

    const unitsUrl = urls.find((u) => u.includes('/billing/units'))
    expect(unitsUrl).toBe('/api/tenant/850256677331562496/billing/units/')

    wrapper.unmount()
  })

  it('onMounted 加载用量记录请求 billing/usages/ 带尾部斜杠（含查询串）', async () => {
    const urls = []
    apiFetch.mockImplementation(async (url) => {
      urls.push(String(url))
      if (String(url).includes('/billing/units/')) return jsonResponse([])
      return jsonResponse({ results: [] })
    })

    const wrapper = mount({ setup() { useBillingUsage(); return () => null } })
    await flushPromises()

    const usageUrl = urls.find((u) => u.includes('/billing/usages'))
    expect(usageUrl).toBeTruthy()
    expect(usageUrl.startsWith('/api/tenant/850256677331562496/billing/usages/')).toBe(true)
    expect(usageUrl).not.toContain('/billing/usages?')

    wrapper.unmount()
  })

  it('applyFilters 触发的 usages 请求同样带尾部斜杠', async () => {
    apiFetch.mockImplementation(async (url) => {
      if (String(url).includes('/billing/units/')) return jsonResponse([])
      return jsonResponse({ results: [] })
    })

    let c
    const wrapper = mount({ setup() { c = useBillingUsage(); return () => null } })
    await flushPromises()
    apiFetch.mockClear()

    c.applyFilters()
    await flushPromises()

    const usageUrl = apiFetch.mock.calls.map(([u]) => String(u)).find((u) => u.includes('/billing/usages'))
    expect(usageUrl).toBeTruthy()
    expect(usageUrl.startsWith('/api/tenant/850256677331562496/billing/usages/')).toBe(true)
    expect(usageUrl).not.toContain('/billing/usages?')

    wrapper.unmount()
  })
})

}
