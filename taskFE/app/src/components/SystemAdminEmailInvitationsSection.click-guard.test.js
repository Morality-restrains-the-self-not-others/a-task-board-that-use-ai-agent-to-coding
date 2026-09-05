// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminEmailInvitationsSection.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-email' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Component } = await import('./SystemAdminEmailInvitationsSection.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminEmailInvitationsSection 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk({ invitations: [] })
        }
        return jsonOk({ ok: true })
      })
    })

    it('发送邮箱邀请 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, {})
      await flushPromises()
      wrapper.vm.openEmailInviteModal()
      await flushPromises()

      const emailInput = wrapper.find('#invite-email')
      await emailInput.setValue('alice@example.com')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/system-admin/email-invitations/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-email')
      expect(postCall[1].body).toContain('alice@example.com')
    })

    it('列表行重新发送邀请 POST 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk({
            invitations: [
              { id: 'inv-1', email: 'bob@example.com', canResend: true },
            ],
          })
        }
        return jsonOk({ ok: true })
      })
      const wrapper = mount(Component, {})
      await flushPromises()

      const resendBtn = wrapper.findAll('button').find((b) => b.text().includes('重新发送'))
      expect(resendBtn).toBeTruthy()
      await resendBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/system-admin/email-invitations/resend/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-email')
      expect(postCall[1].body).toContain('bob@example.com')
    })

    it('批量重发 POST 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk({
            invitations: [
              { id: 'inv-1', email: 'bob@example.com', canResend: true },
            ],
          })
        }
        return jsonOk({ ok: true, message: '批量重发完成' })
      })
      const wrapper = mount(Component, {})
      await flushPromises()

      const bulkBtn = wrapper.findAll('button').find((b) => b.text().includes('全部重发'))
      expect(bulkBtn).toBeTruthy()
      await bulkBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST' && String(opts?.url ?? '').includes('bulk'))
      const bulkCall = apiFetch.mock.calls.find(([url, opts]) => String(url).includes('bulk-resend'))
      expect(bulkCall).toBeTruthy()
      expect(bulkCall[0]).toBe('/api/system-admin/email-invitations/bulk-resend/')
      expect(bulkCall[1].headers['Idempotency-Key']).toBe('ik-test-email')
    })
  })
}
