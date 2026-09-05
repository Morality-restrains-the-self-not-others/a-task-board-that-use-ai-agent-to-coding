// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentsSection.git-identity-summary.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const here = dirname(fileURLToPath(import.meta.url))

  describe('评论执行细节 Git 身份摘要接线', () => {
    it('CommentsSection 把 comment.repo_identities 与 gitIdentityOptions 传给执行细节', () => {
      const section = readFileSync(join(here, 'TaskDetailCommentsSection.vue'), 'utf8')
      expect(section).toMatch(/:repo-identities="Array\.isArray\(comment\.repo_identities\) \? comment\.repo_identities : \[\]"/)
      expect(section).toMatch(/:git-identity-options="gitIdentityOptions"/)
      expect(section).toMatch(/gitIdentityOptions:\s*\{\s*type:\s*Array/)
    })

    it('CommentsSection 把任务级回退身份与 repoCloneIdentityByUrl 传给执行细节（OPT-20260821-028）', () => {
      const section = readFileSync(join(here, 'TaskDetailCommentsSection.vue'), 'utf8')
      expect(section).toMatch(/:fallback-repo-identities="taskRepoIdentities"/)
      expect(section).toMatch(/repoCloneIdentityByUrl:/)
      expect(section).toMatch(/resolveRepoCloneIdentityFromMap/)
    })

    it('bindings 把 layerGitIdentityOptions 透传到评论区 gitIdentityOptions', () => {
      const bindings = readFileSync(
        join(here, '../../views/taskDetailSectionBindings.js'),
        'utf8',
      )
      const commentsFn = bindings.split('export function useTaskDetailCommentsSectionBindings')[1] || ''
      expect(commentsFn).toMatch(/layerGitIdentityOptions/)
      expect(commentsFn).toMatch(/gitIdentityOptions:\s*Array\.isArray\(layerGitIdentityOptions\?\.value\)/)
      const detail = readFileSync(join(here, '../../views/TaskDetail.vue'), 'utf8')
      expect(detail).toMatch(/layerPanelStore, layerGitIdentityOptions,/)
    })

    it('CommentsSection 把探测后的 Git OAuth 就绪状态传给执行细节摘要', () => {
      const section = readFileSync(join(here, 'TaskDetailCommentsSection.vue'), 'utf8')
      expect(section).toMatch(/:oauth-readiness="oauthReadinessForDetails"/)
      expect(section).toMatch(/:last-push-error="lastPushErrorFor\(comment\.id\)"/)
      expect(section).toMatch(/useCommentGitOauthAccessTokenProbe/)
      expect(section).toMatch(/repoOAuthReadiness:\s*\{\s*type:\s*Object/)
      const bindings = readFileSync(
        join(here, '../../views/taskDetailSectionBindings.js'),
        'utf8',
      )
      expect(bindings).toMatch(/repoOAuthReadiness:\s*repoOAuthReadiness\?\.value/)
      const detail = readFileSync(join(here, '../../views/TaskDetail.vue'), 'utf8')
      expect(detail).toMatch(/useTaskDetailCommentsSectionBindings\(\{[\s\S]*repoOAuthReadiness/)
      expect(detail).toMatch(/TaskDetailCommentsSection[\s\S]*@repo-oauth-readiness="onRepoOAuthReadiness"/)
      const panel = readFileSync(join(here, 'TaskDetailCommentsPanel.vue'), 'utf8')
      expect(panel).toMatch(/:oauth-readiness="oauthReadiness"/)
      const feed = readFileSync(join(here, 'TaskDetailConversationFeed.vue'), 'utf8')
      expect(feed).toMatch(/:oauth-readiness="oauthReadiness"/)
    })
  })
}
