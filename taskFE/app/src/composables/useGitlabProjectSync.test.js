// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] useGitlabProjectSync.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

const apiFetchMock = vi.hoisted(() => vi.fn())

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => apiFetchMock(...args),
}))

  const { computed, defineComponent, nextTick, ref } = await import('vue')
  const { mount } = await import('@vue/test-utils')
  const { default: GitlabSyncProjectsModal } = await import('../components/GitlabSyncProjectsModal.vue')
  const { useGitlabProjectSync } = await import('./useGitlabProjectSync.js')

const MOCK_REPOS = [
  {
    gitlab_project_id: '1',
    name: 'valuestream',
    path_with_namespace: 'example-user/valuestream',
    description: '',
    http_url_to_repo: 'http://127.0.0.1:8012/example-user/valuestream.git',
    web_url: 'http://127.0.0.1:8012/example-user/valuestream',
    default_branch: 'main',
    imported_in_single_repo_project: false,
    imported_in_any_project: false,
  },
  {
    gitlab_project_id: '2',
    name: 'other-repo',
    path_with_namespace: 'example-user/other-repo',
    description: '',
    http_url_to_repo: 'http://127.0.0.1:8012/example-user/other-repo.git',
    web_url: 'http://127.0.0.1:8012/example-user/other-repo',
    default_branch: 'main',
    imported_in_single_repo_project: false,
    imported_in_any_project: false,
  },
]

const MOCK_REPOS_WITH_OCCUPIED = [
  ...MOCK_REPOS,
  {
    gitlab_project_id: '3',
    name: 'occupied-repo',
    path_with_namespace: 'example-user/occupied-repo',
    description: '',
    http_url_to_repo: 'http://127.0.0.1:8012/example-user/occupied-repo.git',
    web_url: 'http://127.0.0.1:8012/example-user/occupied-repo',
    default_branch: 'main',
    imported_in_single_repo_project: true,
    imported_in_any_project: true,
  },
]

function jsonResponse(body, ok = true, { status, headers = {} } = {}) {
  const headerMap = new Map(
    Object.entries(headers).map(([k, v]) => [String(k).toLowerCase(), String(v)]),
  )
  return {
    ok,
    status: status ?? (ok ? 200 : 500),
    headers: {
      get(name) {
        return headerMap.get(String(name).toLowerCase()) || null
      },
    },
    json: async () => body,
  }
}

function seedPurchasedRegion(sync, webUrl = 'http://127.0.0.1:8012', slug = 'local-gitlab') {
  sync.purchasedRegions.value = [
    {
      region: slug,
      region_name: '本地 GitLab',
      gitlab_web_url: webUrl,
      provisioning_status: 'active',
    },
  ]
  sync.selectedRegionSlug.value = slug
  sync.gitlabWebsite.value = webUrl
}

describe('useGitlabProjectSync', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
    window.localStorage.clear()
  })

  it('loadPurchasedRegions 多区域时优先恢复上次选用的区域（localStorage）', async () => {
    window.localStorage.setItem('gitlab-sync-region:850256677331562496', 'tencent-sh-1')
    apiFetchMock.mockImplementation(async (url) => {
      const path = String(url)
      if (path.includes('/billing/gitlab-resources/')) {
        return jsonResponse({
          resources: [
            {
              region: 'tencent-sh-5',
              region_name: '腾讯五区',
              gitlab_web_url: 'https://gl5.example',
              provisioning_status: 'active',
            },
            {
              region: 'tencent-sh-1',
              region_name: '腾讯一区',
              gitlab_web_url: 'https://gl1.example',
              provisioning_status: 'active',
            },
          ],
        })
      }
      if (path.includes('/workspaces/')) {
        return jsonResponse([{ id: '1', name: 'ws' }])
      }
      if (path.includes('/gitlab-remote-repos/')) {
        expect(path).toContain(encodeURIComponent('https://gl1.example'))
        return jsonResponse({
          oauth_bound: true,
          gitlab_website: 'https://gl1.example',
          gitlab_login: 'u1',
          repos: MOCK_REPOS,
          error: null,
        })
      }
      return jsonResponse({})
    })

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    await sync.openModal()
    await nextTick()

    expect(sync.purchasedRegions.value).toHaveLength(2)
    expect(sync.selectedRegionSlug.value).toBe('tencent-sh-1')
    expect(sync.gitlabWebsite.value).toBe('https://gl1.example')
    expect(sync.repos.value).toHaveLength(2)
  })

  it('selectRegion 后把所选区域写入 localStorage，按 tenantId 隔离', async () => {
    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    sync.purchasedRegions.value = [
      {
        region: 'a',
        region_name: 'A',
        gitlab_web_url: 'https://a.example',
        provisioning_status: 'active',
      },
      {
        region: 'b',
        region_name: 'B',
        gitlab_web_url: 'https://b.example',
        provisioning_status: 'active',
      },
    ]
    sync.selectedRegionSlug.value = 'a'
    sync.gitlabWebsite.value = 'https://a.example'

    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        oauth_bound: true,
        gitlab_website: 'https://b.example',
        repos: MOCK_REPOS.slice(0, 1),
        error: null,
      }),
    )

    await sync.selectRegion('b')
    expect(sync.selectedRegionSlug.value).toBe('b')
    expect(window.localStorage.getItem('gitlab-sync-region:850256677331562496')).toBe('b')
    expect(window.localStorage.getItem('gitlab-sync-region:other-tenant')).toBeNull()
  })

  it('localStorage 中的区域已失效时回退到列表首项并修正存储', async () => {
    window.localStorage.setItem('gitlab-sync-region:850256677331562496', 'retired-region')
    apiFetchMock.mockImplementation(async (url) => {
      const path = String(url)
      if (path.includes('/billing/gitlab-resources/')) {
        return jsonResponse({
          resources: [
            {
              region: 'tencent-sh-5',
              region_name: '腾讯五区',
              gitlab_web_url: 'https://gl5.example',
              provisioning_status: 'active',
            },
          ],
        })
      }
      if (path.includes('/workspaces/')) {
        return jsonResponse([{ id: '1', name: 'ws' }])
      }
      if (path.includes('/gitlab-remote-repos/')) {
        return jsonResponse({
          oauth_bound: true,
          gitlab_website: 'https://gl5.example',
          gitlab_login: 'u1',
          repos: MOCK_REPOS,
          error: null,
        })
      }
      return jsonResponse({})
    })

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    await sync.openModal()
    await nextTick()

    expect(sync.selectedRegionSlug.value).toBe('tencent-sh-5')
    expect(window.localStorage.getItem('gitlab-sync-region:850256677331562496')).toBe(
      'tencent-sh-5',
    )
  })

  it('loadRemoteRepos 成功后 loading 变为 false 并默认全选', async () => {
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        oauth_bound: true,
        gitlab_website: 'http://127.0.0.1:8012',
        gitlab_login: 'example-user',
        provider_key: 'gitlab:daydaymoney-gitlab',
        repos: MOCK_REPOS,
        error: null,
      }),
    )

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    seedPurchasedRegion(sync)

    await sync.loadRemoteRepos()
    await nextTick()

    expect(sync.loading.value).toBe(false)
    expect(sync.oauthBound.value).toBe(true)
    expect(sync.repos.value).toHaveLength(2)
    expect(sync.selectedCount.value).toBe(2)
    const remoteUrl = String(apiFetchMock.mock.calls[0][0])
    expect(remoteUrl).toContain('gitlab_host=')
    expect(remoteUrl).toContain(encodeURIComponent('http://127.0.0.1:8012'))
  })

  it('openModal 先拉已购区域；无可用 URL 时不请求 remote-repos', async () => {
    apiFetchMock.mockImplementation(async (url) => {
      const path = String(url)
      if (path.includes('/billing/gitlab-resources/')) {
        return jsonResponse({
          resources: [
            {
              region: 'aliyun-pending',
              region_name: '待挂载',
              gitlab_web_url: '',
              provisioning_status: 'pending_admin',
            },
          ],
        })
      }
      if (path.includes('/workspaces/')) {
        return jsonResponse([{ id: '1', name: 'ws' }])
      }
      throw new Error(`unexpected fetch ${path}`)
    })

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    await sync.openModal()
    await nextTick()

    expect(sync.purchasedRegions.value).toHaveLength(0)
    expect(sync.showModal.value).toBe(true)
    expect(
      apiFetchMock.mock.calls.some((c) => String(c[0]).includes('/gitlab-remote-repos/')),
    ).toBe(false)
  })

  it('openModal 多区域时默认选第一项并带 gitlab_host 拉仓库', async () => {
    apiFetchMock.mockImplementation(async (url) => {
      const path = String(url)
      if (path.includes('/billing/gitlab-resources/')) {
        return jsonResponse({
          resources: [
            {
              region: 'tencent-sh-5',
              region_name: '腾讯五区',
              gitlab_web_url: 'https://gl5.example',
              provisioning_status: 'active',
            },
            {
              region: 'tencent-sh-1',
              region_name: '腾讯一区',
              gitlab_web_url: 'https://gl1.example',
              provisioning_status: 'active',
            },
          ],
        })
      }
      if (path.includes('/workspaces/')) {
        return jsonResponse([{ id: '1', name: 'ws' }])
      }
      if (path.includes('/gitlab-remote-repos/')) {
        expect(path).toContain(encodeURIComponent('https://gl5.example'))
        return jsonResponse({
          oauth_bound: true,
          gitlab_website: 'https://gl5.example',
          gitlab_login: 'u1',
          repos: MOCK_REPOS,
          error: null,
        })
      }
      return jsonResponse({})
    })

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    await sync.openModal()
    await nextTick()

    expect(sync.purchasedRegions.value).toHaveLength(2)
    expect(sync.selectedRegionSlug.value).toBe('tencent-sh-5')
    expect(sync.gitlabWebsite.value).toBe('https://gl5.example')
    expect(sync.repos.value).toHaveLength(2)
  })

  it('selectRegion 切换后重新请求对应 gitlab_host', async () => {
    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    sync.purchasedRegions.value = [
      {
        region: 'a',
        region_name: 'A',
        gitlab_web_url: 'https://a.example',
        provisioning_status: 'active',
      },
      {
        region: 'b',
        region_name: 'B',
        gitlab_web_url: 'https://b.example',
        provisioning_status: 'active',
      },
    ]
    sync.selectedRegionSlug.value = 'a'
    sync.gitlabWebsite.value = 'https://a.example'

    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        oauth_bound: true,
        gitlab_website: 'https://b.example',
        repos: MOCK_REPOS.slice(0, 1),
        error: null,
      }),
    )

    await sync.selectRegion('b')
    expect(sync.selectedRegionSlug.value).toBe('b')
    expect(String(apiFetchMock.mock.calls[0][0])).toContain(
      encodeURIComponent('https://b.example'),
    )
    expect(sync.repos.value).toHaveLength(1)
  })

  it('computed modal props 在 loading 结束后会展示仓库列表', async () => {
    let resolveRemote
    const remotePromise = new Promise((resolve) => {
      resolveRemote = resolve
    })
    apiFetchMock.mockReturnValueOnce(remotePromise)

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    seedPurchasedRegion(sync)
    const modalProps = computed(() => ({
      show: true,
      loading: sync.loading.value,
      creating: sync.creating.value,
      error: sync.error.value,
      oauthBound: sync.oauthBound.value,
      gitlabWebsite: sync.gitlabWebsite.value,
      gitlabLogin: sync.gitlabLogin.value,
      purchasedRegions: sync.purchasedRegions.value,
      selectedRegionSlug: sync.selectedRegionSlug.value,
      regionsLoading: sync.regionsLoading.value,
      repos: sync.repos.value,
      selectedCount: sync.selectedCount.value,
      batchEligibleCount: sync.batchEligibleCount.value,
      allSelectableChecked: sync.allSelectableChecked.value,
      selectedRepoKeys: sync.selectedRepoKeys.value,
      workspaces: sync.workspaces.value,
      selectedWorkspaceId: sync.selectedWorkspaceId.value,
      createResult: sync.createResult.value,
      combinedProjectName: sync.combinedProjectName.value,
    }))

    const wrapper = mount(GitlabSyncProjectsModal, {
      props: modalProps.value,
    })

    const loadPromise = sync.loadRemoteRepos()
    await nextTick()
    wrapper.setProps(modalProps.value)
    await nextTick()
    expect(wrapper.text()).toContain('正在加载 GitLab 仓库')

    resolveRemote(
      jsonResponse({
        oauth_bound: true,
        gitlab_website: 'http://127.0.0.1:8012',
        gitlab_login: 'example-user',
        repos: MOCK_REPOS,
        error: null,
      }),
    )
    await loadPromise
    await nextTick()
    wrapper.setProps(modalProps.value)
    await nextTick()

    expect(sync.loading.value).toBe(false)
    expect(wrapper.text()).not.toContain('正在加载 GitLab 仓库')
    expect(wrapper.text()).toContain('valuestream')
  })

  it('静态 snapshot props 不会随 loading 结束而更新（回归说明）', async () => {
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        oauth_bound: true,
        gitlab_website: 'http://127.0.0.1:8012',
        gitlab_login: 'example-user',
        repos: MOCK_REPOS,
        error: null,
      }),
    )

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    seedPurchasedRegion(sync)
    sync.loading.value = true

    const wrapper = mount(GitlabSyncProjectsModal, {
      props: {
        show: true,
        loading: sync.loading.value,
        creating: false,
        error: '',
        oauthBound: false,
        gitlabWebsite: '',
        gitlabLogin: '',
        purchasedRegions: sync.purchasedRegions.value,
        selectedRegionSlug: sync.selectedRegionSlug.value,
        regionsLoading: false,
        repos: [],
        selectedCount: 0,
        batchEligibleCount: 0,
        allSelectableChecked: false,
        selectedRepoKeys: {},
        workspaces: [],
        selectedWorkspaceId: '',
        createResult: null,
        combinedProjectName: '',
      },
    })

    await sync.loadRemoteRepos()
    await nextTick()

    expect(sync.loading.value).toBe(false)
    expect(wrapper.text()).toContain('正在加载 GitLab 仓库')
  })

  it('用户手动修改合并项目名称后不再被自动覆盖', async () => {
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        oauth_bound: true,
        gitlab_website: 'http://127.0.0.1:8012',
        gitlab_login: 'example-user',
        repos: MOCK_REPOS,
        error: null,
      }),
    )

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    seedPurchasedRegion(sync)

    await sync.loadRemoteRepos()
    expect(sync.combinedProjectName.value).toBe('example-user')

    sync.setCombinedProjectName('my-custom-name')
    sync.toggleRepo(MOCK_REPOS[0].http_url_to_repo, false)
    sync.toggleRepo(MOCK_REPOS[0].http_url_to_repo, true)
    expect(sync.combinedProjectName.value).toBe('my-custom-name')
  })

  it('选中 2 个仓库时建议合并项目名称并调用 combined API', async () => {
    apiFetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          oauth_bound: true,
          gitlab_website: 'http://127.0.0.1:8012',
          gitlab_login: 'example-user',
          provider_key: 'gitlab:daydaymoney-gitlab',
          repos: MOCK_REPOS,
          error: null,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          created: {
            id: '888001',
            name: 'example-user',
            repos: MOCK_REPOS.map((r) => ({ http_url_to_repo: r.http_url_to_repo })),
          },
          skipped: [],
          errors: [],
        }),
      )

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    seedPurchasedRegion(sync)

    await sync.loadRemoteRepos()
    sync.selectedWorkspaceId.value = '900001'
    expect(sync.selectedCount.value).toBe(2)
    expect(sync.combinedProjectName.value).toBe('example-user')

    await sync.createCombinedProject()

    expect(apiFetchMock).toHaveBeenLastCalledWith(
      '/api/projects/combined-from-gitlab-repos/tenant_id/850256677331562496/',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"name":"example-user"'),
      }),
    )
    expect(sync.showModal.value).toBe(false)
  })

  it('gitlabOAuthStartHref 由 providerKey/区域直接拼等价 start API href（OPT-20260904-003）', async () => {
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        oauth_bound: false,
        gitlab_website: 'https://gitlab.example',
        provider_key: 'gitlab:daydaymoney-gitlab',
        repos: [],
      }),
    )

    const tenantId = ref('882864422450655232')
    const sync = useGitlabProjectSync({ tenantId })
    seedPurchasedRegion(sync, 'https://gitlab.example')
    await sync.loadRemoteRepos()

    const href = sync.gitlabOAuthStartHref.value
    expect(href).toContain('/api/git-oauth/gitlab-start-from-gateway/')
    expect(href).toContain('service_provider=daydaymoney-gitlab')
    expect(href).toContain(`repo_url=${encodeURIComponent('https://gitlab.example')}`)
    expect(href).toContain('next=')
    // 真实 <a> 导航：不再发 fetch 取 authorize_url
    expect(apiFetchMock).not.toHaveBeenCalledWith(
      expect.stringContaining('/api/git-oauth/gitlab-start-from-gateway/'),
      expect.objectContaining({ headers: { Accept: 'application/json' } }),
    )
  })

  it('未绑定 OAuth 时「前往 GitLab 授权」渲染为真实 <a href>（OPT-20260904-003）', () => {
    const wrapper = mount(GitlabSyncProjectsModal, {
      props: {
        show: true,
        loading: false,
        creating: false,
        error: '',
        oauthBound: false,
        gitlabWebsite: 'https://gitlab.example',
        gitlabLogin: '',
        oauthStartHref:
          '/api/git-oauth/gitlab-start-from-gateway/?repo_url=https%3A%2F%2Fgitlab.example&next=%2Ftenant%2F882864422450655232%2Fprojects%2F',
        purchasedRegions: [
          {
            region: 'a',
            region_name: 'A',
            gitlab_web_url: 'https://gitlab.example',
            provisioning_status: 'active',
          },
        ],
        selectedRegionSlug: 'a',
        regionsLoading: false,
        repos: [],
        selectedCount: 0,
        batchEligibleCount: 0,
        allSelectableChecked: false,
        selectedRepoKeys: {},
        workspaces: [],
        selectedWorkspaceId: '',
        createResult: null,
        combinedProjectName: '',
      },
    })
    const btn = wrapper.get('[data-testid="gitlab-sync-oauth-btn"]')
    expect(btn.element.tagName).toBe('A')
    expect(btn.attributes('href')).toContain('/api/git-oauth/gitlab-start-from-gateway/')
  })

  it('模态错误区仍挂 data-traceId（回归：OAuth 改 <a> 后保留错误展示）', () => {
    const wrapper = mount(GitlabSyncProjectsModal, {
      props: {
        show: true,
        loading: false,
        creating: false,
        error: '无法启动 GitLab 授权（bad_state）',
        errorTraceId: 'tid-oauth-fail-1',
        oauthBound: false,
        gitlabWebsite: '',
        gitlabLogin: '',
        repos: [],
        selectedCount: 0,
        batchEligibleCount: 0,
        allSelectableChecked: false,
        selectedRepoKeys: {},
        workspaces: [],
        selectedWorkspaceId: '',
        createResult: null,
        combinedProjectName: '',
      },
    })
    const alert = wrapper.get('[role="alert"]')
    expect(alert.text()).toContain('无法启动 GitLab 授权')
    expect(alert.attributes('data-traceid') || alert.attributes('data-traceId')).toBe(
      'tid-oauth-fail-1',
    )
  })

  it('单仓已占用仓库可合并但不可批量创建', async () => {
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        oauth_bound: true,
        gitlab_website: 'http://127.0.0.1:8012',
        gitlab_login: 'example-user',
        repos: MOCK_REPOS_WITH_OCCUPIED,
        error: null,
      }),
    )

    const tenantId = ref('850256677331562496')
    const sync = useGitlabProjectSync({ tenantId })
    seedPurchasedRegion(sync)

    await sync.loadRemoteRepos()

    expect(sync.selectedCount.value).toBe(3)
    expect(sync.batchEligibleCount.value).toBe(2)
    expect(sync.selectedReposForMerge.value).toHaveLength(3)
    expect(sync.selectedReposForBatch.value).toHaveLength(2)
  })

  it('取消全选或取消单个勾选后，弹窗「已选」计数与按钮括号同步更新', async () => {
    apiFetchMock.mockImplementation(async (url) => {
      const path = String(url)
      if (path.includes('/billing/gitlab-resources/')) {
        return jsonResponse({
          resources: [
            {
              region: 'local-gitlab',
              region_name: '本地 GitLab',
              gitlab_web_url: 'http://127.0.0.1:8012',
              provisioning_status: 'active',
            },
          ],
        })
      }
      if (path.includes('/workspaces/')) {
        return jsonResponse([{ id: '900001', name: 'ws' }])
      }
      if (path.includes('/gitlab-remote-repos/')) {
        return jsonResponse({
          oauth_bound: true,
          gitlab_website: 'http://127.0.0.1:8012',
          gitlab_login: 'example-user',
          repos: MOCK_REPOS_WITH_OCCUPIED,
          error: null,
        })
      }
      return jsonResponse({})
    })

    const Parent = defineComponent({
      components: { GitlabSyncProjectsModal },
      setup() {
        const tenantId = ref('850256677331562496')
        const sync = useGitlabProjectSync({ tenantId })
        const gitlabSyncModalProps = computed(() => ({
          show: true,
          loading: sync.loading.value,
          creating: sync.creating.value,
          error: sync.error.value,
          oauthBound: sync.oauthBound.value,
          gitlabWebsite: sync.gitlabWebsite.value,
          gitlabLogin: sync.gitlabLogin.value,
          purchasedRegions: sync.purchasedRegions.value,
          selectedRegionSlug: sync.selectedRegionSlug.value,
          regionsLoading: sync.regionsLoading.value,
          repos: sync.repos.value,
          selectedCount: sync.selectedCount.value,
          batchEligibleCount: sync.batchEligibleCount.value,
          allSelectableChecked: sync.allSelectableChecked.value,
          selectedRepoKeys: sync.selectedRepoKeys.value,
          workspaces: sync.workspaces.value,
          selectedWorkspaceId: sync.selectedWorkspaceId.value,
          createResult: sync.createResult.value,
          combinedProjectName: sync.combinedProjectName.value,
        }))
        return { sync, gitlabSyncModalProps }
      },
      template: `
        <GitlabSyncProjectsModal
          v-bind="gitlabSyncModalProps"
          @toggle-repo="sync.toggleRepo"
          @toggle-select-all="sync.toggleSelectAll"
        />
      `,
    })

    const wrapper = mount(Parent)
    await wrapper.vm.sync.openModal()
    await nextTick()
    await nextTick()

    const selectedCountEl = () => wrapper.get('[data-testid="gitlab-sync-selected-count"]')
    expect(selectedCountEl().text()).toContain('已选 3 / 3')
    expect(wrapper.text()).toContain('合并为一个项目（3）')
    expect(wrapper.text()).toContain('每个仓库各建一个项目（2）')

    await wrapper.get('[data-testid="gitlab-sync-select-all"]').setValue(false)
    await nextTick()

    expect(wrapper.vm.sync.selectedCount.value).toBe(0)
    expect(selectedCountEl().text()).toContain('已选 0 / 3')
    expect(wrapper.text()).toContain('合并为一个项目（0）')
    expect(wrapper.text()).toContain('每个仓库各建一个项目（0）')

    await wrapper.get('[data-testid="gitlab-sync-select-all"]').setValue(true)
    await nextTick()
    expect(selectedCountEl().text()).toContain('已选 3 / 3')

    const rows = wrapper.findAll('li')
    await rows[0].trigger('click')
    await nextTick()

    expect(wrapper.vm.sync.selectedCount.value).toBe(2)
    expect(selectedCountEl().text()).toContain('已选 2 / 3')
    expect(wrapper.text()).toContain('合并为一个项目（2）')
  })
})
}
