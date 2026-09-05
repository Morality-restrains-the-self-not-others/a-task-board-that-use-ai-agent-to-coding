// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] apiUtils.test.js requires vitest runtime')
} else {
const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')
const { apiFetch, parseCompanyMembersResponse, setCachedAuthToken } = await import('./apiUtils.js')

// OPT-20260808-014: GET 去重竞态 —— 并发相同 GET 在 auth token 解析挂起期间全部穿透去重检查
// （检查在 await 之前、注册在 await 之后），导致 Navbar/Sidebar/路由守卫的并发 /me/
// 在每个页面重复发出。修复后应合并为 1 次网络请求。
describe('apiUtils GET dedup race', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(mockOkResponse()))
    window.config = { API_BASE_URL: 'http://api.test' }
    vi.spyOn(console, 'log').mockImplementation(() => {})
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('coalesces concurrent identical GETs fired while token resolution is pending', async () => {
    // 重置模块状态，保证 _cachedAuthToken 回到 null（进入异步 token 解析路径）
    vi.resetModules()
    const fresh = await import('./apiUtils.js')
    const freshApiFetch = fresh.apiFetch

    // token 解析挂起：两个请求都会卡在 await resolveAuthToken() 期间穿过 dedup 检查
    let release
    hoisted.getActiveToken.mockReturnValue(
      new Promise((res) => { release = () => res('tok-race') }),
    )

    const p1 = freshApiFetch('/accounts/me/')
    const p2 = freshApiFetch('/accounts/me/')
    release()

    const [r1, r2] = await Promise.all([p1, p2])
    expect(r1.ok).toBe(true)
    expect(r2.ok).toBe(true)
    // 并发同 URL GET 应合并为 1 次网络请求
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('dedup failure releases waiters when the shared fetch rejects', async () => {
    vi.resetModules()
    const fresh = await import('./apiUtils.js')
    const freshApiFetch = fresh.apiFetch

    let release
    hoisted.getActiveToken.mockReturnValue(
      new Promise((res) => { release = () => res('tok-race') }),
    )
    fetch.mockRejectedValueOnce(new Error('network down'))

    const p1 = freshApiFetch('/accounts/me/')
    const p2 = freshApiFetch('/accounts/me/')
    release()

    const [e1, e2] = await Promise.allSettled([p1, p2])
    expect(e1.status).toBe('rejected')
    expect(e2.status).toBe('rejected')
    expect(fetch).toHaveBeenCalledTimes(1)
  })
})

const hoisted = vi.hoisted(() => ({ getActiveToken: vi.fn() }))

vi.mock('./domain/auth/services/saved_accounts_store.js', () => ({
  getActiveToken: hoisted.getActiveToken,
}))

function mockOkResponse() {
  const body = {}
  return {
    ok: true,
    status: 200,
    json: async () => body,
    clone: () => ({
      ok: true,
      status: 200,
      json: async () => body,
      text: async () => '',
    }),
  }
}

describe('apiUtils trace propagation', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(mockOkResponse()))
    window.config = { API_BASE_URL: 'http://api.test' }
    vi.spyOn(console, 'log').mockImplementation(() => {})
    vi.spyOn(console, 'error').mockImplementation(() => {})
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: 'csrftoken=test-csrf-token',
    })
    localStorage.clear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('apiFetch adds trace and span headers when missing', async () => {
    await apiFetch('/accounts/me/')

    expect(fetch).toHaveBeenCalledOnce()
    const [, options] = fetch.mock.calls[0]
    expect(options.headers['X-Trace-Id']).toBeTruthy()
    expect(options.headers['X-Parent-Span-Id']).toMatch(/^[0-9a-f]{16}$/)
    expect(options.headers.traceparent).toMatch(/^00-[0-9a-f]{32}-[0-9a-f]{16}-01$/)
  })

  it('apiFetch preserves caller-supplied trace headers', async () => {
    await apiFetch('/accounts/me/', {
      headers: {
        'X-Trace-Id': 'custom-trace-abc12345',
        'X-Parent-Span-Id': 'a1b2c3d4e5f67890',
      },
    })

    const [, options] = fetch.mock.calls[0]
    expect(options.headers['X-Trace-Id']).toBe('custom-trace-abc12345')
    expect(options.headers['X-Parent-Span-Id']).toBe('a1b2c3d4e5f67890')
    expect(options.headers.traceparent).toBeUndefined()
  })

  it('apiFetch does not attach CSRF token (CSRF handled server-side via cookie)', async () => {
    await apiFetch('/accounts/login/', { method: 'POST', body: JSON.stringify({}) })

    const [, options] = fetch.mock.calls[0]
    expect(options.headers['X-CSRFToken']).toBeUndefined()
  })

  it('apiFetch does not attach CSRF when credentials omit', async () => {
    await apiFetch('/accounts/login/', {
      method: 'POST',
      credentials: 'omit',
      body: JSON.stringify({}),
    })

    const [, options] = fetch.mock.calls[0]
    expect(options.headers['X-CSRFToken']).toBeUndefined()
  })

  it('apiFetch prefixes relative URLs with API_BASE_URL', async () => {
    await apiFetch('accounts/me/')

    const [url] = fetch.mock.calls[0]
    expect(url).toBe('http://api.test/accounts/me/')
  })

  it('apiFetch attaches response.traceId from request header when response omits it', async () => {
    const res = await apiFetch('/accounts/me/', {
      headers: { 'X-Trace-Id': 'client-trace-xyz' },
    })
    expect(res.traceId).toBe('client-trace-xyz')
  })

  it('apiFetch prefers response X-Trace-Id for response.traceId', async () => {
    fetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      headers: new Headers({ 'X-Trace-Id': 'server-trace-1' }),
      json: async () => ({ detail: 'boom' }),
      clone: () => ({
        json: async () => ({ detail: 'boom' }),
        text: async () => '{"detail":"boom"}',
      }),
    })
    const res = await apiFetch('/fail/', {
      headers: { 'X-Trace-Id': 'client-trace-xyz' },
    })
    expect(res.ok).toBe(false)
    expect(res.traceId).toBe('server-trace-1')
  })

  it('apiFetch TimeoutError carries request traceId', async () => {
    // OPT-20260809-013: 此前用真实 10ms 定时器，全量 1672 例并发时调度延迟
    // 偶发使超时在 abort 监听器挂载前触发（或迟于断言窗口），导致偶发失败。
    // 改为 vi.useFakeTimers 确定性推进：挂载监听器 → advanceTimersByTime 触发 abort。
    vi.useFakeTimers()
    try {
      fetch.mockImplementationOnce(
        (_url, options) =>
          new Promise((_resolve, reject) => {
            options.signal.addEventListener('abort', () => {
              const err = new Error('Aborted')
              err.name = 'AbortError'
              reject(err)
            })
          }),
      )
      const pending = apiFetch('/slow/', {
        timeout: 10,
        headers: { 'X-Trace-Id': 'timeout-trace' },
      })
      // 先同步挂载 rejects 断言，避免推进定时器时 rejection 先于 handler 触发
      // （否则 Node 会报 Unhandled Rejection，Vitest 记 1 error）。
      const assertion = expect(pending).rejects.toMatchObject({ name: 'TimeoutError', traceId: 'timeout-trace' })
      // 让 apiFetch 内部的 token/header 解析微任务先完成（fetch 已被调用、abort 监听器已挂载），
      // 再确定性推进 10ms 超时。先 advance 0ms 排空微任务队列。
      await vi.advanceTimersByTimeAsync(0)
      await vi.advanceTimersByTimeAsync(10)
      await assertion
    } finally {
      vi.useRealTimers()
    }
  })

  it('apiFetch on task-detail accessCode share page omits stale Authorization', async () => {
    hoisted.getActiveToken.mockResolvedValue('stale-token')
    setCachedAuthToken('stale-token')
    vi.stubGlobal('location', {
      pathname: '/tenant/1/workspace/ws_1/task-detail/task_abc/',
      search: '?accessCode=abc123',
      href: '',
    })
    await apiFetch('/api/tenant/1/tasks/task_abc/')
    const [, options] = fetch.mock.calls[0]
    expect(options.headers.Authorization).toBeUndefined()
    expect(options.headers['Authorization']).toBeUndefined()
    // 仍保留 cookie 会话：credentials 未改
    expect(options.credentials).toBe('include')
  })

  it('apiFetch clears invalid authToken and retries without Authorization', async () => {
    hoisted.getActiveToken.mockResolvedValue('stale-token')
    setCachedAuthToken('stale-token')
    const invalid = {
      ok: false,
      status: 403,
      json: async () => ({ detail: 'Invalid token.' }),
      clone: () => ({
        json: async () => ({ detail: 'Invalid token.' }),
        text: async () => '{"detail":"Invalid token."}',
      }),
    }
    const ok = mockOkResponse()
    fetch.mockResolvedValueOnce(invalid).mockResolvedValueOnce(ok)

    const res = await apiFetch('/api/privacy-policy/public/current/')

    expect(res.ok).toBe(true)
    expect(fetch).toHaveBeenCalledTimes(2)
    const firstHeaders = fetch.mock.calls[0][1].headers
    const secondHeaders = fetch.mock.calls[1][1].headers
    expect(firstHeaders.Authorization).toBe('Token stale-token')
    expect(secondHeaders.Authorization).toBeUndefined()
    expect(secondHeaders._retriedWithoutAuth).toBeUndefined()

    // token 缓存已被清除：后续请求不再带 Authorization（不再回退到旧 token）
    hoisted.getActiveToken.mockResolvedValue(null)
    fetch.mockResolvedValueOnce(mockOkResponse())
    await apiFetch('/api/privacy-policy/public/current/')
    expect(fetch).toHaveBeenCalledTimes(3)
    const thirdHeaders = fetch.mock.calls[2][1].headers
    expect(thirdHeaders.Authorization).toBeUndefined()
  })

  it('apiFetch redirects to login on gateway forward-auth session expiry (unified)', async () => {
    const detail = '无法解析登录凭据，请重新登录'
    fetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({ detail }),
      clone: () => ({
        json: async () => ({ detail }),
        text: async () => `{"detail":"${detail}"}`,
      }),
    })
    vi.stubGlobal('location', { pathname: '/system-admin/users/', search: '', href: '' })

    const res = await apiFetch('/api/system-admin/users/')

    // 响应仍原样返回（调用方可继续处理），_errorData 已挂载
    expect(res.ok).toBe(false)
    expect(res.status).toBe(401)
    expect(res._errorData.detail).toBe(detail)
    // 统一收口：自动跳转登录页并携带当前页回跳
    expect(window.location.href).toBe('/auth/login/?next=%2Fsystem-admin%2Fusers%2F')
  })

  it('skipSessionExpiredRedirect 时 401 不跳登录，响应原样返回', async () => {
    vi.resetModules()
    const { apiFetch: freshApiFetch } = await import('./apiUtils.js')
    const detail = '无法解析登录凭据，请重新登录'
    fetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({ detail }),
      clone: () => ({
        json: async () => ({ detail }),
        text: async () => `{"detail":"${detail}"}`,
      }),
    })
    vi.stubGlobal('location', { pathname: '/system-admin/users/', search: '', href: '' })

    const res = await freshApiFetch('/api/git-oauth/user-app-connection/', {
      skipSessionExpiredRedirect: true,
    })

    expect(res.ok).toBe(false)
    expect(res.status).toBe(401)
    expect(window.location.href).toBe('')
  })

  it('skipSessionExpiredRedirect 时即使 body 含 redirect_url 也不跳转', async () => {
    vi.resetModules()
    const { apiFetch: freshApiFetch } = await import('./apiUtils.js')
    const detail = '无法解析登录凭据，请重新登录'
    fetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({ detail, redirect_url: '/auth/login/?next=%2Fsystem-admin%2F' }),
      clone: () => ({
        json: async () => ({ detail, redirect_url: '/auth/login/?next=%2Fsystem-admin%2F' }),
        text: async () => '{}',
      }),
    })
    vi.stubGlobal('location', { pathname: '/system-admin/users/', search: '', href: '' })

    const res = await freshApiFetch('/api/git-oauth/user-app-connection/', {
      skipSessionExpiredRedirect: true,
    })

    expect(res.ok).toBe(false)
    expect(window.location.href).toBe('')
  })
})

describe('parseCompanyMembersResponse', () => {
  it('normalizes array legacy format', () => {
    const out = parseCompanyMembersResponse([{ id: '1' }])
    expect(out.members).toEqual([{ id: '1' }])
    expect(out.meta).toEqual({})
  })

  it('passes through object format', () => {
    const out = parseCompanyMembersResponse({ members: [{ id: '2' }], meta: { total: 1 } })
    expect(out.members).toEqual([{ id: '2' }])
    expect(out.meta).toEqual({ total: 1 })
  })
})
}
