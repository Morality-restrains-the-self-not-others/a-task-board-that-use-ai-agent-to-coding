// @vitest-environment jsdom
// OPT-20260819-038 回归：编辑项目 PUT 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] ProjectEdit.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    push: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1', id: 'p1' }, query: {} }),
    useRouter: () => ({ push: (...args) => mocks.push(...args), replace: vi.fn(), back: vi.fn() }),
  }))
  vi.mock('../components/ProjectTagsInput.vue', () => ({
    default: { template: '<div />' },
  }))
  vi.mock('../components/ProjectRunTemplatePanel.vue', () => ({
    default: { template: '<div />' },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PUT 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-projedit' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.apiFetch.mockImplementation(async (url, opts) => {
      if (opts?.method === 'PUT') {
        return { ok: true, json: async () => ({ id: 'p1' }) }
      }
      if (String(url).includes('/api/projects/tenant_id/t1/p1/')) {
        return {
          ok: true,
          json: async () => ({
            id: 'p1',
            name: '项目A',
            description: '',
            container_image_id: '',
            workspaces: [],
            tags: [],
            git_repos: [],
            server_run_template: {},
          }),
        }
      }
      if (String(url).includes('installed-images')) {
        return { ok: true, json: async () => [] }
      }
      if (String(url).includes('workspaces')) {
        return { ok: true, json: async () => [] }
      }
      return { ok: true, json: async () => ({}) }
    })
  })

  const { default: ProjectEdit } = await import('./ProjectEdit.vue')

  describe('ProjectEdit 写操作 clickGuard 接线', () => {
    it('保存项目 PUT 携带 Idempotency-Key', async () => {
      const wrapper = mount(ProjectEdit)
      await flushPromises()

      const form = wrapper.find('form')
      expect(form.exists()).toBe(true)
      await form.trigger('submit')
      await flushPromises()

      const putCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'PUT')
      expect(putCall).toBeTruthy()
      expect(putCall[0]).toBe('/api/projects/tenant_id/t1/p1/')
      expect(putCall[1].headers['Idempotency-Key']).toBe('ik-test-projedit')
    })
  })
}
