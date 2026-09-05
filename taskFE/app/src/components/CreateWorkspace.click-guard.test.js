// @vitest-environment jsdom
// OPT-20260819-038: 创建工作空间是写操作（POST），防连点双发。
if (!process.env.VITEST) {
  console.log('[skip] CreateWorkspace.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-create-ws' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Component } = await import('./CreateWorkspace.vue')

  describe('CreateWorkspace 写操作 clickGuard 接线', () => {
    let apiFetch
    beforeEach(() => {
      apiFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ id: 'ws-1', name: '新工作空间' }),
        headers: { get: () => 'application/json' },
      })
      vi.stubGlobal('apiFetch', apiFetch)
    })
    afterEach(() => {
      vi.unstubAllGlobals()
    })

    it('创建工作空间 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, { props: { tenantId: 'ten1' } })
      await wrapper.find('#name').setValue('新工作空间')
      await wrapper.find('#description').setValue('描述')

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/projects/workspaces/tenant_id/ten1')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-create-ws')
      expect(JSON.parse(postCall[1].body).name).toBe('新工作空间')
      wrapper.unmount()
    })
  })
}
