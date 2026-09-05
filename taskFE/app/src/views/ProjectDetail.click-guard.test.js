// @vitest-environment jsdom
// OPT-20260819-038 回归：项目详情 删除项目 DELETE 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetail.click-guard.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: { params: { tenant: '827923618468040704', id: '846027310833254400' }, path: '/', query: {} },
    pushMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))
  vi.mock('../utils/githubAppReturnStorage.js', () => ({
    createGithubAppReturnKey: () => 'a'.repeat(32),
    setGithubAppReturnTarget: vi.fn(),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => ({ push: hoistedMocks.pushMock }),
  }))
  vi.mock('../utils/repoOAuthAuthorizeUtils.js', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, getGitOAuthProviderCatalog: () => [] }
  })
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-projdetail' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const PROJECT = {
    id: '846027310833254400',
    name: '项目A',
    description: '',
    git_repos: [],
    git_repo_entries: [],
    workspaces: [],
    tags: [],
    container_image_id: '',
    server_run_template: {},
  }

  beforeEach(() => {
    vi.clearAllMocks()
    window.confirm = vi.fn(() => true)
    hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
      const path = String(url || '')
      if (path.includes('/projects/846027310833254400/')) {
        return { ok: true, json: async () => PROJECT }
      }
      if (path.includes('/workspaces/')) {
        return { ok: true, json: async () => [] }
      }
      if (path.includes('/installed-images/')) {
        return { ok: true, json: async () => [], text: async () => '[]' }
      }
      return { ok: true, json: async () => ({}) }
    })
  })

  const { default: ProjectDetail } = await import('./ProjectDetail.vue')

  describe('ProjectDetail 写操作 clickGuard 接线', () => {
    it('删除项目 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(ProjectDetail)
      for (let i = 0; i < 4; i += 1) {
        await Promise.resolve()
        await nextTick()
      }

      const deleteBtn = wrapper.findAll('button').find((b) => b.text().includes('删除项目'))
      expect(deleteBtn).toBeTruthy()
      await deleteBtn.trigger('click')
      await nextTick()
      await Promise.resolve()

      const delCall = hoistedMocks.apiFetchMock.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/projects/846027310833254400/tenant_id/827923618468040704/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-projdetail')
    })
  })
}
