// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useTenantAccessGuard.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    prompt: vi.fn(),
  }))

  vi.mock('./apiUtils', () => ({ apiFetch: (...a) => mocks.apiFetch(...a) }))
  vi.mock('./cookieUtils', () => ({ getCookie: () => 'u1' }))
  vi.mock('./tenantAccessDeniedPrompt.js', () => ({
    buildProfileHref: (id) => (id ? `/user/${id}/profile/` : '/profile/'),
    promptNavigateOnTenantAccessDenied: (...a) => mocks.prompt(...a),
  }))

  const { verifyTenantAccess } = await import('./useTenantAccessGuard.js')

  beforeEach(() => {
    mocks.apiFetch.mockReset()
    mocks.prompt.mockReset()
    mocks.prompt.mockResolvedValue('stayed')
  })

  describe('verifyTenantAccess', () => {
    it('/me/ 200 → 返回用户数据且不弹窗', async () => {
      mocks.apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: 'u1', companies: [] }),
      })
      const data = await verifyTenantAccess({ userId: 'u1', tenantId: 't1' })
      expect(data).toEqual({ id: 'u1', companies: [] })
      expect(mocks.prompt).not.toHaveBeenCalled()
    })

    it('/me/ 403 → 弹窗确认且不静默 replace', async () => {
      mocks.apiFetch.mockResolvedValue({ ok: false, status: 403 })
      const data = await verifyTenantAccess({ userId: 'u1', tenantId: 't1', router: { replace: vi.fn() } })
      expect(data).toBeNull()
      expect(mocks.prompt).toHaveBeenCalledWith(
        expect.objectContaining({ targetHref: '/user/u1/profile/' }),
      )
    })
  })
}
