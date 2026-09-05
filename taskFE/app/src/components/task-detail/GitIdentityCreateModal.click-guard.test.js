// @vitest-environment jsdom
// OPT-20260819-038: 创建 Git 身份是写操作（POST），防连点双发。
if (!process.env.VITEST) {
  console.log('[skip] GitIdentityCreateModal.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils', () => ({
    apiFetch,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-git-identity' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('../../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: async () => 'user-1',
  }))

  const { default: Component } = await import('./GitIdentityCreateModal.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('GitIdentityCreateModal 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') {
          return jsonOk({ id: 'gi-1' })
        }
        return jsonOk({ company_nicknames: [{ company_id: 'c1', company_name: '公司1' }] })
      })
    })

    it('创建身份 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, { props: { visible: false, tenantId: 'ten1' } })
      // visible 由 false → true 触发 watch → fetchMembershipCompanies 加载公司列表
      await wrapper.setProps({ visible: true })
      await flushPromises()

      await wrapper.find('#git-id-modal-company').setValue('c1')
      await wrapper.find('#git-id-modal-username').setValue('alice')
      await wrapper.find('#git-id-modal-email').setValue('alice@example.com')

      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建身份'))
      expect(createBtn).toBeTruthy()
      await createBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/git-identities/user/user-1/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-git-identity')
      wrapper.unmount()
    })
  })
}
