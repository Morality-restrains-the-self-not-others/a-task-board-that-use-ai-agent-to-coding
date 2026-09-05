// @vitest-environment jsdom
/**
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */

if (!process.env.VITEST) {
  console.log('[skip] ProjectDetailGitReposSection.nested-oauth.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')

  // vi.mock() 工厂在 Vitest 中被 hoist 到文件顶部执行，早于 dynamic import。
  // 所有在 mock 工厂中引用的变量必须通过 vi.hoisted() 定义。
  const Mock = vi.hoisted(() => {
    const { ref } = require('vue')

    const statusByUrl = {
      'https://gitlab.daydaymoney.com/g/parent.git': 'token_available',
      'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git': 'token_available',
      'https://gitlab.daydaymoney.com/g/docs.git': 'not_bound',
      'https://gitlab.daydaymoney.com/g/runAll.git': 'token_available',
    }

    const refreshGitReposOAuthStatus = vi.fn(async () => {})

    const nestedReposRef = ref([
      { path: 'DaydaymoneyGrafana', url: 'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git', source: 'gitmodules' },
      { path: 'docs', url: 'https://gitlab.daydaymoney.com/g/docs.git', source: 'gitmodules' },
      { path: 'runAll', url: 'https://gitlab.daydaymoney.com/g/runAll.git', source: 'gitmodules' },
    ])
    const nestedLoadingRef = ref(false)
    const nestedErrorRef = ref('')
    const nestedErrorTraceIdRef = ref('')

    return {
      statusByUrl,
      refreshGitReposOAuthStatus,
      nestedReposRef,
      nestedLoadingRef,
      nestedErrorRef,
      nestedErrorTraceIdRef,
    }
  })

  const { mount, flushPromises } = await import('@vue/test-utils')
  const { ref } = await import('vue')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1', id: 'p1' }, query: {} }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  }))

  vi.mock('../utils/gitSiteOAuthCallbackUtils.js', () => ({
    applyOAuthCallbackFromRoute: vi.fn(async () => null),
  }))

  vi.mock('../composables/useProjectDetailGitRepos.js', () => {
    const { ref } = require('vue')
    return {
    useProjectDetailGitRepos: () => ({
      router: { push: vi.fn(), replace: vi.fn() },
      projectGitRepoList: ref(['https://gitlab.daydaymoney.com/g/parent.git']),
      projectGitRepoEntries: ref([
        { url: 'https://gitlab.daydaymoney.com/g/parent.git', cloneAlias: '' },
      ]),
      branchPreviewLoading: ref(false),
      branchPreviewError: ref(''),
      repoBranchPreviews: ref([]),
      fetchProjectRepoBranchesPreview: vi.fn(),
      bootstrapGitOAuthCatalog: vi.fn(async () => {}),
      applyGitReposStatusFromApi: vi.fn(),
      refreshGitReposOAuthStatus: Mock.refreshGitReposOAuthStatus,
      isRepoOAuthActionDisabled: () => false,
      repoOAuthButtonLabel: () => 'OAuth 授权',
      shouldShowRepoOAuthButton: (url) => Mock.statusByUrl[url] === 'not_bound',
      repoOAuthErrorByUrl: () => '',
      repoOAuthErrorTraceIdByUrl: () => '',
      repoOAuthTokenStatus: (url) => Mock.statusByUrl[url] || '',
      repoOAuthStatusLabel: (url) => {
        const status = Mock.statusByUrl[url]
        if (status === 'token_available') return '已授权'
        if (status === 'not_bound') return '未授权'
        return '检查中…'
      },
      repoOAuthStatusBadgeClass: () => 'border-gray-200 bg-gray-50 text-gray-700',
      startRepoOAuthConnect: vi.fn(),
    }),
  }})

  vi.mock('../composables/useProjectNestedGitRepos.js', () => ({
    useProjectNestedGitRepos: () => ({
      nestedRepos: Mock.nestedReposRef,
      nestedLoading: Mock.nestedLoadingRef,
      nestedError: Mock.nestedErrorRef,
      nestedErrorTraceId: Mock.nestedErrorTraceIdRef,
      fetchNestedGitRepos: vi.fn(),
    }),
  }))

  const { default: ProjectDetailGitReposSection } = await import('./ProjectDetailGitReposSection.vue')

  const _n = Mock.nestedReposRef
  const _ne = Mock.nestedErrorRef
  const _nt = Mock.nestedErrorTraceIdRef
  const _s = Mock.statusByUrl
  const _r = Mock.refreshGitReposOAuthStatus

  describe('ProjectDetailGitReposSection nested oauth', () => {
    it('父仓授权异常时发出 auto-run-git-gate blocked', async () => {
      _s['https://gitlab.daydaymoney.com/g/parent.git'] = 'token_error'
      _ne.value = ''
      _n.value = []
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            git_repos: ['https://gitlab.daydaymoney.com/g/parent.git'],
            git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/parent.git' }],
            git_repos_status: [],
          },
        },
      })
      await flushPromises()
      const gates = wrapper.emitted('auto-run-git-gate') || []
      expect(gates.length).toBeGreaterThan(0)
      const last = gates[gates.length - 1][0]
      expect(last.blocked).toBe(true)
      expect(last.code).toBe('AUTO_RUN_GIT_AUTH_ERROR')
      expect(last.displayLabel).toBe('无法启动')
      _s['https://gitlab.daydaymoney.com/g/parent.git'] = 'token_available'
      _n.value = [
        { path: 'DaydaymoneyGrafana', url: 'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git', source: 'gitmodules' },
        { path: 'docs', url: 'https://gitlab.daydaymoney.com/g/docs.git', source: 'gitmodules' },
        { path: 'runAll', url: 'https://gitlab.daydaymoney.com/g/runAll.git', source: 'gitmodules' },
      ]
      wrapper.unmount()
    })

    it('子仓列表获取失败时发出 auto-run-git-gate blocked', async () => {
      _ne.value = '无法获取子 Git 仓库列表：未检测到可用授权。'
      _nt.value = 'tid-nested-oauth-1'
      _n.value = []
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            git_repos: ['https://gitlab.daydaymoney.com/g/parent.git'],
            git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/parent.git' }],
            git_repos_status: [],
          },
        },
      })
      await flushPromises()
      const gates = wrapper.emitted('auto-run-git-gate') || []
      const last = gates[gates.length - 1][0]
      expect(last.blocked).toBe(true)
      expect(last.code).toBe('AUTO_RUN_NESTED_REPOS_UNAVAILABLE')
      expect(last.traceId).toBe('tid-nested-oauth-1')
      const errEl = wrapper.get('[data-testid="project-detail-nested-git-repos-error"]')
      expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe(
        'tid-nested-oauth-1',
      )
      _ne.value = ''
      _nt.value = ''
      _n.value = [
        { path: 'DaydaymoneyGrafana', url: 'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git', source: 'gitmodules' },
        { path: 'docs', url: 'https://gitlab.daydaymoney.com/g/docs.git', source: 'gitmodules' },
        { path: 'runAll', url: 'https://gitlab.daydaymoney.com/g/runAll.git', source: 'gitmodules' },
      ]
      wrapper.unmount()
    })

    it('自动克隆关闭时子仓授权异常不阻断 auto-run-git-gate', async () => {
      _s['https://gitlab.daydaymoney.com/g/parent.git'] = 'token_available'
      _s['https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git'] = 'token_error'
      _ne.value = ''
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            auto_clone_nested_repos: false,
            git_repos: ['https://gitlab.daydaymoney.com/g/parent.git'],
            git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/parent.git' }],
            git_repos_status: [],
          },
        },
      })
      await flushPromises()
      const gates = wrapper.emitted('auto-run-git-gate') || []
      const last = gates[gates.length - 1][0]
      expect(last.blocked).toBe(false)
      expect(last.code).toBe('')
      _s['https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git'] = 'token_available'
      wrapper.unmount()
    })

    it('自动克隆关闭时子仓列表获取失败不阻断 auto-run-git-gate', async () => {
      _s['https://gitlab.daydaymoney.com/g/parent.git'] = 'token_available'
      _ne.value = '无法获取子 Git 仓库列表：未检测到可用授权。'
      _nt.value = 'tid-nested-oauth-off-1'
      _n.value = []
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            auto_clone_nested_repos: false,
            git_repos: ['https://gitlab.daydaymoney.com/g/parent.git'],
            git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/parent.git' }],
            git_repos_status: [],
          },
        },
      })
      await flushPromises()
      const gates = wrapper.emitted('auto-run-git-gate') || []
      const last = gates[gates.length - 1][0]
      expect(last.blocked).toBe(false)
      expect(last.code).toBe('')
      _ne.value = ''
      _nt.value = ''
      _n.value = [
        { path: 'DaydaymoneyGrafana', url: 'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git', source: 'gitmodules' },
        { path: 'docs', url: 'https://gitlab.daydaymoney.com/g/docs.git', source: 'gitmodules' },
        { path: 'runAll', url: 'https://gitlab.daydaymoney.com/g/runAll.git', source: 'gitmodules' },
      ]
      wrapper.unmount()
    })

    it('关闭自动克隆后子仓授权异常不再阻断（开关切换）', async () => {
      _s['https://gitlab.daydaymoney.com/g/parent.git'] = 'token_available'
      _s['https://gitlab.daydaymoney.com/g/docs.git'] = 'token_error'
      _ne.value = ''
      const project = {
        id: 'p1',
        auto_clone_nested_repos: true,
        git_repos: ['https://gitlab.daydaymoney.com/g/parent.git'],
        git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/parent.git' }],
        git_repos_status: [],
      }
      const wrapper = mount(ProjectDetailGitReposSection, { props: { project } })
      await flushPromises()
      let last = (wrapper.emitted('auto-run-git-gate') || []).at(-1)[0]
      expect(last.blocked).toBe(true)
      expect(last.code).toBe('AUTO_RUN_GIT_AUTH_ERROR')

      await wrapper.setProps({
        project: { ...project, auto_clone_nested_repos: false },
      })
      await flushPromises()
      last = (wrapper.emitted('auto-run-git-gate') || []).at(-1)[0]
      expect(last.blocked).toBe(false)
      expect(last.code).toBe('')
      _s['https://gitlab.daydaymoney.com/g/docs.git'] = 'not_bound'
      wrapper.unmount()
    })

    it('自动克隆开启时子仓授权异常发出 auto-run-git-gate blocked', async () => {
      _s['https://gitlab.daydaymoney.com/g/parent.git'] = 'token_available'
      _s['https://gitlab.daydaymoney.com/g/docs.git'] = 'token_error'
      _ne.value = ''
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            auto_clone_nested_repos: true,
            git_repos: ['https://gitlab.daydaymoney.com/g/parent.git'],
            git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/parent.git' }],
            git_repos_status: [],
          },
        },
      })
      await flushPromises()
      const gates = wrapper.emitted('auto-run-git-gate') || []
      const last = gates[gates.length - 1][0]
      expect(last.blocked).toBe(true)
      expect(last.code).toBe('AUTO_RUN_GIT_AUTH_ERROR')
      _s['https://gitlab.daydaymoney.com/g/docs.git'] = 'not_bound'
      wrapper.unmount()
    })

    it('展示子仓库授权状态，并将未授权子仓库置顶', async () => {
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            git_repos: ['https://gitlab.daydaymoney.com/g/parent.git'],
            git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/parent.git' }],
            git_repos_status: [],
          },
        },
      })
      await flushPromises()

      const nestedSection = wrapper.get('[data-testid="project-detail-nested-git-repos"]')
      const rows = nestedSection.findAll('[data-testid="project-detail-nested-git-repo-row"]')
      expect(rows).toHaveLength(3)

      expect(rows[0].text()).toContain('docs')
      expect(rows[0].get('[data-testid="nested-git-repo-oauth-status-0"]').text()).toBe('未授权')
      expect(rows[0].find('[data-testid="nested-git-repo-oauth-button"]').exists()).toBe(true)

      expect(rows[1].get('[data-testid="nested-git-repo-oauth-status-1"]').text()).toBe('已授权')
      expect(rows[2].get('[data-testid="nested-git-repo-oauth-status-2"]').text()).toBe('已授权')

      expect(_r).toHaveBeenCalled()
      const nestedUrlsCall = _r.mock.calls.find((args) =>
        (args[0] || []).includes('https://gitlab.daydaymoney.com/g/docs.git'),
      )
      expect(nestedUrlsCall).toBeTruthy()

      wrapper.unmount()
    })
  })
}
