// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceMachinePolicyModal.saveGuard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  function jsonOk(body) {
    return Promise.resolve({ ok: true, status: 200, json: async () => body })
  }

  describe('WorkspaceMachinePolicyModal save guard', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      vi.stubGlobal('crypto', { randomUUID: () => 'ik-policy-1' })
    })

    function mockGets() {
      hoisted.apiFetchMock.mockImplementation((url, opts = {}) => {
        if (opts.method === 'PUT') return jsonOk({})
        if (opts.method === 'PATCH') return jsonOk({})
        if (url.includes('workspace-machine-policy')) {
          return jsonOk({ idle_recycle_minutes: 5, enabled_authorization_ids: [] })
        }
        if (url.includes('cloud-platform-authorizations')) return jsonOk([])
        return jsonOk({ container_image_at_mode_enabled: true })
      })
    }

    it('double-click save issues only one PUT with the same Idempotency-Key', async () => {
      mockGets()
      const { default: Modal } = await import('./WorkspaceMachinePolicyModal.vue')
      const wrapper = mount(Modal, {
        props: { show: true, tenantId: 't1', workspaceId: 'w1', workspaceName: 'ws' },
      })
      await flushPromises()

      const saveBtn = wrapper.get('button.btn-primary')
      await saveBtn.trigger('click')
      await saveBtn.trigger('click')
      await flushPromises()

      const puts = hoisted.apiFetchMock.mock.calls.filter((c) => c[1] && c[1].method === 'PUT')
      expect(puts).toHaveLength(1)
      expect(puts[0][1].headers['Idempotency-Key']).toBe('ik-policy-1')
    })

    it('save button carries aria-busy while in flight', async () => {
      let releasePut
      const putGate = new Promise((resolve) => {
        releasePut = resolve
      })
      hoisted.apiFetchMock.mockImplementation((url, opts = {}) => {
        if (opts.method === 'PUT') return putGate.then(() => jsonOk({}))
        if (opts.method === 'PATCH') return jsonOk({})
        if (url.includes('workspace-machine-policy')) {
          return jsonOk({ idle_recycle_minutes: 5, enabled_authorization_ids: [] })
        }
        if (url.includes('cloud-platform-authorizations')) return jsonOk([])
        return jsonOk({ container_image_at_mode_enabled: true })
      })
      const { default: Modal } = await import('./WorkspaceMachinePolicyModal.vue')
      const wrapper = mount(Modal, {
        props: { show: true, tenantId: 't1', workspaceId: 'w1', workspaceName: 'ws' },
      })
      await flushPromises()

      const saveBtn = wrapper.get('button.btn-primary')
      await saveBtn.trigger('click')
      await flushPromises()
      expect(saveBtn.attributes('aria-busy')).toBe('true')
      releasePut()
      await flushPromises()
      expect(saveBtn.attributes('aria-busy')).toBe('false')
    })
  })
}
