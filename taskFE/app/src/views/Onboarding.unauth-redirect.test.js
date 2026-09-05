// @vitest-environment jsdom
/**
 * Onboarding 未登录重定向守卫 — 单元测试
 *
 * 目标：未登录（无有效会话）访问 /onboarding/ 必须跳转登录页，
 * 不得停留在「创建公司」表单（此前 roles 401 被静默忽略）。
 *
 * 覆盖：
 * 1. roles API 返回 401 → 重定向 /auth/login/（不带 next=/onboarding/，
 *    避免该 next 被扫码回调无条件回显 → 有公司用户二次登录仍回引导页）
 * 2. roles API 200 且无平台角色 → 留在引导页（行为不变）
 * 3. roles API 200 且平台角色 super_admin → 跳 /system-admin/（行为不变）
 */
if (!process.env.VITEST) {
  console.log('[skip] Onboarding.unauth-redirect.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')
const { default: Onboarding } = await import('./Onboarding.vue')

// vi.mock 工厂中引用的变量必须通过 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const { apiFetchMock, getCookieMock, setUserCompaniesMock } = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  getCookieMock: vi.fn(),
  setUserCompaniesMock: vi.fn(),
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => apiFetchMock(...args),
}))

vi.mock('../utils/cookieUtils.js', () => ({
  getCookie: (...args) => getCookieMock(...args),
}))

vi.mock('../utils/sharedUserTenantCache.js', () => ({
  setUserCompanies: (...args) => setUserCompaniesMock(...args),
}))

vi.mock('../utils/traceId.js', () => ({
  extractTraceId: () => '',
}))

describe('Onboarding 未登录重定向守卫', () => {
  let locationMock

  beforeEach(() => {
    vi.clearAllMocks()
    locationMock = {
      href: 'http://localhost:4000/onboarding/',
      pathname: '/onboarding/',
      search: '',
    }
    Object.defineProperty(window, 'location', {
      value: locationMock,
      writable: true,
      configurable: true,
    })
    getCookieMock.mockReturnValue('')
  })

  it('roles API 返回 401（未登录）→ 重定向到登录页，不带 next=/onboarding/ 回跳', async () => {
    apiFetchMock.mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ detail: '请先登录' }),
    })

    mount(Onboarding)
    await flushPromises()

    // 不带 next=/onboarding/：该 next 会被微信扫码回调无条件回显，令「已设置公司名的
    // 用户第二次扫码登录依旧跳到公司名称设置页」（登录后落点由后端按角色计算）。
    expect(window.location.href).toBe('/auth/login/')
    // 重定向后不得继续查询已有公司（避免多余请求/停留表单）
    expect(apiFetchMock).toHaveBeenCalledTimes(1)
    expect(apiFetchMock).toHaveBeenCalledWith(
      '/api/auth/user-roles/',
      expect.objectContaining({ credentials: 'include' }),
    )
  })

  it('已登录且无平台角色 → 留在引导页，不跳转', async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ roles: [] }),
    })

    mount(Onboarding)
    await flushPromises()

    expect(window.location.href).toBe('http://localhost:4000/onboarding/')
  })

  it('已登录且平台角色 super_admin → 跳系统管理（回归保护）', async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ roles: [{ role: 'super_admin' }] }),
    })

    mount(Onboarding)
    await flushPromises()

    expect(window.location.href).toBe('/system-admin/')
  })
})

describe('Onboarding 提交 401 恢复跳转（凭据链兜底）', () => {
  let locationMock

  beforeEach(() => {
    vi.clearAllMocks()
    locationMock = {
      href: 'http://localhost:4000/onboarding/',
      pathname: '/onboarding/',
      search: '',
    }
    Object.defineProperty(window, 'location', {
      value: locationMock,
      writable: true,
      configurable: true,
    })
    getCookieMock.mockReturnValue('user-123')
    // mounted 的 roles 检查返回已登录空角色
    apiFetchMock.mockImplementation(async (url) => {
      if (url === '/api/auth/user-roles/') {
        return { ok: true, status: 200, json: async () => ({ roles: [] }) }
      }
      if (url.startsWith('/api/tenant/_/accounts/companies/by-creator')) {
        return { ok: true, status: 200, json: async () => [] }
      }
      return { ok: false, status: 401, json: async () => ({}) }
    })
  })

  it('创建公司提交返回 401（凭据失效）→ 跳登录页，不带 next=/onboarding/（避免有公司用户扫码后仍回引导页）', async () => {
    const wrapper = mount(Onboarding)
    await flushPromises()
    await wrapper.find('#companyName').setValue('我的公司')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(window.location.href).toBe('/auth/login/')
  })

  it('改名提交返回 401 → 同样跳登录页，不带 next', async () => {
    apiFetchMock.mockImplementation(async (url) => {
      if (url === '/api/auth/user-roles/') {
        return { ok: true, status: 200, json: async () => ({ roles: [] }) }
      }
      if (url.startsWith('/api/tenant/_/accounts/companies/by-creator')) {
        return { ok: true, status: 200, json: async () => [{ id: 'c1', name: '现有公司' }] }
      }
      return { ok: false, status: 401, json: async () => ({}) }
    })

    const wrapper = mount(Onboarding)
    await flushPromises()
    expect(wrapper.find('#companyName').element.value).toBe('现有公司')
    await wrapper.find('#companyName').setValue('新名字')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(window.location.href).toBe('/auth/login/')
  })

  it('改名首次 403（PDP 缓存陈旧窗口）→ 自动重试 1 次成功后进入工作台', async () => {
    vi.useFakeTimers()
    try {
      const patchCalls = []
      apiFetchMock.mockImplementation(async (url) => {
        if (url === '/api/auth/user-roles/') {
          return { ok: true, status: 200, json: async () => ({ roles: [] }) }
        }
        if (url.startsWith('/api/tenant/_/accounts/companies/by-creator')) {
          return { ok: true, status: 200, json: async () => [{ id: 'c1', name: '现有公司' }] }
        }
        if (url === '/api/tenant/_/accounts/companies/') {
          patchCalls.push(url)
          // 首次 403，重试成功
          if (patchCalls.length === 1) {
            return { ok: false, status: 403, json: async () => ({ detail: '无权限' }) }
          }
          return { ok: true, status: 200, json: async () => ({ company_id: 'c1' }) }
        }
        return { ok: false, status: 500, json: async () => ({}) }
      })

      const wrapper = mount(Onboarding)
      await flushPromises()
      await wrapper.find('#companyName').setValue('新名字')
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      // 重试间隔 ~1s
      await vi.advanceTimersByTimeAsync(1000)
      await flushPromises()

      expect(patchCalls).toHaveLength(2)
      expect(window.location.href).toBe('/tenant/c1/work-panel/')
    } finally {
      vi.useRealTimers()
    }
  })

  it('改名重试后仍 403 → 提示权限尚未同步而非死胡同', async () => {
    const patchCalls = []
    apiFetchMock.mockImplementation(async (url) => {
      if (url === '/api/auth/user-roles/') {
        return { ok: true, status: 200, json: async () => ({ roles: [] }) }
      }
      if (url.startsWith('/api/tenant/_/accounts/companies/by-creator')) {
        return { ok: true, status: 200, json: async () => [{ id: 'c1', name: '现有公司' }] }
      }
      if (url === '/api/tenant/_/accounts/companies/') {
        patchCalls.push(url)
        return { ok: false, status: 403, json: async () => ({ detail: '无权限' }) }
      }
      return { ok: false, status: 500, json: async () => ({}) }
    })

    const wrapper = mount(Onboarding)
    await flushPromises()
    await wrapper.find('#companyName').setValue('新名字')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await new Promise((resolve) => setTimeout(resolve, 1050))
    await flushPromises()

    expect(patchCalls).toHaveLength(2)
    expect(wrapper.text()).toContain('权限尚未同步')
    expect(window.location.href).toBe('http://localhost:4000/onboarding/')
  })
})

}
