// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceAssociation.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    alert: vi.fn(),
    showRequestError: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({ apiFetch: (...a) => mocks.apiFetch(...a), extractErrorMessage: () => '' }))
  vi.mock('../utils/modalService.js', () => ({ default: { alert: (...a) => mocks.alert(...a) } }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({ showRequestError: (...a) => mocks.showRequestError(...a) }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-wsassoc-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Panel } = await import('./WorkspaceAssociation.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  function mountPanel() {
    return mount(Panel, {
      props: {
        project: { id: 'p-1' },
        workspaces: [{ id: 'w1', name: 'WS1' }],
        availableWorkspaces: [{ id: 'w2', name: 'WS2' }],
        loading: false,
        tenantId: 't-1',
      },
    })
  }

  describe('WorkspaceAssociation 添加/移除关联实施 clickGuard 接线', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') return jsonOk({})
        return jsonOk({ workspaces: [] })
      })
    })

    it('点「添加关联」的 POST 携带 Idempotency-Key 头', async () => {
      const wrapper = mountPanel()
      await wrapper.find('select').setValue('w2')
      await wrapper.findAll('button').find((b) => b.text().includes('添加关联')).trigger('click')
      await flushPromises()

      const postCalls = mocks.apiFetch.mock.calls.filter(
        ([url, opts]) => opts?.method === 'POST' && !JSON.stringify(opts.body).includes('remove')
      )
      expect(postCalls.length).toBeGreaterThan(0)
      for (const [, opts] of postCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-wsassoc-1')
      }
      wrapper.unmount()
    })

    it('点「移除关联」的 POST 携带 Idempotency-Key 头', async () => {
      const wrapper = mountPanel()
      await wrapper.findAll('button').find((b) => b.text().includes('移除关联')).trigger('click')
      await flushPromises()

      const postCalls = mocks.apiFetch.mock.calls.filter(
        ([url, opts]) => opts?.method === 'POST' && JSON.stringify(opts.body).includes('remove')
      )
      expect(postCalls.length).toBeGreaterThan(0)
      for (const [, opts] of postCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-wsassoc-1')
      }
      wrapper.unmount()
    })
  })
}
