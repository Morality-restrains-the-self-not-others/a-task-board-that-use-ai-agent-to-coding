// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsTaskPanel.click-guard.test.js requires vitest runtime')
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
      run: async (fn) => fn({ idempotencyKey: 'ik-test-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 'tenant-test' }, query: {} }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  }))
  vi.mock('../composables/useTenantPageAccess.js', () => ({
    useTenantPageAccess: () => ({ accessAllowed: true }),
  }))

  const { default: View } = await import('./WorkspaceSettingsTaskPanel.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  const STUBS = {
    TenantPageAccessEmpty: { template: '<div />' },
    WorkspaceSettingsTaskPanelActions: { template: '<div />' },
    DeliverableSystemSettings: { template: '<div />' },
    ProgressSystemSettings: { template: '<div />' },
    ProgressSystemSettingsModal: { template: '<div />' },
    AccessManagementModal: { template: '<div />' },
    WorkspaceMachinePolicyModal: { template: '<div />' },
    WorkspaceSettingsTaskPanelOptionsModals: { template: '<div />' },
  }

  describe('WorkspaceSettingsTaskPanel 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (!opts?.method) return jsonOk([]) // 工作空间列表空数组 → 不触发 loadTaskPanels
        return jsonOk({ ok: true })
      })
    })

    it('保存工作空间 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, { global: { stubs: STUBS } })
      await flushPromises()
      await wrapper.find('button.btn-primary').trigger('click') // 添加工作空间
      await flushPromises()
      const nameInput = wrapper.find('#workspace-name')
      await nameInput.setValue('测试工作空间')
      await flushPromises()
      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/projects/workspaces/tenant_id/tenant-test')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    })
  })
}
