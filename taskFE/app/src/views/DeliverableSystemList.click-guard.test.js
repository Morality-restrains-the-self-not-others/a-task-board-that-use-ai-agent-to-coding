// @vitest-environment jsdom
// OPT-20260819-038 回归：交付物体系 创建/删除/设默认 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] DeliverableSystemList.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    showRequestError: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: (...args) => mocks.showRequestError(...args),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: '873472655125147648' }, query: {}, name: 'deliverable_systems', fullPath: '/', path: '/' }),
    useRouter: () => ({ push: vi.fn() }),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => 'x' }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-deliverable' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const COMPANY_SYS = { id: 'ds_company_a', name: '公司体系A', description: '', is_system: false, is_default: false, level_names: ['价值流', '活动'] }
  const SYSTEM_SYS = { id: 'ds_default_global', name: '系统默认', description: '', is_system: true, is_default: false, level_names: ['价值流', '活动'] }

  beforeEach(() => {
    vi.clearAllMocks()
    window.confirm = vi.fn(() => true)
    mocks.apiFetch.mockImplementation((url, options) => {
      if (url.includes('deliverable-systems/tenant_id/') && !url.includes('set-default')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([COMPANY_SYS, SYSTEM_SYS]) })
      }
      if (url.includes('default-deliverable-system')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success' }) })
      }
      return Promise.resolve({ ok: true, status: 200, traceId: 'tid', json: () => Promise.resolve({ status: 'success' }) })
    })
  })

  const { default: DeliverableSystemList } = await import('./DeliverableSystemList.vue')

  describe('DeliverableSystemList 写操作 clickGuard 接线', () => {
    it('设默认 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(DeliverableSystemList)
      await flushPromises()

      await wrapper.vm.setAsDefault(SYSTEM_SYS)
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/projects/deliverable-systems/ds_default_global/set-default/tenant_id/873472655125147648/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-deliverable')
    })

    it('删除 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(DeliverableSystemList)
      await flushPromises()

      // 公司交付物体系行有删除按钮（系统级不渲染删除按钮）
      const rows = wrapper.findAll('.border.rounded-lg')
      const companyRow = rows.find((r) => r.text().includes('公司体系A'))
      expect(companyRow).toBeTruthy()
      const deleteBtn = companyRow.findAll('button').at(-1)
      expect(deleteBtn).toBeTruthy()
      await deleteBtn.trigger('click')
      await flushPromises()

      const delCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/projects/deliverable-systems/tenant_id/873472655125147648/ds_company_a/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-deliverable')
    })
  })
}
