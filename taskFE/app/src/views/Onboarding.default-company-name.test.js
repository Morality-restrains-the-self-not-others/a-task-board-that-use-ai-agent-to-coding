// @vitest-environment jsdom
/**
 * Onboarding 默认公司名预填 — 「{user}的公司」
 */
if (!process.env.VITEST) {
  console.log('[skip] Onboarding.default-company-name.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')
const { default: Onboarding } = await import('./Onboarding.vue')

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

function mockApis({ nickname = '软刀', companies = [] } = {}) {
  apiFetchMock.mockImplementation(async (url) => {
    if (url === '/api/auth/user-roles/') {
      return { ok: true, status: 200, json: async () => ({ roles: [] }) }
    }
    if (url === '/api/accounts/users/profile/') {
      return {
        ok: true,
        status: 200,
        json: async () => ({ user_id: 'user-123', personal_nickname: nickname }),
      }
    }
    if (url.startsWith('/api/tenant/_/accounts/companies/by-creator')) {
      return { ok: true, status: 200, json: async () => companies }
    }
    return { ok: false, status: 500, json: async () => ({}) }
  })
}

describe('Onboarding 默认公司名预填', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, 'location', {
      value: {
        href: 'http://localhost:4000/onboarding/',
        pathname: '/onboarding/',
        search: '',
      },
      writable: true,
      configurable: true,
    })
    getCookieMock.mockReturnValue('user-123')
  })

  it('无已有公司时预填「{昵称}的公司」', async () => {
    mockApis({ nickname: '软刀', companies: [] })
    const wrapper = mount(Onboarding)
    await flushPromises()
    expect(wrapper.find('#companyName').element.value).toBe('软刀的公司')
  })

  it('已有自动生成名「我的公司」时改为「{昵称}的公司」', async () => {
    mockApis({
      nickname: '软刀',
      companies: [{ id: 'c1', name: '我的公司' }],
    })
    const wrapper = mount(Onboarding)
    await flushPromises()
    expect(wrapper.find('#companyName').element.value).toBe('软刀的公司')
  })

  it('已有自定义公司名时保持原名', async () => {
    mockApis({
      nickname: '软刀',
      companies: [{ id: 'c1', name: '现有公司' }],
    })
    const wrapper = mount(Onboarding)
    await flushPromises()
    expect(wrapper.find('#companyName').element.value).toBe('现有公司')
  })

  it('无展示名时回退「我的公司」', async () => {
    mockApis({ nickname: '', companies: [] })
    const wrapper = mount(Onboarding)
    await flushPromises()
    expect(wrapper.find('#companyName').element.value).toBe('我的公司')
  })
})
}
