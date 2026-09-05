// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] repro-gitlab-count.test.js requires vitest runtime')
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
    path_with_namespace: 'a/valuestream',
    description: '',
    http_url_to_repo: 'http://example/a/valuestream.git',
    web_url: 'http://example/a/valuestream',
    default_branch: 'main',
    imported_in_single_repo_project: false,
    imported_in_any_project: false,
  },
  {
    gitlab_project_id: '2',
    name: 'other-repo',
    path_with_namespace: 'a/other-repo',
    description: '',
    http_url_to_repo: 'http://example/a/other-repo.git',
    web_url: 'http://example/a/other-repo',
    default_branch: 'main',
    imported_in_single_repo_project: false,
    imported_in_any_project: false,
  },
  {
    gitlab_project_id: '3',
    name: 'third',
    path_with_namespace: 'a/third',
    description: '',
    http_url_to_repo: 'http://example/a/third.git',
    web_url: 'http://example/a/third',
    default_branch: 'main',
    imported_in_single_repo_project: false,
    imported_in_any_project: false,
  },
]

function jsonResponse(body) {
  return { ok: true, status: 200, json: async () => body }
}

function mountParent() {
  const Parent = defineComponent({
    components: { GitlabSyncProjectsModal },
    setup() {
      const tenantId = ref('t1')
      const sync = useGitlabProjectSync({ tenantId })
      const gitlabSyncModalProps = computed(() => ({
        show: true,
        loading: sync.loading.value,
        creating: sync.creating.value,
        error: sync.error.value,
        oauthBound: sync.oauthBound.value,
        gitlabWebsite: sync.gitlabWebsite.value,
        gitlabLogin: sync.gitlabLogin.value,
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
  return mount(Parent)
}

describe('repro selectedCount via v-bind computed', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  it('取消全选后按钮上的 selectedCount 应更新', async () => {
    apiFetchMock
      .mockResolvedValueOnce(jsonResponse([{ id: '900001', name: 'ws' }]))
      .mockResolvedValueOnce(
        jsonResponse({
          oauth_bound: true,
          gitlab_website: 'http://example',
          gitlab_login: 'u',
          repos: MOCK_REPOS,
          error: null,
        }),
      )

    const wrapper = mountParent()
    await wrapper.vm.sync.openModal()
    await nextTick()
    await nextTick()

    expect(wrapper.vm.sync.selectedCount.value).toBe(3)
    expect(wrapper.text()).toContain('合并为一个项目（3）')
    expect(wrapper.text()).toContain('每个仓库各建一个项目（3）')

    const selectAll = wrapper.find('input[type="checkbox"]')
    await selectAll.setValue(false)
    await nextTick()

    expect(wrapper.vm.sync.selectedCount.value).toBe(0)
    expect(wrapper.text()).toContain('合并为一个项目（0）')
    expect(wrapper.text()).toContain('每个仓库各建一个项目（0）')
  })

  it('取消单个勾选后 selectedCount 应递减', async () => {
    apiFetchMock
      .mockResolvedValueOnce(jsonResponse([{ id: '900001', name: 'ws' }]))
      .mockResolvedValueOnce(
        jsonResponse({
          oauth_bound: true,
          gitlab_website: 'http://example',
          gitlab_login: 'u',
          repos: MOCK_REPOS,
          error: null,
        }),
      )

    const wrapper = mountParent()
    await wrapper.vm.sync.openModal()
    await nextTick()
    await nextTick()

    expect(wrapper.text()).toContain('合并为一个项目（3）')

    const rows = wrapper.findAll('li')
    await rows[0].trigger('click')
    await nextTick()

    expect(wrapper.vm.sync.selectedCount.value).toBe(2)
    expect(wrapper.text()).toContain('合并为一个项目（2）')
    expect(wrapper.text()).toContain('每个仓库各建一个项目（2）')
  })
})
}
