// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserReferral.traceId.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { nextTick } = await import('vue')
const { mount, flushPromises } = await import('@vue/test-utils')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  routeMock: {
    params: { tenant: '' },
    query: {},
    path: '/user/873093522473906176/profile/referral/',
  },
  routerMock: { replace: vi.fn(() => Promise.resolve()) },
}))
const apiFetchMock = hoisted.apiFetchMock
const routeMock = hoisted.routeMock

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetchMock(...args),
}))

vi.mock('../utils/cookieUtils.js', () => ({
  getCookie: (name) => (name === 'userId' ? '873093522473906176' : ''),
}))

vi.mock('vue-router', () => ({
  useRoute: () => hoisted.routeMock,
  useRouter: () => hoisted.routerMock,
}))

// 返回状态接口成功的最小响应；stats 接口按用例覆盖
function statusOk() {
  return Promise.resolve({
    ok: true,
    status: 200,
    json: async () => ({ application_status: null, has_active_code: false }),
  })
}

describe('UserReferral stats error data-traceId', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
    routeMock.query = {}
    hoisted.routerMock.replace.mockReset()
  })

  it('binds response.traceId on HTTP error (non-ok branch)', async () => {
    apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/referral/stats/')) {
        return Promise.resolve({
          ok: false,
          status: 500,
          traceId: 'tid-referral-500',
          json: async () => ({ detail: 'server error' }),
        })
      }
      return statusOk()
    })

    const { default: UserReferral } = await import('./UserReferral.vue')
    const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
    await flushPromises()
    await nextTick()

    const errEl = wrapper.find('p.text-red-500.mb-4')
    expect(errEl.exists()).toBe(true)
    expect(errEl.text()).toContain('获取推荐收益统计失败')
    expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe(
      'tid-referral-500',
    )
  })

  it('binds error.traceId on network failure (catch branch)', async () => {
    const netErr = new Error('network down')
    netErr.traceId = 'tid-referral-net'
    apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/referral/stats/')) {
        return Promise.reject(netErr)
      }
      return statusOk()
    })

    const { default: UserReferral } = await import('./UserReferral.vue')
    const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
    await flushPromises()
    await nextTick()

    const errEl = wrapper.find('p.text-red-500.mb-4')
    expect(errEl.exists()).toBe(true)
    expect(errEl.text()).toContain('获取推荐收益统计失败')
    expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe(
      'tid-referral-net',
    )
  })
})

}
