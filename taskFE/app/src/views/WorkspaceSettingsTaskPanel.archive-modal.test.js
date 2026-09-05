// @vitest-environment jsdom
/**
 * 回归：settings/task-panel「套餐设置」必须打开任务存档档位模态。
 * 修前 isTaskArchiveFeatureEnabled=false，点击空操作，行上写「该功能暂未开放」。
 */
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsTaskPanel.archive-modal.test.js requires vitest runtime')
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
      run: async (fn) => fn({ idempotencyKey: 'ik-archive-1' }),
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
    DeliverableSystemSettings: { template: '<div />' },
    ProgressSystemSettings: { template: '<div />' },
    ProgressSystemSettingsModal: { template: '<div />' },
    AccessManagementModal: { template: '<div />' },
    WorkspaceMachinePolicyModal: { template: '<div />' },
    WorkspaceSettingsTaskPanelOptionsModals: { template: '<div />' },
    WorkspaceCreateEditModal: { template: '<div />' },
  }

  const LIST = [
    {
      id: 'ws-1',
      name: '用户的工作空间',
      is_current: true,
      is_default: true,
      description: '',
      task_archive_tier: '7d',
    },
  ]

  describe('WorkspaceSettingsTaskPanel 套餐设置打开任务存档模态', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (String(url).includes('manage-task-panels')) {
          return jsonOk({ status: 'success', task_panels: [] })
        }
        if (opts?.method === 'PATCH') return jsonOk({ ok: true })
        return jsonOk(LIST)
      })
    })

    it('列表展示当前存档档位，不再写该功能暂未开放', async () => {
      const wrapper = mount(View, { global: { stubs: STUBS } })
      await flushPromises()
      const label = wrapper.find('[data-testid="task-archive-tier-label"]')
      expect(label.exists()).toBe(true)
      expect(label.text()).toContain('存档：免费存放期(7天)')
      expect(wrapper.text()).not.toContain('该功能暂未开放')
      wrapper.unmount()
    })

    it('点击套餐设置打开「套餐设置 · 任务存档时间」模态', async () => {
      const wrapper = mount(View, { global: { stubs: STUBS } })
      await flushPromises()
      const btn = wrapper.find('[data-testid="open-task-archive-settings"]')
      expect(btn.exists()).toBe(true)
      await btn.trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('套餐设置 · 任务存档时间')
      wrapper.unmount()
    })

    it('保存存档档位 PATCH 携带 Idempotency-Key 与 task_archive_tier', async () => {
      const wrapper = mount(View, { global: { stubs: STUBS } })
      await flushPromises()
      await wrapper.find('[data-testid="open-task-archive-settings"]').trigger('click')
      await flushPromises()
      const select = wrapper.find('[data-testid="task-archive-tier-select"]')
      expect(select.exists()).toBe(true)
      await select.setValue('3m')
      await flushPromises()
      const saveBtn = wrapper.find('[data-testid="task-archive-tier-save"]')
      await saveBtn.trigger('click')
      await flushPromises()

      const patchCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PATCH')
      expect(patchCall).toBeTruthy()
      expect(patchCall[0]).toBe('/api/projects/workspaces/tenant_id/tenant-test/ws-1/')
      expect(patchCall[1].headers['Idempotency-Key']).toBe('ik-archive-1')
      expect(JSON.parse(patchCall[1].body)).toEqual({ task_archive_tier: '3m' })
      wrapper.unmount()
    })
  })
}
