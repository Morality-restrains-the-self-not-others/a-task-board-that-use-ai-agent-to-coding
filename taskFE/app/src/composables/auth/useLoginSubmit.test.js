// @vitest-environment jsdom
// useLoginSubmit 回归：登录成功必须先完成凭据落盘（activate-session 的 HttpOnly
// userId+token 会话 cookie）再跳转——此前 fire-and-forget + 立即 location.href 会
// 取消在途 activate-session，Set-Cookie 未落盘 → 网关 forward-auth「无法解析登录凭据」。
if (!process.env.VITEST) {
  console.log('[skip] useLoginSubmit.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { flushPromises } = await import('@vue/test-utils')
const { ref } = await import('vue')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
  persistCredentials: vi.fn(),
  resolveRedirect: vi.fn(),
  modalAlert: vi.fn(() => Promise.resolve(true)),
  modalConfirm: vi.fn(() => Promise.reject(false)),
  routeMock: { query: {} },
}))

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
  extractErrorMessage: () => null,
}))

vi.mock('../../domain/auth/services/activate_session_service.js', () => ({
  persistLoginSuccessCredentials: (...args) => hoisted.persistCredentials(...args),
}))

vi.mock('../../utils/resolvePostLoginRedirect.js', () => ({
  resolvePostLoginRedirectUrl: (...args) => hoisted.resolveRedirect(...args),
}))

vi.mock('../../utils/modalService.js', () => ({
  default: {
    alert: (...args) => hoisted.modalAlert(...args),
    confirm: (...args) => hoisted.modalConfirm(...args),
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => hoisted.routeMock,
}))

const { useLoginSubmit } = await import('./useLoginSubmit.js')

/** 模拟无法律文档门禁的环境（privacyLoadError/licenseLoadError 置真可跳过勾选校验） */
function legalConsentMocks() {
  return {
    acceptPrivacy: ref(false),
    acceptLicense: ref(false),
    currentPrivacyPolicy: ref(null),
    currentLicenseAgreement: ref(null),
    privacyLoadError: ref(true),
    licenseLoadError: ref(true),
  }
}

function createHandleSubmit(extra = {}) {
  const { handleSubmit } = useLoginSubmit({
    emit: () => {},
    loginMethod: ref('emailPassword'),
    phoneFields: {
      isPhoneValidForPassword: ref(false),
      loginPhonePasswordForApi: ref(''),
    },
    legalConsent: legalConsentMocks(),
    accessTokenUsernameInput: ref(''),
    accessTokenInput: ref(''),
    ...extra,
  })
  return handleSubmit
}

function setupLoginPage() {
  document.body.innerHTML = `
    <input id="email" value="author@example.com">
    <input id="password" value="admin123">
    <input id="remember-me" type="checkbox" checked>
  `
  hoisted.apiFetch.mockResolvedValue({
    ok: true,
    status: 200,
    traceId: '',
    json: async () => ({
      user: {
        id: '827923618451263488',
        login_methods: [{ method_type: 'phone', is_verified: true }],
      },
      token: 'tok-abc',
    }),
  })
  hoisted.resolveRedirect.mockReturnValue('/system-admin/')
  hoisted.modalConfirm.mockRejectedValue(false)
  hoisted.persistCredentials.mockResolvedValue(undefined)
  // jsdom 不支持真实导航：stub 一个可写的 location 记录 href 赋值
  vi.stubGlobal('location', { pathname: '/auth/login/', search: '', href: '' })
}

describe('useLoginSubmit 登录后凭据落盘等待', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    hoisted.resolveRedirect.mockReturnValue('/system-admin/')
    hoisted.modalConfirm.mockRejectedValue(false)
  })

  it('登录成功等待凭据落盘完成后再跳转（regression：跳转不得取消在途 activate-session）', async () => {
    await setupLoginPage()
    let release
    hoisted.persistCredentials.mockReturnValue(new Promise((r) => { release = r }))
    const handleSubmit = createHandleSubmit()

    const submitPromise = handleSubmit({ preventDefault: () => {} })
    await flushPromises()

    // 登录响应已返回、已进入凭据落盘等待 → 此时尚未跳转
    expect(hoisted.persistCredentials).toHaveBeenCalledWith(
      expect.objectContaining({ token: 'tok-abc' }),
    )
    expect(window.location.href).toBe('')

    // 凭据落盘完成后才跳转
    release()
    await submitPromise
    expect(window.location.href).toBe('/system-admin/')
  })

  it('凭据落盘超过 5s 上限不阻塞登录跳转', async () => {
    vi.useFakeTimers()
    try {
      await setupLoginPage()
      hoisted.persistCredentials.mockReturnValue(new Promise(() => {})) // 永不 resolve
      const handleSubmit = createHandleSubmit()

      const submitPromise = handleSubmit({ preventDefault: () => {} })
      await flushPromises()
      expect(window.location.href).toBe('')

      await vi.advanceTimersByTimeAsync(5000)
      await submitPromise
      expect(window.location.href).toBe('/system-admin/')
    } finally {
      vi.useRealTimers()
    }
  })
})

// OPT-20260824-001: 管理员/客户登录入口分离 —— adminLogin=true 提交到
// /api/auth/admin-login/（后端仅放行管理员账号）；客户表单保持 /api/auth/；
// access-token 方式不受影响。
describe('useLoginSubmit 管理员/客户登录入口分离', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    hoisted.resolveRedirect.mockReturnValue('/system-admin/')
    hoisted.modalConfirm.mockRejectedValue(false)
  })

  async function submitAndGetEndpoint(extra = {}) {
    await setupLoginPage()
    const handleSubmit = createHandleSubmit(extra)
    await handleSubmit({ preventDefault: () => {} })
    await flushPromises()
    return hoisted.apiFetch.mock.calls[0][0]
  }

  it('管理员入口（adminLogin=true）提交到 /api/auth/admin-login/', async () => {
    const endpoint = await submitAndGetEndpoint({ adminLogin: true })
    expect(endpoint).toBe('/api/auth/admin-login/')
  })

  it('客户登录入口（默认）仍提交到 /api/auth/', async () => {
    const endpoint = await submitAndGetEndpoint()
    expect(endpoint).toBe('/api/auth/')
  })

  it('未验证手机跳转到资料页绑定深链', async () => {
    await setupLoginPage()
    hoisted.persistCredentials.mockResolvedValue(undefined)
    hoisted.apiFetch.mockResolvedValue({
      ok: true,
      status: 200,
      traceId: '',
      json: async () => ({ user: { id: '827923618451263488' }, token: 'tok-abc' }),
    })
    hoisted.modalAlert.mockResolvedValue(true)
    const handleSubmit = createHandleSubmit()
    await handleSubmit({ preventDefault: () => {} })
    await flushPromises()
    expect(hoisted.modalAlert).toHaveBeenCalled()
    expect(hoisted.modalConfirm).not.toHaveBeenCalled()
    expect(window.location.href).toBe('/profile/#rg=profile.phone_binding')
  })

  it('未验证且关闭提示仍去资料绑定（不可跳过）', async () => {
    await setupLoginPage()
    hoisted.persistCredentials.mockResolvedValue(undefined)
    hoisted.apiFetch.mockResolvedValue({
      ok: true,
      status: 200,
      traceId: '',
      json: async () => ({ user: { id: '827923618451263488' }, token: 'tok-abc' }),
    })
    hoisted.modalAlert.mockRejectedValue(false)
    const handleSubmit = createHandleSubmit()
    await handleSubmit({ preventDefault: () => {} })
    await flushPromises()
    expect(window.location.href).toBe('/profile/#rg=profile.phone_binding')
  })

  it('已验证手机不弹窗，直达原落点', async () => {
    await setupLoginPage()
    hoisted.persistCredentials.mockResolvedValue(undefined)
    const handleSubmit = createHandleSubmit()
    await handleSubmit({ preventDefault: () => {} })
    await flushPromises()
    expect(hoisted.modalAlert).not.toHaveBeenCalled()
    expect(window.location.href).toBe('/system-admin/')
  })

  it('管理员入口不弹手机验证引导', async () => {
    await setupLoginPage()
    hoisted.persistCredentials.mockResolvedValue(undefined)
    const handleSubmit = createHandleSubmit({ adminLogin: true })
    await handleSubmit({ preventDefault: () => {} })
    await flushPromises()
    expect(hoisted.modalAlert).not.toHaveBeenCalled()
    expect(window.location.href).toBe('/system-admin/')
  })

  it('客户 access-token 方式（无 adminLogin）仍提交到 login-with-access-token', async () => {
    await setupLoginPage()
    document.body.innerHTML = `
      <input id="email" value="api-user@example.com">
      <input id="password" value="x">
      <input id="remember-me" type="checkbox" checked>
    `
    const handleSubmit = createHandleSubmit({
      loginMethod: ref('accessToken'),
      accessTokenUsernameInput: ref('api-user@example.com'),
      accessTokenInput: ref('tok-xyz'),
    })
    await handleSubmit({ preventDefault: () => {} })
    await flushPromises()
    expect(hoisted.apiFetch.mock.calls[0][0]).toBe('/api/accounts/users/login-with-access-token/')
  })
})
}
