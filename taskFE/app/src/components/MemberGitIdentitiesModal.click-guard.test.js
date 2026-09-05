// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] MemberGitIdentitiesModal.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    showRequestError: vi.fn(),
    humanizeRequestErrorMessage: vi.fn((msg) => msg),
  }))

  vi.mock('../utils/apiUtils.js', () => ({ apiFetch: (...a) => mocks.apiFetch(...a) }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: (...a) => mocks.showRequestError(...a),
    humanizeRequestErrorMessage: (...a) => mocks.humanizeRequestErrorMessage(...a),
  }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-gitid-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Modal } = await import('./MemberGitIdentitiesModal.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  function mountModal() {
    return mount(Modal, {
      props: { tenantId: 't-1', memberId: 'm-1', memberName: '张三' },
    })
  }

  describe('MemberGitIdentitiesModal 创建/设默认/删除实施 clickGuard 接线', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method) return jsonOk({})
        return jsonOk({
          identities: [
            { id: 'i-1', git_user_name: 'u1', git_user_email: 'u1@x.com', is_default: true },
            { id: 'i-2', git_user_name: 'u2', git_user_email: 'u2@x.com', is_default: false },
          ],
        })
      })
      vi.stubGlobal('confirm', vi.fn(() => true))
    })
    afterEach(() => {
      vi.unstubAllGlobals()
    })

    it('提交创建表单的 POST 携带 Idempotency-Key 头', async () => {
      const wrapper = mountModal()
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const postCalls = mocks.apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'POST')
      expect(postCalls.length).toBeGreaterThan(0)
      for (const [, opts] of postCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-gitid-1')
      }
      wrapper.unmount()
    })

    it('点「设默认」的 PATCH 携带 Idempotency-Key 头', async () => {
      const wrapper = mountModal()
      await flushPromises()
      await wrapper.findAll('button').find((b) => b.text().includes('设默认')).trigger('click')
      await flushPromises()

      const patchCalls = mocks.apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'PATCH')
      expect(patchCalls.length).toBeGreaterThan(0)
      for (const [, opts] of patchCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-gitid-1')
      }
      wrapper.unmount()
    })

    it('点「删除」的 DELETE 携带 Idempotency-Key 头', async () => {
      const wrapper = mountModal()
      await flushPromises()
      await wrapper.findAll('button').find((b) => b.text().includes('删除')).trigger('click')
      await flushPromises()

      const delCalls = mocks.apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'DELETE')
      expect(delCalls.length).toBeGreaterThan(0)
      for (const [, opts] of delCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-gitid-1')
      }
      wrapper.unmount()
    })
  })
}
