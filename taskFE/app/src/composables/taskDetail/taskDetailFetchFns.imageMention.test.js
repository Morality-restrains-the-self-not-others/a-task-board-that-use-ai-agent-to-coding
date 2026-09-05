// @vitest-environment node
/**
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.imageMention.test.js requires vitest runtime')
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
      blockCommentRunIfGitOauthUnbound: vi.fn(async () => false),
    }
  })

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { showRequestError } = await import('../../utils/requestErrorDisplay.js')
  const { blockCommentRunIfGitOauthUnbound } = await import('../../utils/gitOauthPushPrecheck.js')
  const { submitComment } = await import('./taskDetailFetchFns.js')
  const {
    registerCommentHardwarePanelReader,
  } = await import('./commentRunHardwareTemplate.js')

  describe('submitComment image mentions', () => {
    beforeEach(async () => {
      vi.clearAllMocks()
      blockCommentRunIfGitOauthUnbound.mockResolvedValue(false)
      registerCommentHardwarePanelReader(() => null)
      const { resetCommentRepoIdentityDraft } = await import('./commentRepoIdentityDraft.js')
      resetCommentRepoIdentityDraft()
    })

    it('posts mentions when pendingImageMention is set', async () => {
      apiFetch.mockResolvedValue({ ok: true })
      const { setPendingImageMention, pendingImageMention } = await import('./commentImageMentionState.js')
      setPendingImageMention({ id: 'img-1', name: 'coder' })
      const newComment = ref('please fix')
      const commentComposerChips = ref([])
      const fetchTaskDetail = vi.fn().mockResolvedValue(undefined)

      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment,
        commentComposerChips,
        buildCommentBodyWithChips: () => 'please fix',
        fetchTaskDetail,
      })

      expect(apiFetch).toHaveBeenCalledTimes(1)
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.content).toBe('please fix')
      expect(body.mentions).toEqual([
        { type: 'installed_image', id: 'img-1', name: 'coder' },
      ])
      expect(pendingImageMention.value).toBeNull()
      expect(fetchTaskDetail).toHaveBeenCalled()
    })

    it('posts mentions.skill when pending mention includes skill', async () => {
      apiFetch.mockResolvedValue({ ok: true })
      const { setPendingImageMention } = await import('./commentImageMentionState.js')
      setPendingImageMention({ id: 'img-1', name: 'coder', skill: 'k8s-debug' })
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('$coder /k8s-debug'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => '$coder /k8s-debug',
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
      })
      const body = JSON.parse(apiFetch.mock.calls[0][1].body)
      expect(body.mentions).toEqual([
        { type: 'installed_image', id: 'img-1', name: 'coder', skill: 'k8s-debug' },
      ])
    })

    it('omits mentions for plain comments', async () => {
      apiFetch.mockResolvedValue({ ok: true })
      const { clearPendingImageMention } = await import('./commentImageMentionState.js')
      clearPendingImageMention()
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('hello'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => 'hello',
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
      })
      const body = JSON.parse(apiFetch.mock.calls[0][1].body)
      expect(body.mentions).toBeUndefined()
      expect(body.server_run_template).toBeUndefined()
    })

    it('includes server_run_template when temporary hardware is complete', async () => {
      apiFetch.mockResolvedValue({ ok: true })
      const { setPendingImageMention } = await import('./commentImageMentionState.js')
      setPendingImageMention({ id: 'img-1', name: 'coder' })
      registerCommentHardwarePanelReader(() => ({
        hardwareConfigSource: 'temporary',
        buildRunTemplatePayload: () => ({ cloud_platform_id: 'plat-1', region: 'cn-hangzhou' }),
      }))
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('please fix'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => 'please fix',
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
      })
      const body = JSON.parse(apiFetch.mock.calls[0][1].body)
      expect(body.mentions).toEqual([
        { type: 'installed_image', id: 'img-1', name: 'coder' },
      ])
      expect(body.server_run_template).toEqual({
        cloud_platform_id: 'plat-1',
        region: 'cn-hangzhou',
      })
    })

    it('omits server_run_template in project template mode', async () => {
      apiFetch.mockResolvedValue({ ok: true })
      const { setPendingImageMention } = await import('./commentImageMentionState.js')
      setPendingImageMention({ id: 'img-1', name: 'coder' })
      registerCommentHardwarePanelReader(() => ({
        getHardwareConfigSource: () => 'project_template',
        buildRunTemplatePayload: () => ({ cloud_platform_id: 'plat-1', region: 'cn-hangzhou' }),
      }))
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('please fix'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => 'please fix',
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
      })
      const body = JSON.parse(apiFetch.mock.calls[0][1].body)
      expect(body.server_run_template).toBeUndefined()
    })

    it('still posts when $镜像 but git identity is missing if oauth is bound', async () => {
      apiFetch.mockResolvedValue({ ok: true })
      const { setPendingImageMention } = await import('./commentImageMentionState.js')
      const { setCommentRepoIdentityDraft } = await import('./commentRepoIdentityDraft.js')
      setPendingImageMention({ id: 'img-1', name: 'coder' })
      setCommentRepoIdentityDraft([
        { repo_url: 'https://github.com/a/b.git', git_identity_id: '', github_user_id: '' },
      ])
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('please fix'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => 'please fix',
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
        taskProjectsWithDetails: ref([
          { project: { git_repos: ['https://github.com/a/b.git'] } },
        ]),
      })
      expect(showRequestError).not.toHaveBeenCalled()
      expect(apiFetch).toHaveBeenCalledTimes(1)
      const body = JSON.parse(apiFetch.mock.calls[0][1].body)
      expect(body.mentions).toEqual([
        { type: 'installed_image', id: 'img-1', name: 'coder' },
      ])
      expect(body.repo_identities).toEqual([
        {
          repo_url: 'https://github.com/a/b.git',
          git_identity_id: '',
          github_user_id: '',
          oauth_gitsite: 'github.com',
        },
      ])
    })

    it('does not post when $镜像 and git oauth is unbound', async () => {
      blockCommentRunIfGitOauthUnbound.mockResolvedValueOnce(true)
      const { setPendingImageMention } = await import('./commentImageMentionState.js')
      setPendingImageMention({ id: 'img-1', name: 'coder' })
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('please fix'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => 'please fix',
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
        taskProjectsWithDetails: ref([
          { project: { git_repos: ['https://github.com/a/b.git'] } },
        ]),
      })
      expect(apiFetch).not.toHaveBeenCalled()
      expect(blockCommentRunIfGitOauthUnbound).toHaveBeenCalled()
    })

    it('still posts when temporary hardware is incomplete and omits partial template', async () => {
      apiFetch.mockResolvedValue({ ok: true })
      const { setPendingImageMention } = await import('./commentImageMentionState.js')
      setPendingImageMention({ id: 'img-1', name: 'coder' })
      registerCommentHardwarePanelReader(() => ({
        hardwareConfigSource: 'temporary',
        buildRunTemplatePayload: () => ({ cloud_platform_id: 'plat-1' }),
      }))
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('please fix'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => 'please fix',
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
      })
      expect(showRequestError).not.toHaveBeenCalled()
      expect(apiFetch).toHaveBeenCalledTimes(1)
      const body = JSON.parse(apiFetch.mock.calls[0][1].body)
      expect(body.content).toBe('please fix')
      expect(body.mentions).toEqual([
        { type: 'installed_image', id: 'img-1', name: 'coder' },
      ])
      expect(body.server_run_template).toBeUndefined()
    })
  })
}
