// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useImpersonationStatus.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

describe('useImpersonationStatus', () => {
  beforeEach(() => {
    vi.resetModules()
  })

  it('refresh sets impersonating from status payload', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({ impersonating: true, target_username: 'bob' }),
    }))
    const { useImpersonationStatus } = await import('./useImpersonationStatus.js')
    const c = useImpersonationStatus({ apiFetch, persist: vi.fn(), assignHref: vi.fn() })
    await c.refresh()
    expect(c.impersonating.value).toBe(true)
    expect(c.targetUsername.value).toBe('bob')
  })

  it('refresh overwrites local currentUserId with server-confirmed target (stale-session self-heal)', async () => {
    // 存量模拟会话（模拟登录早于 OPT-20260824-021 修复上线）localStorage
    // currentUserId 残留模拟者 ID → git-identities 等页面 403「获取身份列表失败」。
    // refresh 确认 impersonating 后以服务端 target_user_id 覆盖本地主存储。
    localStorage.setItem('currentUserId', 'bootstrap-admin')
    const apiFetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({
        impersonating: true,
        target_user_id: '877397583960502272',
        target_username: 'bob',
      }),
    }))
    const { useImpersonationStatus } = await import('./useImpersonationStatus.js')
    const c = useImpersonationStatus({ apiFetch, persist: vi.fn(), assignHref: vi.fn() })
    await c.refresh()
    expect(c.impersonating.value).toBe(true)
    expect(localStorage.getItem('currentUserId')).toBe('877397583960502272')
  })

  it('refresh does not touch currentUserId when not impersonating', async () => {
    localStorage.setItem('currentUserId', 'plain-user')
    const apiFetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({ impersonating: false }),
    }))
    const { useImpersonationStatus } = await import('./useImpersonationStatus.js')
    const c = useImpersonationStatus({ apiFetch, persist: vi.fn(), assignHref: vi.fn() })
    await c.refresh()
    expect(c.impersonating.value).toBe(false)
    expect(localStorage.getItem('currentUserId')).toBe('plain-user')
  })

  it('stop persists actor credentials and redirects', async () => {
    const persist = vi.fn(async () => {})
    const assignHref = vi.fn()
    const apiFetch = vi.fn(async () => ({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({ token: 'admin', user: { id: 'a' }, redirect_url: '/system-admin/users/' }),
    }))
    const { useImpersonationStatus } = await import('./useImpersonationStatus.js')
    const c = useImpersonationStatus({ apiFetch, persist, assignHref })
    const result = await c.stopImpersonation()
    expect(result.ok).toBe(true)
    expect(persist).toHaveBeenCalled()
    expect(assignHref).toHaveBeenCalledWith('/system-admin/users/')
  })
})
}
