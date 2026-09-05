// @vitest-environment jsdom
// OPT-20260819-038 回归：进度体系管理 创建 POST / 删除 DELETE / 设默认 POST 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsStatus.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    showRequestError: vi.fn(),
    useRoute: () => ({ params: { tenant: '873472655125147648' }, query: {}, fullPath: '/', path: '/' }),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    extractErrorMessage: () => 'mock-err',
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: (...args) => mocks.showRequestError(...args),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.useRoute(),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => 'x' }))
  vi.mock('../composables/useTenantPageAccess.js', () => ({
    useTenantPageAccess: () => ({ accessAllowed: true }),
  }))
  vi.mock('../components/TenantPageAccessEmpty.vue', () => ({
    default: { template: '<div />' },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-status' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const SYSTEM = { id: 'ps_sys_default', name: '系统默认', columns: [{ id: 'c1', name: '待办' }] }
  const TENANT = { id: 'ps_tenant_a', name: '租户体系A', columns: [{ id: 'c2', name: '进行中' }] }

  beforeEach(() => {
    vi.clearAllMocks()
    window.confirm = vi.fn(() => true)
    mocks.apiFetch.mockImplementation((url, opts) => {
      if (url.includes('/api/system/progress-systems/')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success', project_progress_systems: [SYSTEM] }) })
      }
      if (url.includes('/api/projects/progress-systems/tenant_id/')) {
        if (opts?.method === 'DELETE' || opts?.method === 'PUT') {
          return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success', project_progress_systems: [TENANT] }) })
        }
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success', project_progress_systems: [TENANT] }) })
      }
      if (url.includes('/api/projects/settings/default-progress-system/')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success', progress_system: SYSTEM }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    })
  })

  const { default: WorkspaceSettingsStatus } = await import('./WorkspaceSettingsStatus.vue')

  describe('WorkspaceSettingsStatus 写操作 clickGuard 接线', () => {
    it('创建进度体系 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(WorkspaceSettingsStatus)
      await flushPromises()

      await wrapper.find('#add-status-btn').trigger('click')
      await flushPromises()

      await wrapper.find('input[type="text"]').setValue('新体系')
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/projects/progress-systems/tenant_id/873472655125147648')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-status')
    })

    it('删除租户进度体系 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(WorkspaceSettingsStatus)
      await flushPromises()

      const row = wrapper.findAll('.flex.items-center.justify-between.p-3').find((el) =>
        el.text().includes('租户体系A'),
      )
      expect(row).toBeTruthy()
      const deleteBtn = row.findAll('button').at(-1)
      expect(deleteBtn).toBeTruthy()
      await deleteBtn.trigger('click')
      await flushPromises()

      const delCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/projects/progress-systems/tenant_id/873472655125147648/ps_tenant_a/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-status')
    })
  })
}
