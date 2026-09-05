// @vitest-environment jsdom
/**
 * 回归：settings/task-panel 列表「默认」badge 必须跟 is_default，
 * 不能跟 is_current（当前选中工作空间）。修前勾选「是否设为默认」保存后
 * badge 仍钉在 is_current 行，用户感知为「修改没有生效」。
 */
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsTaskPanel.default-badge.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-1' }),
      isBusy: () => false,
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

  const LIST = [
    { id: 'ws-current', name: 'Alpha Current', is_current: true, is_default: false, description: '' },
    { id: 'ws-default', name: 'Beta Flagged', is_current: false, is_default: true, description: '' },
  ]

  describe('WorkspaceSettingsTaskPanel 默认 badge 跟 is_default', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (_url, opts) => {
        if (!opts?.method || opts.method === 'GET') return jsonOk(LIST)
        return jsonOk({ ok: true })
      })
    })

    it('只在 is_default 行显示「默认」，即使另一行 is_current', async () => {
      const wrapper = mount(View, { global: { stubs: STUBS } })
      await flushPromises()

      const badges = wrapper.findAll('[data-testid="workspace-default-badge"]')
      expect(badges).toHaveLength(1)
      const rowText = badges[0].element.closest('.border')?.textContent || ''
      expect(rowText).toContain('Beta Flagged')
      expect(rowText).not.toContain('Alpha Current')
    })

    it('创建弹窗勾选「是否设为默认」后 POST body 含 is_default:true', async () => {
      const wrapper = mount(View, { global: { stubs: STUBS } })
      await flushPromises()
      await wrapper.find('button.btn-primary').trigger('click')
      await flushPromises()

      const nameInput = wrapper.find('#workspace-name')
      await nameInput.setValue('Gamma New')
      const checkbox = wrapper.find('#workspace-is-default')
      expect(checkbox.exists()).toBe(true)
      await checkbox.setValue(true)
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      const body = JSON.parse(postCall[1].body)
      expect(body.is_default).toBe(true)
      expect(body.name).toBe('Gamma New')
    })
  })
}
