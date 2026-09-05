// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useImpersonateUser.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
  persist: vi.fn(async () => {}),
  getActiveToken: vi.fn(async () => 'admin-token'),
  getStoredUserId: vi.fn(() => 'bootstrap-admin'),
  storeUserId: vi.fn(),
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))
vi.mock('../domain/auth/services/activate_session_service.js', () => ({
  persistLoginSuccessCredentials: (...args) => hoisted.persist(...args),
}))
vi.mock('../domain/auth/services/saved_accounts_store.js', () => ({
  getActiveToken: (...args) => hoisted.getActiveToken(...args),
}))
vi.mock('../utils/sessionUserIdUtils.js', () => ({
  getStoredUserId: () => hoisted.getStoredUserId(),
  storeUserId: (...args) => hoisted.storeUserId(...args),
}))

const { useImpersonateUser, IMPERSONATOR_BACKUP_KEY, NESTED_IMPERSONATION_DETAIL } = await import('./useImpersonateUser.js')

function jsonResponse(body, { ok = true, status = 200, headers = {} } = {}) {
  return {
    ok,
    status,
    headers: {
      get: (k) => headers[k] || headers[String(k).toLowerCase()] || null,
    },
    json: async () => body,
  }
}

describe('useImpersonateUser', () => {
  let store
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
    hoisted.persist.mockClear()
    store = {
      data: {},
      setItem(k, v) { this.data[k] = v },
      getItem(k) { return this.data[k] || null },
    }
  })

  it('posts with Idempotency-Key and redirects on success', async () => {
    const assignHref = vi.fn()
    hoisted.apiFetch.mockResolvedValue(jsonResponse({
      token: 'imp_abc',
      user: { id: 'u2', username: 'bob' },
      redirect_url: '/onboarding/',
    }))
    const c = useImpersonateUser({
      apiFetch: hoisted.apiFetch,
      persist: hoisted.persist,
      assignHref,
      sessionStore: store,
    })
    const result = await c.impersonateUser('u2', '排查线上工单问题')
    expect(result.ok).toBe(true)
    expect(hoisted.apiFetch).toHaveBeenCalledTimes(1)
    const [, opts] = hoisted.apiFetch.mock.calls[0]
    expect(opts.method).toBe('POST')
    expect(opts.headers['Idempotency-Key']).toBeTruthy()
    expect(JSON.parse(opts.body).reason).toBe('排查线上工单问题')
    expect(hoisted.persist).toHaveBeenCalled()
    expect(assignHref).toHaveBeenCalledWith('/onboarding/')
    expect(JSON.parse(store.getItem(IMPERSONATOR_BACKUP_KEY)).userId).toBe('bootstrap-admin')
  })

  it('sets error and data-traceId source on 403', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(
      { detail: '权限不足，需要平台权限码: user:impersonate', trace_id: 'tr-imp-1' },
      { ok: false, status: 403, headers: { 'X-Trace-Id': 'tr-imp-1' } },
    ))
    const c = useImpersonateUser({
      apiFetch: hoisted.apiFetch,
      persist: hoisted.persist,
      assignHref: vi.fn(),
      sessionStore: store,
    })
    const result = await c.impersonateUser('u2', '排查线上工单问题')
    expect(result.ok).toBe(false)
    expect(c.impersonateError.value).toContain('权限不足')
    expect(c.impersonateErrorTraceId.value).toBe('tr-imp-1')
    expect(hoisted.persist).not.toHaveBeenCalled()
  })

  it('skips in-flight double click', async () => {
    const deferred = {}
    deferred.promise = new Promise((resolve) => { deferred.resolve = resolve })
    hoisted.apiFetch.mockReturnValue(deferred.promise)
    const c = useImpersonateUser({
      apiFetch: hoisted.apiFetch,
      persist: hoisted.persist,
      assignHref: vi.fn(),
      sessionStore: store,
    })
    const p1 = c.impersonateUser('u2', '排查线上工单问题')
    await vi.waitFor(() => expect(hoisted.apiFetch).toHaveBeenCalled())
    const p2 = await c.impersonateUser('u2', '排查线上工单问题')
    expect(p2.skipped).toBe(true)
    deferred.resolve(jsonResponse({ token: 'imp_x', user: { id: 'u2' }, redirect_url: '/' }))
    await p1
    expect(hoisted.apiFetch).toHaveBeenCalledTimes(1)
  })

  it('redirects when already impersonating the same target', async () => {
    const assignHref = vi.fn()
    hoisted.apiFetch
      .mockResolvedValueOnce(jsonResponse(
        { detail: 'already impersonating', trace_id: 'tr-imp-nested' },
        { ok: false, status: 409, headers: { 'X-Trace-Id': 'tr-imp-nested' } },
      ))
      .mockResolvedValueOnce(jsonResponse({
        impersonating: true,
        target_user_id: 'u2',
        redirect_url: '/tenant/c1/work-panel/',
      }))
    const c = useImpersonateUser({
      apiFetch: hoisted.apiFetch,
      persist: hoisted.persist,
      assignHref,
      sessionStore: store,
    })
    const result = await c.impersonateUser('u2', '排查线上工单问题')
    expect(result.ok).toBe(true)
    expect(assignHref).toHaveBeenCalledWith('/tenant/c1/work-panel/')
    expect(c.impersonateError.value).toBe('')
    expect(hoisted.persist).not.toHaveBeenCalled()
    // 409 复用会话：跳转前把 currentUserId 同步为目标用户，避免旧 ID 残留
    expect(hoisted.storeUserId).toHaveBeenCalledWith('u2')
  })

  it('redirects on Chinese 409 when already impersonating the same target', async () => {
    const assignHref = vi.fn()
    hoisted.apiFetch
      .mockResolvedValueOnce(jsonResponse(
        { detail: NESTED_IMPERSONATION_DETAIL, trace_id: 'tr-imp-zh' },
        { ok: false, status: 409, headers: { 'X-Trace-Id': 'tr-imp-zh' } },
      ))
      .mockResolvedValueOnce(jsonResponse({
        impersonating: true,
        target_user_id: 'u2',
        redirect_url: '/onboarding/',
      }))
    const c = useImpersonateUser({
      apiFetch: hoisted.apiFetch,
      persist: hoisted.persist,
      assignHref,
      sessionStore: store,
    })
    const result = await c.impersonateUser('u2', '排查线上工单问题')
    expect(result.ok).toBe(true)
    expect(assignHref).toHaveBeenCalledWith('/onboarding/')
    expect(c.impersonateError.value).toBe('')
  })

  it('shows Chinese nested error when impersonating a different user', async () => {
    const assignHref = vi.fn()
    hoisted.apiFetch
      .mockResolvedValueOnce(jsonResponse(
        { detail: NESTED_IMPERSONATION_DETAIL },
        { ok: false, status: 409 },
      ))
      .mockResolvedValueOnce(jsonResponse({
        impersonating: true,
        target_user_id: 'u-other',
        redirect_url: '/onboarding/',
      }))
    const c = useImpersonateUser({
      apiFetch: hoisted.apiFetch,
      persist: hoisted.persist,
      assignHref,
      sessionStore: store,
    })
    const result = await c.impersonateUser('u2', '排查线上工单问题')
    expect(result.ok).toBe(false)
    expect(c.impersonateError.value).toBe(NESTED_IMPERSONATION_DETAIL)
    expect(assignHref).not.toHaveBeenCalled()
    expect(hoisted.persist).not.toHaveBeenCalled()
  })

  it('rejects short reason without calling API', async () => {
    const c = useImpersonateUser({
      apiFetch: hoisted.apiFetch,
      persist: hoisted.persist,
      assignHref: vi.fn(),
      sessionStore: store,
    })
    const result = await c.impersonateUser('u2', '短')
    expect(result.ok).toBe(false)
    expect(c.impersonateError.value).toContain('理由')
    expect(hoisted.apiFetch).not.toHaveBeenCalled()
  })

  it('rejects modal prompt/placeholder as reason without calling API', async () => {
    const c = useImpersonateUser({
      apiFetch: hoisted.apiFetch,
      persist: hoisted.persist,
      assignHref: vi.fn(),
      sessionStore: store,
    })
    for (const forbidden of [
      '请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。',
      '例如：排查线上工单 T-12345',
    ]) {
      const result = await c.impersonateUser('u2', forbidden)
      expect(result.ok).toBe(false)
      expect(c.impersonateError.value).toContain('请填写具体的模拟登录理由')
    }
    expect(hoisted.apiFetch).not.toHaveBeenCalled()
  })
})
}
