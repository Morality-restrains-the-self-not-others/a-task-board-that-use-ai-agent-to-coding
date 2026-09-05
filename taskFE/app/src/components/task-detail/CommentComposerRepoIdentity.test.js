// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CommentComposerRepoIdentity.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    routeMock: { path: '/t', query: {}, hash: '' },
  }))

  vi.mock('../../utils/apiUtils.js', () => ({ apiFetch: hoisted.apiFetch }))
  vi.mock('../../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: async () => 'u1',
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
    useRouter: () => ({ replace: vi.fn() }),
  }))
  vi.mock('../../utils/githubAppReturnStorage.js', () => ({
    createGithubAppReturnKey: () => 'a'.repeat(32),
    setGithubAppReturnTarget: () => {},
  }))

  const { default: CommentComposerRepoIdentity } = await import('./CommentComposerRepoIdentity.vue')
  const { rememberGrantTicket } = await import('../../utils/grantTicketSession.js')
  const {
    readCommentRepoIdentityDraft,
    resetCommentRepoIdentityDraft,
  } = await import('../../composables/taskDetail/commentRepoIdentityDraft.js')

  describe('CommentComposerRepoIdentity', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      resetCommentRepoIdentityDraft()
      sessionStorage.clear()
      document.cookie = 'userId=u1; path=/'
      hoisted.apiFetch.mockImplementation(async (url) => {
        const path = String(url || '')
        if (path.includes('/git-identities/')) {
          return {
            ok: true,
            json: async () => ({ identities: [{ id: 'gid-1', git_user_name: 'Ann', git_user_email: 'a@b.c' }] }),
          }
        }
        if (path.includes('/github-credential-status/')) {
          return {
            ok: true,
            json: async () => ({
              github_connections: [{ connected: true, github_user_id: 9, github_login: 'ann' }],
            }),
          }
        }
        return { ok: true, json: async () => ({ connected: true, connections: [{ connected: true }] }) }
      })
    })

    it('无关联仓库时不渲染', async () => {
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: { tenantId: 't1', workspaceId: 'w1', taskId: 'task1', taskProjectsWithDetails: [] },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="comment-composer-repo-identity"]').exists()).toBe(false)
    })

    it('按仓库渲染 Git 身份与 GitHub 账号选择', async () => {
      rememberGrantTicket('tkt-1', 'https://github.com/acme/demo.git')
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            {
              project: { git_repos: ['https://github.com/acme/demo.git'] },
            },
          ],
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="comment-composer-repo-identity"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="comment-repo-git-identity-select"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="comment-repo-github-account-select"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('https://github.com/acme/demo.git')
      const hint = wrapper.get('[data-testid="comment-composer-git-oauth-hint"]')
      expect(hint.attributes('data-kind')).toBe('bound')
      expect(hint.text()).toMatch(/已绑定 Git OAuth/)
    })

    it('http(s) 仓库地址渲染为可跳转外链，非 http(s) 保持纯文本', async () => {
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            {
              project: { git_repos: ['https://gitlab.daydaymoney.com/acme/demo.git', 'git@github.com:acme/ssh-only.git'] },
            },
          ],
        },
      })
      await flushPromises()
      const link = wrapper.get('[data-testid="comment-composer-repo-url"]')
      expect(link.attributes('href')).toBe('https://gitlab.daydaymoney.com/acme/demo.git')
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toContain('noopener')
      expect(wrapper.text()).toContain('git@github.com:acme/ssh-only.git')
      const rows = wrapper.findAll('[data-testid="comment-composer-repo-identity-row"]')
      expect(rows.length).toBe(2)
    })

    it('$镜像未绑定 Git OAuth 时提示且不挡住发送，设置入口为真实 href', async () => {
      hoisted.apiFetch.mockImplementation(async (url) => {
        const path = String(url || '')
        if (path.includes('/git-identities/')) {
          return { ok: true, json: async () => ({ identities: [] }) }
        }
        if (path.includes('/github-credential-status/')) {
          return { ok: true, json: async () => ({ github_connections: [] }) }
        }
        if (path.includes('/git-oauth/user-app-connection/')) {
          return { ok: true, json: async () => ({ connected: false, connections: [] }) }
        }
        return { ok: true, json: async () => ({ connected: false, connections: [] }) }
      })
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            { project: { git_repos: ['https://github.com/acme/demo.git'] } },
          ],
        },
      })
      await flushPromises()
      await nextTick()
      const hint = wrapper.get('[data-testid="comment-composer-git-oauth-hint"]')
      expect(hint.attributes('data-kind')).toBe('unbound')
      expect(hint.text()).toMatch(/使用授权/)
      expect(hint.text()).toMatch(/提交并运行/)
      expect(hint.text()).not.toMatch(/仍可发送评论/)
      const link = wrapper.get('[data-testid="comment-composer-git-oauth-settings-link"]')
      expect(link.element.tagName).toBe('A')
      expect(link.attributes('href')).toContain('/api/git-oauth/github-start-from-gateway/')
      expect(decodeURIComponent(String(link.attributes('href') || ''))).toContain('https://github.com/acme/demo.git')
      expect(link.attributes('href')).not.toContain('git-site-oauth')
      expect(link.attributes('href')).not.toContain('/user/')
      const rowBind = wrapper.get('[data-testid="comment-repo-oauth-bind-btn"]')
      expect(rowBind.element.tagName).toBe('A')
      expect(rowBind.attributes('href')).toContain('/api/git-oauth/github-start-from-gateway/')
    })

    it('gitlab 仓未绑定入口指向 gitlab-start-from-gateway', async () => {
      hoisted.apiFetch.mockImplementation(async (url) => {
        const path = String(url || '')
        if (path.includes('/git-identities/')) {
          return { ok: true, json: async () => ({ identities: [] }) }
        }
        return { ok: true, json: async () => ({ connected: false, connections: [] }) }
      })
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            { project: { git_repos: ['https://gitlab.daydaymoney.com/group/repo.git'] } },
          ],
        },
      })
      await flushPromises()
      await nextTick()
      const link = wrapper.get('[data-testid="comment-composer-git-oauth-settings-link"]')
      expect(link.attributes('href')).toContain('/api/git-oauth/gitlab-start-from-gateway/')
      expect(decodeURIComponent(String(link.attributes('href') || ''))).toContain('https://gitlab.daydaymoney.com/group/repo.git')
      expect(link.attributes('href')).not.toContain('git-site-oauth')
    })

    it('从最近一条评论 repo_identities 预填 Git 身份与 GitHub 账号', async () => {
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            { project: { git_repos: ['https://github.com/acme/demo.git'] } },
          ],
          comments: [
            { id: 'c-old', repo_identities: [] },
            {
              id: 'c-new',
              repo_identities: [
                {
                  repo_url: 'https://github.com/acme/demo.git',
                  git_identity_id: 'gid-1',
                  github_user_id: '9',
                },
              ],
            },
          ],
        },
      })
      await flushPromises()
      await nextTick()
      const gitSelect = wrapper.get('[data-testid="comment-repo-git-identity-select"]')
      expect(gitSelect.element.value).toBe('gid-1')
      const ghSelect = wrapper.get('[data-testid="comment-repo-github-account-select"]')
      expect(ghSelect.element.value).toBe('9')
      const draft = readCommentRepoIdentityDraft()
      expect(draft).toEqual([
        {
          repo_url: 'https://github.com/acme/demo.git',
          git_identity_id: 'gid-1',
          github_user_id: '9',
        },
      ])
    })

    it('无评论身份时回退任务级 taskRepoIdentities 仅预填 Git 身份', async () => {
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            { project: { git_repos: ['https://github.com/acme/demo.git'] } },
          ],
          comments: [{ id: 'c1', repo_identities: [] }],
          taskRepoIdentities: [
            { repo_url: 'https://github.com/acme/demo.git', git_identity_id: 'gid-1' },
          ],
        },
      })
      await flushPromises()
      await nextTick()
      const gitSelect = wrapper.get('[data-testid="comment-repo-git-identity-select"]')
      expect(gitSelect.element.value).toBe('gid-1')
      const draft = readCommentRepoIdentityDraft()
      expect(draft).toEqual([
        {
          repo_url: 'https://github.com/acme/demo.git',
          git_identity_id: 'gid-1',
          github_user_id: '',
        },
      ])
    })

    it('无任何预填来源时保持空选择，不写草稿身份', async () => {
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            { project: { git_repos: ['https://github.com/acme/demo.git'] } },
          ],
          comments: [{ id: 'c1', repo_identities: [] }],
        },
      })
      await flushPromises()
      await nextTick()
      const gitSelect = wrapper.get('[data-testid="comment-repo-git-identity-select"]')
      expect(gitSelect.element.value).toBe('')
      const draft = readCommentRepoIdentityDraft()
      expect(draft).toEqual([
        {
          repo_url: 'https://github.com/acme/demo.git',
          git_identity_id: '',
          github_user_id: '',
        },
      ])
    })

    it('源码使用真实 a[href] 指向仓库对应 git OAuth start，不用 click.prevent 导航', () => {
      const root = dirname(fileURLToPath(import.meta.url))
      const src = readFileSync(join(root, 'CommentComposerRepoIdentity.vue'), 'utf8')
      expect(src).toMatch(/data-testid="comment-composer-git-oauth-settings-link"/)
      expect(src).toMatch(/:href="oauthStartHref\(url\)"/)
      expect(src).not.toMatch(/git-site-oauth/)
      expect(src).not.toMatch(/@click\.prevent/)
      expect(src).not.toMatch(/<router-link/)
    })

    it('有 project_id 时在身份行展示自动克隆子仓库开关，默认跟随项目字段', async () => {
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            {
              project_id: 'p1',
              project: {
                git_repos: ['https://github.com/acme/demo.git'],
                auto_clone_nested_repos: false,
              },
            },
          ],
        },
      })
      await flushPromises()
      const row = wrapper.get('[data-testid="comment-composer-repo-identity-row"]')
      const toggle = row.get('[data-testid="task-nested-repos-auto-clone-toggle"]')
      expect(toggle.element.checked).toBe(false)
      expect(row.get('[data-testid="task-nested-repos-auto-clone-off-hint"]').text()).toContain(
        '关闭后容器将仅克隆父仓库',
      )
    })

    it('切换自动克隆开关时 PUT 项目 auto_clone_nested_repos', async () => {
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const path = String(url || '')
        if (path.includes('/git-identities/')) {
          return {
            ok: true,
            json: async () => ({ identities: [{ id: 'gid-1', git_user_name: 'Ann', git_user_email: 'a@b.c' }] }),
          }
        }
        if (path.includes('/github-credential-status/')) {
          return {
            ok: true,
            json: async () => ({
              github_connections: [{ connected: true, github_user_id: 9, github_login: 'ann' }],
            }),
          }
        }
        if (opts.method === 'PUT' && path.includes('/api/projects/tenant_id/t1/p1/')) {
          return { ok: true, json: async () => ({ auto_clone_nested_repos: true }) }
        }
        return { ok: true, json: async () => ({ connected: true, connections: [{ connected: true }] }) }
      })
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            {
              project_id: 'p1',
              project: {
                git_repos: ['https://github.com/acme/demo.git'],
                auto_clone_nested_repos: false,
              },
            },
          ],
        },
      })
      await flushPromises()
      const toggle = wrapper.get('[data-testid="task-nested-repos-auto-clone-toggle"]')
      await toggle.setValue(true)
      await flushPromises()
      const putCall = hoisted.apiFetch.mock.calls.find((args) => String(args[0] || '').includes('/api/projects/'))
      expect(putCall).toBeTruthy()
      expect(putCall[1]).toEqual(expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ auto_clone_nested_repos: true }),
      }))
      expect(toggle.element.checked).toBe(true)
      expect(wrapper.find('[data-testid="task-nested-repos-auto-clone-off-hint"]').exists()).toBe(false)
    })

    it('保存自动克隆开关失败时展示错误并带 data-traceId', async () => {
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const path = String(url || '')
        if (path.includes('/git-identities/')) {
          return { ok: true, json: async () => ({ identities: [] }) }
        }
        if (opts.method === 'PUT' && path.includes('/api/projects/')) {
          return {
            ok: false,
            traceId: 'tr-auto-clone-1',
            headers: { get: (name) => (String(name).toLowerCase() === 'x-trace-id' ? 'tr-auto-clone-1' : null) },
            json: async () => ({ error: '保存失败', trace_id: 'tr-auto-clone-1' }),
          }
        }
        return { ok: true, json: async () => ({ connected: true, connections: [{ connected: true }] }) }
      })
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            {
              project_id: 'p1',
              project: {
                git_repos: ['https://github.com/acme/demo.git'],
                auto_clone_nested_repos: true,
              },
            },
          ],
        },
      })
      await flushPromises()
      const toggle = wrapper.get('[data-testid="task-nested-repos-auto-clone-toggle"]')
      await toggle.setValue(false)
      await flushPromises()
      const err = wrapper.get('[data-testid="task-nested-repos-auto-clone-save-error"]')
      expect(err.text()).toContain('保存失败')
      expect(err.attributes('data-traceid')).toBe('tr-auto-clone-1')
    })

    it('无 project_id 时不渲染自动克隆开关', async () => {
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            { project: { git_repos: ['https://github.com/acme/demo.git'] } },
          ],
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="comment-composer-repo-identity-row"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="task-nested-repos-auto-clone-toggle"]').exists()).toBe(false)
    })

    it('不因 repoUrls watch 再查 user-app-connection（由 composable 统一查询）', async () => {
      const wrapper = mount(CommentComposerRepoIdentity, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          taskProjectsWithDetails: [
            { project: { git_repos: ['https://github.com/acme/demo.git'] } },
          ],
        },
      })
      await flushPromises()
      const connectionCalls = hoisted.apiFetch.mock.calls.filter(([url]) =>
        String(url || '').includes('/git-oauth/user-app-connection/'),
      )
      expect(connectionCalls.length).toBe(1)
      wrapper.unmount()
    })
  })
}
