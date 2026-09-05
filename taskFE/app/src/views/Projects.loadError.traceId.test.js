// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Projects.loadError.traceId.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { nextTick, ref, computed } = await import('vue')
const { mount, flushPromises } = await import('@vue/test-utils')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  routeMock: {
    params: { tenant: '850256677331562496' },
    query: {},
    path: '/tenant/850256677331562496/projects/',
  },
  replaceMock: vi.fn(),
  ref: require('vue').ref,
  computed: require('vue').computed,
  gitlabSync: null,
}))
const apiFetchMock = hoisted.apiFetchMock
const routeMock = hoisted.routeMock
const replaceMock = hoisted.replaceMock

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetchMock(...args),
}))

vi.mock('../utils/cookieUtils.js', () => ({
  getCookie: () => '',
}))

vi.mock('vue-router', () => ({
  useRoute: () => hoisted.routeMock,
  useRouter: () => ({ replace: hoisted.replaceMock, push: vi.fn() }),
}))

vi.mock('../composables/useGitlabProjectSync.js', () => {
  const sync = {
    showModal: hoisted.ref(false),
    loading: hoisted.ref(false),
    creating: hoisted.ref(false),
    error: hoisted.ref(''),
    errorTraceId: hoisted.ref(''),
    oauthBound: hoisted.ref(false),
    gitlabWebsite: hoisted.ref(''),
    gitlabLogin: hoisted.ref(''),
    oauthStartHref: hoisted.ref(''),
    purchasedRegions: hoisted.ref([]),
    selectedRegionSlug: hoisted.ref(''),
    regionsLoading: hoisted.ref(false),
    repos: hoisted.ref([]),
    selectedCount: hoisted.ref(0),
    batchEligibleCount: hoisted.ref(0),
    allSelectableChecked: hoisted.ref(false),
    selectedRepoKeys: hoisted.ref({}),
    workspaces: hoisted.ref([]),
    selectedWorkspaceId: hoisted.ref(''),
    createResult: hoisted.ref(null),
    combinedProjectName: hoisted.ref(''),
    openModal: vi.fn(),
    closeModal: vi.fn(),
    loadRemoteRepos: vi.fn(),
    selectRegion: vi.fn(),
    toggleRepo: vi.fn(),
    toggleSelectAll: vi.fn(),
    startGitlabOAuth: vi.fn(),
    createSelectedProjects: vi.fn(),
    createCombinedProject: vi.fn(),
    setCombinedProjectName: vi.fn(),
  }
  hoisted.gitlabSync = sync
  return { useGitlabProjectSync: () => sync }
})

vi.mock('../composables/useProjectsBatchDelete.js', () => ({
  useProjectsBatchDelete: () => ({
    selectionMode: hoisted.ref(false),
    selectedProjectIds: hoisted.ref({}),
    showConfirmModal: hoisted.ref(false),
    deleting: hoisted.ref(false),
    error: hoisted.ref(''),
    errorTraceId: hoisted.ref(''),
    deleteResult: hoisted.ref(null),
    selectedCount: hoisted.ref(0),
    selectedProjects: hoisted.computed(() => []),
    allFilteredChecked: hoisted.ref(false),
    toggleSelectionMode: vi.fn(),
    exitSelectionMode: vi.fn(),
    toggleProject: vi.fn(),
    toggleSelectAllFiltered: vi.fn(),
    openBatchDeleteConfirm: vi.fn(),
    closeConfirmModal: vi.fn(),
    executeBatchDelete: vi.fn(),
  }),
}))

describe('Projects.loadError data-traceId', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
    replaceMock.mockReset()
    routeMock.params = { tenant: '850256677331562496' }
    routeMock.query = {}
  })

  it('shows 请先登录 and binds traceId on 401', async () => {
    apiFetchMock.mockResolvedValue({
      ok: false,
      status: 401,
      traceId: 'tid-projects-load-401',
      json: async () => ({ error: '请先登录', message: '请先登录' }),
    })

    const { default: Projects } = await import('./Projects.vue')
    const wrapper = mount(Projects)
    await flushPromises()
    await nextTick()

    const errEl = wrapper.find('p.text-red-600')
    expect(errEl.exists()).toBe(true)
    expect(errEl.text()).toContain('请先登录')
    expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe(
      'tid-projects-load-401',
    )
  })

  it('binds response.traceId on load failure', async () => {
    apiFetchMock.mockResolvedValue({
      ok: false,
      status: 500,
      traceId: 'tid-projects-load-500',
      json: async () => ({ detail: 'server error' }),
    })

    const { default: Projects } = await import('./Projects.vue')
    const wrapper = mount(Projects)
    await flushPromises()
    await nextTick()

    const errEl = wrapper.find('p.text-red-600')
    expect(errEl.exists()).toBe(true)
    expect(errEl.text()).toContain('加载项目失败')
    expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe(
      'tid-projects-load-500',
    )
  })

  it('omits data-traceId when tenant id is missing (validation)', async () => {
    routeMock.params = { tenant: '' }

    const { default: Projects } = await import('./Projects.vue')
    const wrapper = mount(Projects)
    await flushPromises()
    await nextTick()

    const errEl = wrapper.find('p.text-red-600')
    expect(errEl.exists()).toBe(true)
    expect(errEl.text()).toContain('缺少租户ID')
    expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBeUndefined()
    expect(apiFetchMock).not.toHaveBeenCalled()
  })

  it('gitlab sync modal props 透传 errorTraceId（OAuth 启动失败可挂 data-traceId）', async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => [],
    })
    hoisted.gitlabSync.error.value = '网络错误，无法启动 GitLab 授权'
    hoisted.gitlabSync.errorTraceId.value = 'tid-gitlab-sync-oauth-wire'
    hoisted.gitlabSync.showModal.value = true

    const { default: Projects } = await import('./Projects.vue')
    const wrapper = mount(Projects)
    await flushPromises()
    await nextTick()

    const modal = wrapper.findComponent({ name: 'GitlabSyncProjectsModal' })
    expect(modal.exists()).toBe(true)
    expect(modal.props('error')).toContain('无法启动 GitLab 授权')
    expect(modal.props('errorTraceId')).toBe('tid-gitlab-sync-oauth-wire')

    hoisted.gitlabSync.error.value = ''
    hoisted.gitlabSync.errorTraceId.value = ''
    hoisted.gitlabSync.showModal.value = false
  })
})

}
