// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailLinkedProjectsViewMode.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(async () => ({
      ok: true,
      json: async () => ({ results: [] }),
    })),
  }))

  describe('TaskDetailLinkedProjectsViewMode — 只读展示', () => {
    const baseProps = {
      tenantId: '875326170655125504',
      taskProjectsWithDetails: [
        {
          id: 'proj-1',
          project_id: 'p1',
          project_name: 'Demo',
          project: { git_repos: ['https://github.com/demo/repo-a.git'] },
          repo_branches: { 'https://github.com/demo/repo-a.git': 'main' },
        },
      ],
      staleRepoSyncLoading: false,
    }

    it('展示项目名、仓库 URL 与基准分支，不含身份或进度控件', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsViewMode.vue')).default
      const wrapper = mount(Comp, {
        props: baseProps,
        global: { stubs: { 'router-link': { template: '<a><slot /></a>' } } },
      })
      expect(wrapper.get('[data-testid="task-linked-projects-display"]').text()).toContain('Demo')
      expect(wrapper.get('[data-testid="task-linked-repo-row"]').text()).toContain('https://github.com/demo/repo-a.git')
      expect(wrapper.get('[data-testid="task-linked-repo-row"]').text()).toContain('main')
      expect(wrapper.find('[data-testid="repo-git-commit-identity-section"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="repo-github-oauth-section"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="task-repo-oauth-bind-btn"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('保存账号')
      expect(wrapper.text()).not.toContain('重新克隆')
      expect(wrapper.text()).not.toContain('Git 提交身份')
      expect(wrapper.find('[data-testid="task-nested-repos-auto-clone-toggle"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="task-linked-repo-git-probe-badge"]').exists()).toBe(true)
    })

    it('无关联项目时展示空态', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsViewMode.vue')).default
      const wrapper = mount(Comp, {
        props: { ...baseProps, taskProjectsWithDetails: [] },
      })
      expect(wrapper.text()).toContain('暂无关联项目')
    })

    it('details 为空时从任务 GET 的 projects 回退展示仓库，不显示空态', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsViewMode.vue')).default
      const wrapper = mount(Comp, {
        props: {
          ...baseProps,
          taskProjectsWithDetails: [],
          fallbackApiProjects: [{
            project_id: 'proj_881195029417193472',
            repo_index: 0,
            base_branch: 'master',
            project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
            stored_repo_address: 'https://github.com/ruandao/helloworld',
            repo_address_mismatch: true,
          }],
          workspaceProjects: [],
        },
        global: { stubs: { 'router-link': { template: '<a><slot /></a>' } } },
      })
      expect(wrapper.text()).not.toContain('暂无关联项目')
      expect(wrapper.get('[data-testid="task-linked-repo-url"]').text())
        .toContain('https://github.com/test-ruandao/helloworld.git')
    })

    it('工作区目录缺失时仍展示任务 API 带回的仓库 URL，不显示空态', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsViewMode.vue')).default
      const wrapper = mount(Comp, {
        props: {
          ...baseProps,
          taskProjectsWithDetails: [
            {
              id: 'proj_881195029417193472',
              project_id: 'proj_881195029417193472',
              project_name: '',
              project_missing: true,
              project: { git_repos: ['https://github.com/test-ruandao/helloworld.git'] },
              repo_branches: { 'https://github.com/test-ruandao/helloworld.git': 'master' },
              repo_address_mismatch: true,
              stored_repo_address: 'https://github.com/ruandao/helloworld',
            },
          ],
        },
        global: { stubs: { 'router-link': { template: '<a><slot /></a>' } } },
      })
      expect(wrapper.text()).not.toContain('暂无关联项目')
      expect(wrapper.get('[data-testid="task-linked-project-missing"]').text()).toBe('项目已删除')
      expect(wrapper.find('[data-testid="task-linked-project-name-link"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="task-linked-repo-url"]').text()).toContain('https://github.com/test-ruandao/helloworld.git')
      expect(wrapper.get('[data-testid="task-repo-address-mismatch-badge"]').text()).toBe('仓库地址已变更')
    })

    it('http(s) 仓库地址渲染为可跳转外链', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsViewMode.vue')).default
      const wrapper = mount(Comp, {
        props: baseProps,
        global: { stubs: { 'router-link': { template: '<a><slot /></a>' } } },
      })
      const link = wrapper.get('[data-testid="task-linked-repo-url"]')
      expect(link.element.tagName).toBe('A')
      expect(link.attributes('href')).toBe('https://github.com/demo/repo-a.git')
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toContain('noopener')
      expect(link.attributes('title')).toBe('https://github.com/demo/repo-a.git')
      expect(link.text()).toContain('https://github.com/demo/repo-a.git')
    })

    it('非 http(s) 仓库地址保持纯文本，不生成 javascript: 链接', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsViewMode.vue')).default
      const wrapper = mount(Comp, {
        props: {
          ...baseProps,
          taskProjectsWithDetails: [
            {
              id: 'proj-1',
              project_id: 'p1',
              project_name: 'Demo',
              project: { git_repos: ['javascript:alert(1)', 'git@github.com:demo/repo.git'] },
              repo_branches: {},
            },
          ],
        },
        global: { stubs: { 'router-link': { template: '<a><slot /></a>' } } },
      })
      expect(wrapper.find('[data-testid="task-linked-repo-url"]').exists()).toBe(false)
      const rows = wrapper.findAll('[data-testid="task-linked-repo-row"]')
      expect(rows).toHaveLength(2)
      expect(rows[0].text()).toContain('javascript:alert(1)')
      expect(rows[1].text()).toContain('git@github.com:demo/repo.git')
      expect(wrapper.find('a[href^="javascript:"]').exists()).toBe(false)
    })

    it('仓库地址已变更时展示徽章与同步按钮', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsViewMode.vue')).default
      const wrapper = mount(Comp, {
        props: {
          ...baseProps,
          taskProjectsWithDetails: [
            {
              id: 'proj-1',
              project_id: 'p1',
              project_name: 'helloworld',
              project: { git_repos: ['https://github.com/test-ruandao/helloworld.git'] },
              repo_branches: { 'https://github.com/test-ruandao/helloworld.git': 'master' },
              repo_address_mismatch: true,
              stored_repo_address: 'https://github.com/ruandao/helloworld',
            },
          ],
        },
        global: { stubs: { 'router-link': { template: '<a><slot /></a>' } } },
      })
      const badge = wrapper.get('[data-testid="task-repo-address-mismatch-badge"]')
      expect(badge.text()).toBe('仓库地址已变更')
      expect(badge.attributes('title')).toContain('https://github.com/ruandao/helloworld')
      expect(badge.attributes('title')).toContain('https://github.com/test-ruandao/helloworld.git')
      expect(badge.attributes('title')).toContain('重新克隆')
      expect(wrapper.get('[data-testid="task-repo-address-sync-btn"]').text()).toContain('同步仓库地址')
    })
  })
}
