// @vitest-environment jsdom
// OPT-20260819-038 回归：工作空间设置 各分项保存 POST 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettings.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    showRequestError: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    extractErrorMessage: () => 'mock-err',
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: (...args) => mocks.showRequestError(...args),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 'ws-1' }, query: {} }),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => 'x' }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-wssettings' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  beforeEach(() => {
    vi.clearAllMocks()
    window.alert = vi.fn()
    mocks.apiFetch.mockImplementation(async () => ({
      ok: true,
      json: async () => ({ status: 'success' }),
    }))
  })

  const { default: WorkspaceSettings } = await import('./WorkspaceSettings.vue')

  describe('WorkspaceSettings 写操作 clickGuard 接线', () => {
    it('保存基本信息 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(WorkspaceSettings)
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().includes('保存'))
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/workspace/settings/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-wssettings')
    })
  })
}
