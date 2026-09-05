// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserReferral.accessCode.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { nextTick } = await import('vue')
const { mount, flushPromises } = await import('@vue/test-utils')

const USER_ID = '873093522473906176'
const ACCESS_CODE = 'kN3pQ8xWm2'

const hoisted = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  getCookieMock: vi.fn(() => ''),
  getStoredUserIdMock: vi.fn(() => ''),
  resolveAuthenticatedUserIdMock: vi.fn(async () => ''),
  routeMock: {
    params: { tenant: '' },
    query: {},
    path: '/profile/referral/',
  },
  routerMock: { replace: vi.fn(() => Promise.resolve()) },
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetchMock(...args),
}))

vi.mock('../utils/cookieUtils.js', () => ({
  getCookie: (...args) => hoisted.getCookieMock(...args),
}))

vi.mock('../utils/sessionUserIdUtils.js', () => ({
  getStoredUserId: () => hoisted.getStoredUserIdMock(),
  resolveAuthenticatedUserId: () => hoisted.resolveAuthenticatedUserIdMock(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => hoisted.routeMock,
  useRouter: () => hoisted.routerMock,
}))

function jsonOk(body) {
  return Promise.resolve({
    ok: true,
    status: 200,
    json: async () => body,
  })
}

function stubApis({ status = {}, stats = {} } = {}) {
  hoisted.apiFetchMock.mockImplementation((url) => {
    if (String(url).includes('/referral-codes/status/')) {
      return jsonOk({
        application_status: null,
        has_active_code: false,
        ...status,
      })
    }
    if (String(url).includes('/referral/stats/')) {
      return jsonOk({
        referral_count: 0,
        monthly_earnings: [],
        ...stats,
      })
    }
    return jsonOk({})
  })
}

async function mountReferral() {
  const { default: UserReferral } = await import('./UserReferral.vue')
  const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
  await flushPromises()
  await nextTick()
  return wrapper
}

function shareLinkText(wrapper) {
  const el = wrapper.find('[data-testid="referral-share-link"]')
  if (el.exists()) return el.text()
  return wrapper.find('p.text-sm.text-text.break-all').text()
}

describe('UserReferral accessCode without qualification', () => {
  beforeEach(() => {
    hoisted.apiFetchMock.mockReset()
    hoisted.getCookieMock.mockReset()
    hoisted.getCookieMock.mockReturnValue('')
    hoisted.getStoredUserIdMock.mockReset()
    hoisted.getStoredUserIdMock.mockReturnValue('')
    hoisted.resolveAuthenticatedUserIdMock.mockReset()
    hoisted.resolveAuthenticatedUserIdMock.mockResolvedValue('')
    hoisted.routeMock.query = {}
    hoisted.routerMock.replace.mockReset()
  })

  it('shows opaque accessCode from status API when user has no referral qualification', async () => {
    stubApis({
      status: {
        has_active_code: false,
        application_status: null,
        access_code: ACCESS_CODE,
      },
    })

    const wrapper = await mountReferral()

    expect(wrapper.text()).toContain('暂无推荐资格')
    const noQual = wrapper.get('[data-testid="referral-no-qualification-notice"]')
    expect(noQual.text()).toContain('后续下单')
    expect(noQual.text()).not.toContain('无法获得收益分成')
    expect(shareLinkText(wrapper)).toContain(`accessCode=${ACCESS_CODE}`)
    expect(wrapper.find('[data-testid="referral-code"]').text()).toBe(ACCESS_CODE)
    expect(shareLinkText(wrapper)).not.toContain(USER_ID)
  })

  it('does not derive accessCode from userId when status omits it', async () => {
    hoisted.getStoredUserIdMock.mockReturnValue(USER_ID)
    hoisted.resolveAuthenticatedUserIdMock.mockResolvedValue(USER_ID)
    stubApis({ status: { has_active_code: false, application_status: null } })

    const wrapper = await mountReferral()
    const link = shareLinkText(wrapper)

    expect(link).not.toContain(USER_ID)
    expect(link).not.toContain(`u${USER_ID}`)
    expect(link).not.toContain('accessCode=')
    expect(wrapper.find('[data-testid="referral-code"]').text()).toBe('')
  })

  it('does not use referral_code as share code when access_code is missing', async () => {
    hoisted.getStoredUserIdMock.mockReturnValue(USER_ID)
    stubApis({
      status: {
        has_active_code: false,
        referral_code: `u${USER_ID}`,
      },
    })

    const wrapper = await mountReferral()

    expect(shareLinkText(wrapper)).not.toContain(USER_ID)
    expect(shareLinkText(wrapper)).not.toContain('accessCode=')
  })

  it('shows referral_rate_display from status API instead of a hardcoded 5%', async () => {
    stubApis({
      status: {
        has_active_code: false,
        application_status: null,
        access_code: ACCESS_CODE,
        referral_rate_display: '12%',
      },
    })
    const wrapper = await mountReferral()
    expect(wrapper.get('[data-testid="referral-rate-apply-copy"]').text()).toBe('12%')
    expect(wrapper.text()).not.toMatch(/带来\s*5%/)
  })

  it('does not substitute 5% when status omits referral_rate_display', async () => {
    stubApis({
      status: {
        has_active_code: false,
        application_status: null,
        access_code: ACCESS_CODE,
      },
    })
    const wrapper = await mountReferral()
    expect(wrapper.get('[data-testid="referral-rate-apply-copy"]').text()).not.toBe('5%')
    expect(wrapper.get('[data-testid="referral-rate-apply-copy"]').text()).toBe('—')
  })

  it('shows wechat receiver registered hint when qualification is active', async () => {
    stubApis({
      status: {
        has_active_code: true,
        application_status: 'approved',
        access_code: ACCESS_CODE,
        wechat_receiver_status: 'registered',
      },
    })
    const wrapper = await mountReferral()
    expect(wrapper.get('[data-testid="wechat-receiver-registered"]').text()).toContain('已在微信分账后台登记')
  })

  it('expiry warning refers to later orders, not only new registrations', async () => {
    stubApis({
      status: {
        has_active_code: true,
        application_status: 'approved',
        access_code: ACCESS_CODE,
        expires_in_days: 10,
      },
    })
    const wrapper = await mountReferral()
    const notice = wrapper.get('[data-testid="referral-expiry-notice"]')
    expect(notice.text()).toContain('下单')
    expect(notice.text()).not.toContain('新注册用户')
  })

  it('shows bind-wechat hint when receiver status is pending_openid', async () => {
    stubApis({
      status: {
        has_active_code: true,
        application_status: 'approved',
        access_code: ACCESS_CODE,
        wechat_receiver_status: 'pending_openid',
      },
    })
    const wrapper = await mountReferral()
    expect(wrapper.get('[data-testid="wechat-receiver-pending-openid"]').text()).toContain('尚未绑定微信登录')
  })
})

describe('UserReferral URL accessCode normalization (OPT-20260824-005)', () => {
  beforeEach(() => {
    hoisted.apiFetchMock.mockReset()
    hoisted.getStoredUserIdMock.mockReset()
    hoisted.getStoredUserIdMock.mockReturnValue('')
    hoisted.resolveAuthenticatedUserIdMock.mockReset()
    hoisted.resolveAuthenticatedUserIdMock.mockResolvedValue('')
    // 组件复用共享工具 normalizeUrlAccessCodeToOwn（main.js 全局 afterEach 同源），
    // 其基于 window.location 与存储归一化，而非 router.replace
    window.history.replaceState({}, '', '/profile/referral/')
    localStorage.removeItem('referralCode')
    sessionStorage.removeItem('referral_own_access_code')
  })

  it('rewrites URL accessCode to own code when it differs', async () => {
    // 场景：地址栏残留他人/已删除溯源码（如 w25brikFiT），页面码为自己的码
    window.history.replaceState({}, '', '/profile/referral/?accessCode=w25brikFiT')
    stubApis({ status: { has_active_code: false, access_code: ACCESS_CODE } })

    await mountReferral()

    expect(window.location.search).toBe(`?accessCode=${ACCESS_CODE}`)
  })

  it('keeps other URL params when normalizing accessCode', async () => {
    window.history.replaceState({}, '', '/profile/referral/?accessCode=w25brikFiT&utm_source=e2e')
    stubApis({ status: { has_active_code: false, access_code: ACCESS_CODE } })

    await mountReferral()

    const params = new URLSearchParams(window.location.search)
    expect(params.get('accessCode')).toBe(ACCESS_CODE)
    expect(params.get('utm_source')).toBe('e2e')
  })

  it('does not rewrite when URL accessCode already equals own code', async () => {
    window.history.replaceState({}, '', `/profile/referral/?accessCode=${ACCESS_CODE}`)
    stubApis({ status: { has_active_code: false, access_code: ACCESS_CODE } })

    await mountReferral()

    expect(window.location.search).toBe(`?accessCode=${ACCESS_CODE}`)
  })

  it('does not touch URL when no accessCode param present', async () => {
    stubApis({ status: { has_active_code: false, access_code: ACCESS_CODE } })

    await mountReferral()

    expect(window.location.search).toBe('')
  })

  it('removes URL accessCode when user has no own code', async () => {
    window.history.replaceState({}, '', '/profile/referral/?accessCode=w25brikFiT')
    stubApis({ status: { has_active_code: false, application_status: null } })

    await mountReferral()

    expect(window.location.search).toBe('')
  })
})

}
