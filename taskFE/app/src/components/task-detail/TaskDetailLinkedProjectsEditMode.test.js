// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailLinkedProjectsEditMode.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const repoUrl = 'https://gitlab.daydaymoney.com/org/ram-work'

  const baseProps = {
    editingTask: { linkedProjects: [{ project_id: 'p1' }] },
    workspaceProjects: [{ id: 'p1', name: 'Demo' }],
    workspaceProjectsLoading: false,
    getProjectRepos: () => [repoUrl],
    getRepoBranchValue: () => 'main',
    getRepoBranches: () => ['main'],
    getRepoBranchError: () => '',
    isLoadingRepoBranches: () => false,
    repoCloneFieldId: () => 'repo-1',
    editRepoOAuthActionLoadingByUrl: {},
    shouldShowEditRepoOAuthAuthorize: () => false,
    startEditRepoOAuthConnect: () => {},
    normalizeRepoUrlKey: (u) => String(u || '').trim(),
  }

  describe('TaskDetailLinkedProjectsEditMode — 仓库地址链接', () => {
    it('http(s) 仓库地址渲染为可跳转外链', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsEditMode.vue')).default
      const wrapper = mount(Comp, { props: baseProps })
      const link = wrapper.get('[data-testid="task-linked-repo-url"]')
      expect(link.element.tagName).toBe('A')
      expect(link.attributes('href')).toBe(repoUrl)
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toContain('noopener')
    })
  })
}
