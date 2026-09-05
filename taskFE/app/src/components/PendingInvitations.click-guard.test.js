// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PendingInvitations.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  const { confirm } = vi.hoisted(() => ({ confirm: vi.fn() }))
  const { writeText } = vi.hoisted(() => ({ writeText: vi.fn() }))

  vi.mock('../utils/modalService.js', () => ({ default: { confirm } }))
  vi.mock('../utils/toastService.js', () => ({ default: { success: vi.fn() } }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({ toastRequestError: vi.fn() }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言撤销/重发邀请 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-invite-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: PendingInvitations } = await import('./PendingInvitations.vue')

  const INVITES = [
    { id: 'inv-1', token: 'tok-1', invite_method: 'email', invite_target: 'a@x.com', created_at: '', expires_at: '', delivery_status: 'delivered', email_sent_at: '2026-08-01' },
  ]

  function jsonOk(body) {
    return { ok: true, json: async () => body }
  }

  describe('PendingInvitations 撤销/重发邀请实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      confirm.mockReset()
      writeText.mockReset()
      confirm.mockResolvedValue(true)
      Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
      // PendingInvitations 依赖 main.js 挂载的 window.apiFetch 全局（无 import）
      window.apiFetch = apiFetch
    })

    it('点撤销发起一次 POST 且携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST' && String(url).includes('revoke-invitation')) return jsonOk({})
        return jsonOk(INVITES)
      })
      const wrapper = mount(PendingInvitations, { props: { tenantId: 't-1' } })
      await flushPromises()

      await wrapper.findAll('button').find((b) => b.text().includes('撤销')).trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([url]) => String(url).includes('revoke-invitation'))
      expect(postCall).toBeTruthy()
      expect(postCall[1].method).toBe('POST')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-invite-1')
      wrapper.unmount()
    })

    it('点重发发起一次 POST 且携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST' && String(url).includes('resend-invitation-link')) {
          return jsonOk({ invite_token: 'tok-new', expires_at: '2026-08-08' })
        }
        return jsonOk(INVITES)
      })
      const wrapper = mount(PendingInvitations, { props: { tenantId: 't-1' } })
      await flushPromises()

      await wrapper.findAll('button').find((b) => b.text().includes('重新发送')).trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([url]) => String(url).includes('resend-invitation-link'))
      expect(postCall).toBeTruthy()
      expect(postCall[1].method).toBe('POST')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-invite-1')
      expect(JSON.parse(postCall[1].body)).toEqual({ expiration_days: 90 })
      wrapper.unmount()
    })
  })
}
