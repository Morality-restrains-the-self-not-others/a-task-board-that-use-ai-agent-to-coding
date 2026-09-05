// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailLayerActions.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))
  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('../../utils/gitOauthPushPrecheck.js', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      gitOauthUnboundReasonForRepoUrls: vi.fn(async () => ''),
    }
  })
  vi.mock('../../utils/commentOAuthGrantCheck.js', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      commentOAuthGrantMissingReason: vi.fn(() => ''),
    }
  })

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { showRequestError } = await import('../../utils/requestErrorDisplay.js')
  const { gitOauthUnboundReasonForRepoUrls } = await import('../../utils/gitOauthPushPrecheck.js')
  const { onLayerGraphLayerPush, onLayerGraphLayerMerge, onLayerGraphLayerSubmitAndPush } = await import('./taskDetailLayerActions.js')

  describe('onLayerGraphLayerPush — OAuth precheck', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      gitOauthUnboundReasonForRepoUrls.mockResolvedValue('')
      vi.stubGlobal('window', { alert: vi.fn(), open: vi.fn() })
    })

    it('does not block push when GitHub OAuth flag is false but clone identity is selected', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ ok: true }),
      })

      const node = { layerId: 'layer-1' }
      await onLayerGraphLayerPush(node, {
        containerEndpointRegistered: ref(true),
        containerHttpUnreachable: ref(false),
        containerPageUrl: ref('http://localhost:8765/ui/layer-1'),
        taskRepoRows: ref([{ url: 'http://localhost:8012/ljy/somanyad.git' }]),
        repoCloneIdentityIdForUrl: () => 'identity-1',
        firstTaskRepoCloneIdentityId: () => 'identity-1',
        layerGraphPushTargetBranch: ref('main'),
        layerGraphBusyActionKey: ref(''),
        effectiveTenantId: ref('tenant-1'),
        effectiveWorkspaceId: ref('workspace-1'),
        effectiveTaskId: ref('task-1'),
        markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
        markContainerTransportOk: vi.fn(),
        refreshLayerGraphFromServer: vi.fn().mockResolvedValue(undefined),
        refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
        layerGraphSnapshot: ref({
          layers: [
            {
              layer_id: 'layer-1',
              git_worktree_dirty: false,
              git_remote: { is_git: true, ahead: 1, no_upstream: false },
            },
          ],
          jobs: [],
        }),
      })

      expect(window.alert).not.toHaveBeenCalled()
      expect(apiFetch).toHaveBeenCalledTimes(1)
      const [url, options] = apiFetch.mock.calls[0]
      expect(String(url)).toContain('container-layer-git-push')
      expect(JSON.parse(options.body)).toMatchObject({
        layer_id: 'layer-1',
        identity_id: 'identity-1',
        target_branch: 'main',
        wait_for_pr: true,
        allow_bare_git_push: false,
        prefer_container_remote: false,
        repo_url: 'http://localhost:8012/ljy/somanyad.git',
      })
    })

    it('两评论场景 push URL 与 body 带 comment_id', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ ok: true }),
      })
      await onLayerGraphLayerPush(
        { layerId: 'layer-1' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref('http://localhost:8765/ui/layer-1'),
          taskRepoRows: ref([{ url: 'http://localhost:8012/ljy/somanyad.git' }]),
          repoCloneIdentityIdForUrl: () => 'identity-1',
          firstTaskRepoCloneIdentityId: () => 'identity-1',
          layerGraphPushTargetBranch: ref('main'),
          layerGraphBusyActionKey: ref(''),
          effectiveTenantId: ref('tenant-1'),
          effectiveWorkspaceId: ref('workspace-1'),
          effectiveTaskId: ref('task-1'),
          displayComments: ref([{ id: 'cmt-a', commentKind: 'ai' }]),
          activeContainerAgentId: ref(''),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer: vi.fn().mockResolvedValue(undefined),
          refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
          fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
          layerGraphSnapshot: ref({ layers: [], jobs: [] }),
        },
      )
      const [url, options] = apiFetch.mock.calls[0]
      expect(String(url)).toContain('/comment_id/cmt-a/')
      expect(JSON.parse(options.body).comment_id).toBe('cmt-a')
    })

    it('multi-repo passes github repo_url hint and does not allow bare git push', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ ok: true }),
      })
      await onLayerGraphLayerPush(
        { layerId: 'layer-multi' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref('http://host:8765/ui/x'),
          taskRepoRows: ref([
            { url: 'https://github.com/acme/a.git' },
            { url: 'https://github.com/acme/b.git' },
          ]),
          repoCloneIdentityIdForUrl: () => 'identity-1',
          firstTaskRepoCloneIdentityId: () => 'identity-1',
          layerGraphPushTargetBranch: ref('feat/x'),
          layerGraphBusyActionKey: ref(''),
          effectiveTenantId: ref('tenant-1'),
          effectiveWorkspaceId: ref('workspace-1'),
          effectiveTaskId: ref('task-1'),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer: vi.fn().mockResolvedValue(undefined),
          refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
          fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
          layerGraphSnapshot: ref({ layers: [], jobs: [] }),
        },
      )
      const body = JSON.parse(apiFetch.mock.calls[0][1].body)
      expect(body.prefer_container_remote).toBe(true)
      expect(body.allow_bare_git_push).toBe(false)
      expect(body.identity_id).toBe('identity-1')
      expect(body.repo_url).toBe('https://github.com/acme/a.git')
    })

    it('stores pr_html_url and does not auto-open after push succeeds', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          ok: true,
          github_pull_request: { html_url: 'https://github.com/acme/repo/pull/42' },
        }),
      })

      const layerGraphSnapshot = ref({
        layers: [
          {
            layer_id: 'layer-pr',
            git_worktree_dirty: false,
            git_remote: { is_git: true, ahead: 1, no_upstream: false },
          },
        ],
        jobs: [],
      })

      await onLayerGraphLayerPush(
        { layerId: 'layer-pr' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref('http://localhost:8765/ui/layer-pr'),
          taskRepoRows: ref([{ url: 'https://github.com/acme/repo.git' }]),
          repoCloneIdentityIdForUrl: () => 'identity-1',
          firstTaskRepoCloneIdentityId: () => 'identity-1',
          layerGraphPushTargetBranch: ref('main'),
          layerGraphBusyActionKey: ref(''),
          effectiveTenantId: ref('tenant-1'),
          effectiveWorkspaceId: ref('workspace-1'),
          effectiveTaskId: ref('task-1'),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer: vi.fn().mockResolvedValue(undefined),
          refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
          fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
          layerGraphSnapshot,
          displayComments: ref([{ id: 'cmt-exec', commentKind: 'user' }]),
        },
      )

      expect(window.open).not.toHaveBeenCalled()
      expect(layerGraphSnapshot.value.layers[0].git_remote.pr_html_url).toBe(
        'https://github.com/acme/repo/pull/42',
      )
      const commentCall = apiFetch.mock.calls.find(([u]) => String(u).includes('/comments/'))
      expect(commentCall).toBeTruthy()
      expect(String(commentCall[0])).toContain(
        '/api/tenant_id/tenant-1/workspaceId/workspace-1/tasks/task-1/comments/cmt-exec/',
      )
      const commentBody = JSON.parse(commentCall[1].body)
      expect(commentBody).toMatchObject({
        content: 'https://github.com/acme/repo/pull/42',
        execution_mode: 'independent',
        git_pr: { html_url: 'https://github.com/acme/repo/pull/42', provider: 'github' },
      })
      expect(commentBody.parent_comment_id).toBeUndefined()
    })

    it('clears push busy state before layer graph refresh runs', async () => {
      const refreshLayerGraphFromServer = vi.fn(async () => {
        expect(busy.value).toBe('')
      })
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ ok: true, github_pull_request: { skipped: 'no_github_repo' } }),
      })

      const busy = ref('')
      await onLayerGraphLayerPush(
        { layerId: 'layer-2' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref('http://localhost:8765/ui/layer-2'),
          taskRepoRows: ref([{ url: 'http://localhost:8012/ljy/somanyad.git' }]),
          repoCloneIdentityIdForUrl: () => 'identity-1',
          firstTaskRepoCloneIdentityId: () => 'identity-1',
          layerGraphPushTargetBranch: ref('main'),
          layerGraphBusyActionKey: busy,
          effectiveTenantId: ref('tenant-1'),
          effectiveWorkspaceId: ref('workspace-1'),
          effectiveTaskId: ref('task-1'),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer,
          refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
          fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
          layerGraphSnapshot: ref({
            layers: [
              {
                layer_id: 'layer-2',
                git_worktree_dirty: false,
                git_remote: { is_git: true, ahead: 2, no_upstream: false },
              },
            ],
            jobs: [],
          }),
        },
      )

      expect(busy.value).toBe('')
      expect(refreshLayerGraphFromServer).toHaveBeenCalled()
    })

    it('push success sets ahead=0 and last_pushed_count for zTree label', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ ok: true }),
      })
      const layerGraphSnapshot = ref({
        layers: [
          {
            layer_id: 'layer-3',
            git_worktree_dirty: false,
            git_remote: { is_git: true, ahead: 1, no_upstream: false },
          },
        ],
        jobs: [],
      })
      await onLayerGraphLayerPush(
        { layerId: 'layer-3' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref('http://localhost:8765/ui/layer-3'),
          taskRepoRows: ref([{ url: 'http://localhost:8012/ljy/somanyad.git' }]),
          repoCloneIdentityIdForUrl: () => 'identity-1',
          firstTaskRepoCloneIdentityId: () => 'identity-1',
          layerGraphPushTargetBranch: ref('main'),
          layerGraphBusyActionKey: ref(''),
          effectiveTenantId: ref('tenant-1'),
          effectiveWorkspaceId: ref('workspace-1'),
          effectiveTaskId: ref('task-1'),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer: vi.fn().mockResolvedValue(undefined),
          refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
          fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
          layerGraphSnapshot,
        },
      )
      const gr = layerGraphSnapshot.value.layers[0].git_remote
      expect(gr.ahead).toBe(0)
      expect(gr.last_pushed_count).toBe(1)
    })
  })

  describe('onLayerGraphLayerSubmitAndPush — PR attach', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      gitOauthUnboundReasonForRepoUrls.mockResolvedValue('')
      vi.stubGlobal('window', { alert: vi.fn(), open: vi.fn() })
    })

    it('stores pr_html_url after submit-and-push succeeds', async () => {
      apiFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ ok: true }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            ok: true,
            github_pull_request: { html_url: 'https://github.com/acme/repo/pull/99' },
          }),
        })
        .mockResolvedValue({ ok: true, json: async () => ({ id: 'cmt_pr' }) })

      const layerGraphSnapshot = ref({
        layers: [
          {
            layer_id: 'layer-sp',
            git_worktree_dirty: true,
            git_remote: { is_git: true, ahead: 0, no_upstream: false },
          },
        ],
        jobs: [],
      })

      await onLayerGraphLayerSubmitAndPush(
        { layerId: 'layer-sp' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref('http://localhost:8765/ui/layer-sp'),
          taskRepoRows: ref([{ url: 'https://github.com/acme/repo.git' }]),
          repoCloneIdentityIdForUrl: () => 'identity-1',
          firstTaskRepoCloneIdentityId: () => 'identity-1',
          layerGraphPushTargetBranch: ref('main'),
          layerGitGithubAppOauthConnected: ref(true),
          layerGraphBusyActionKey: ref(''),
          effectiveTenantId: ref('tenant-1'),
          effectiveWorkspaceId: ref('workspace-1'),
          effectiveTaskId: ref('task-1'),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer: vi.fn(async () => {
            // after commit refresh: clean + ahead so push proceeds
            layerGraphSnapshot.value = {
              layers: [
                {
                  layer_id: 'layer-sp',
                  git_worktree_dirty: false,
                  git_remote: { is_git: true, ahead: 1, no_upstream: false },
                },
              ],
              jobs: [],
            }
          }),
          refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
          fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
          layerGraphSnapshot,
          layerChangesByLayerId: ref({}),
          displayComments: ref([{ id: 'cmt-exec', commentKind: 'user' }]),
        },
      )

      expect(layerGraphSnapshot.value.layers[0].git_remote.pr_html_url).toBe(
        'https://github.com/acme/repo/pull/99',
      )
      expect(window.open).not.toHaveBeenCalled()
      const commentCall = apiFetch.mock.calls.find(([u]) => String(u).includes('/comments/'))
      expect(String(commentCall[0])).toContain(
        '/api/tenant_id/tenant-1/workspaceId/workspace-1/tasks/task-1/comments/cmt-exec/',
      )
    })

    it('does not commit or push when GitHub OAuth is unbound', async () => {
      gitOauthUnboundReasonForRepoUrls.mockResolvedValueOnce(
        '提交并推送前，请先完成 GitHub/GitLab OAuth 授权，否则无法推送到 HTTPS 远端',
      )
      const layerGraphSnapshot = ref({
        layers: [
          {
            layer_id: 'layer-sp',
            git_worktree_dirty: true,
            git_remote: { is_git: true, ahead: 0, no_upstream: false },
          },
        ],
        jobs: [],
      })
      await onLayerGraphLayerSubmitAndPush(
        { layerId: 'layer-sp' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref('http://localhost:8765/ui/layer-sp'),
          taskRepoRows: ref([{ url: 'https://github.com/acme/repo.git' }]),
          repoCloneIdentityIdForUrl: () => 'identity-1',
          firstTaskRepoCloneIdentityId: () => 'identity-1',
          layerGraphPushTargetBranch: ref('main'),
          layerGitGithubAppOauthConnected: ref(false),
          layerGraphBusyActionKey: ref(''),
          effectiveTenantId: ref('tenant-1'),
          effectiveWorkspaceId: ref('workspace-1'),
          effectiveTaskId: ref('task-1'),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer: vi.fn(),
          refreshZTreeExecutionLog: vi.fn(),
          fetchTaskDetail: vi.fn(),
          layerGraphSnapshot,
          layerChangesByLayerId: ref({}),
        },
      )
      expect(apiFetch).not.toHaveBeenCalled()
      expect(showRequestError).toHaveBeenCalledWith(
        expect.stringMatching(/提交并推送前/),
      )
    })
  })

  describe('onLayerGraphLayerMerge', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      vi.stubGlobal('window', { alert: vi.fn(), open: vi.fn() })
    })

    it('posts merge with target_branch from merge target', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ ok: true, status: 'merged', summary: 'ok' }),
      })
      const refresh = vi.fn().mockResolvedValue(undefined)
      await onLayerGraphLayerMerge(
        { layerId: 'layer-m1' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref('http://localhost:8765/ui'),
          layerGraphMergeTargetBranch: ref('develop'),
          layerGraphBusyActionKey: ref(''),
          effectiveTenantId: ref('tenant-1'),
          effectiveWorkspaceId: ref('workspace-1'),
          effectiveTaskId: ref('task-1'),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer: refresh,
          refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
        },
      )
      expect(apiFetch).toHaveBeenCalled()
      const [url, opts] = apiFetch.mock.calls[0]
      expect(url).toContain('container-layer-git-merge')
      const body = JSON.parse(opts.body)
      expect(body.layer_id).toBe('layer-m1')
      expect(body.target_branch).toBe('develop')
      expect(refresh).toHaveBeenCalled()
    })

    it('alerts when merge target missing', async () => {
      await onLayerGraphLayerMerge(
        { layerId: 'layer-m1' },
        {
          containerEndpointRegistered: ref(true),
          containerHttpUnreachable: ref(false),
          containerPageUrl: ref(''),
          layerGraphMergeTargetBranch: ref(''),
          layerGraphBusyActionKey: ref(''),
          effectiveTenantId: ref('t'),
          effectiveWorkspaceId: ref('w'),
          effectiveTaskId: ref('task'),
          markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
          markContainerTransportOk: vi.fn(),
          refreshLayerGraphFromServer: vi.fn(),
          refreshZTreeExecutionLog: vi.fn(),
        },
      )
      expect(apiFetch).not.toHaveBeenCalled()
      expect(window.alert).toHaveBeenCalled()
    })
  })
}
