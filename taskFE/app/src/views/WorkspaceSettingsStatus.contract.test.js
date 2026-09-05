// @vitest-environment jsdom
// OPT-20260808-006 回归测试：进度体系页挂载只调用新契约端点，不再调用
// manage-progress-column（旧 Django 契约端点未返回 progress_columns，
// 且页内列 CRUD 模态框从未被触发 —— 纯死代码，本次移除）。
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsStatus.contract.test.js requires vitest runtime')
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

  const SYSTEM = { id: 'ps_sys_default', name: '系统默认', columns: [{ id: 'c1', name: '待办' }] }
  const TENANT = { id: 'ps_tenant_a', name: '租户体系A', columns: [{ id: 'c2', name: '进行中' }] }

  function mockLoad() {
    mocks.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/system/progress-systems/')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success', project_progress_systems: [SYSTEM] }) })
      }
      if (url.includes('/api/projects/progress-systems/tenant_id/')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success', project_progress_systems: [TENANT] }) })
      }
      if (url.includes('/api/projects/settings/default-progress-system/')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success', progress_system: SYSTEM }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    })
  }

  const { default: WorkspaceSettingsStatus } = await import('./WorkspaceSettingsStatus.vue')

  describe('WorkspaceSettingsStatus 进度体系页 URL 契约（OPT-20260808-006）', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.showRequestError.mockReset()
    })

    it('挂载时仅调用系统级/租户级进度体系与默认进度体系三个新契约端点', async () => {
      mockLoad()
      const wrapper = mount(WorkspaceSettingsStatus)
      await flushPromises()

      const urls = mocks.apiFetch.mock.calls.map((c) => c[0])
      expect(urls).toContain('/api/system/progress-systems/')
      expect(urls).toContain('/api/projects/progress-systems/tenant_id/873472655125147648')
      expect(urls).toContain('/api/projects/settings/default-progress-system/tenant_id/873472655125147648/')
      // 回归：不再调用旧 manage-progress-column 端点（契约缺口 + 死代码）
      expect(urls.filter((u) => u.includes('manage-progress-column'))).toEqual([])
    })

    it('不渲染旧「添加/编辑进度列」模态框（死代码已移除）', async () => {
      mockLoad()
      const wrapper = mount(WorkspaceSettingsStatus)
      await flushPromises()
      expect(wrapper.text()).not.toContain('添加新进度列')
      expect(wrapper.text()).not.toContain('编辑进度列')
    })
  })
}
