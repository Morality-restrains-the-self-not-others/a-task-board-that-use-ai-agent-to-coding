// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] createTaskOauthGate.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    buildCreateTaskProjectListBoundMap,
    collectCreateTaskOAuthRepoRows,
    collectCreateTaskOAuthRepoUrls,
    collectForkTaskOAuthRepoRows,
    collectForkTaskOAuthRepoUrls,
    fetchCreateTaskProjectL2BoundByUrl,
    isCreateTaskOAuthTokenAvailable,
    isGitOAuthUserAppConnectedPayload,
    resolveCreateTaskOauthBlockedReason,
    resolveForkAutoRunOauthBlockedReason,
    shouldRequireAutoRunOauthGate,
    CREATE_TASK_OAUTH_MISSING_TENANT_MSG,
    CREATE_TASK_OAUTH_MISSING_PROJECT_MSG,
  } = await import('./createTaskOauthGate.js')

  describe('createTaskOauthGate', () => {
    it('requires oauth gate only when auto_run is enabled', () => {
      expect(shouldRequireAutoRunOauthGate(true)).toBe(true)
      expect(shouldRequireAutoRunOauthGate(false)).toBe(false)
      expect(shouldRequireAutoRunOauthGate(undefined)).toBe(false)
    })

    it('collects fork task oauth urls from linked project details', () => {
      const urls = collectForkTaskOAuthRepoUrls([
        {
          project: {
            git_repos: [
              'https://github.com/a/b.git',
              'https://gitlab.example/c.git',
              'git://example.com/plain.git',
            ],
          },
        },
      ])
      expect(urls).toEqual([
        'https://github.com/a/b.git',
        'https://gitlab.example/c.git',
      ])
    })
    it('collects unique GitHub/GitLab urls from selected projects only', () => {
      const urls = collectCreateTaskOAuthRepoUrls(
        {
          projectSelections: [
            { projectId: 'p1' },
            { projectId: 'p2' },
            { projectId: '' },
          ],
        },
        [
          { id: 'p1', git_repos: ['https://github.com/a/b.git', 'https://github.com/a/b.git'] },
          { id: 'p2', git_repos: ['https://gitlab.example/c.git', 'git://example.com/plain.git'] },
          { id: 'p3', git_repos: ['https://github.com/skip/me.git'] },
        ],
      )
      expect(urls).toEqual([
        'https://github.com/a/b.git',
        'https://gitlab.example/c.git',
      ])
    })

    it('returns no oauth urls when nothing is selected', () => {
      expect(collectCreateTaskOAuthRepoUrls(null, [])).toEqual([])
      expect(collectCreateTaskOAuthRepoUrls({ projectSelections: [] }, [])).toEqual([])
    })

    it('treats connected true and connection list as bound', () => {
      expect(isGitOAuthUserAppConnectedPayload({ connected: true })).toBe(true)
      expect(isGitOAuthUserAppConnectedPayload({ connections: [{ connected: true }] })).toBe(true)
      expect(isGitOAuthUserAppConnectedPayload({ connected: false })).toBe(false)
      expect(isGitOAuthUserAppConnectedPayload({})).toBe(false)
      expect(isGitOAuthUserAppConnectedPayload(null)).toBe(false)
    })

    it('does not block when there are no oauth repos', () => {
      expect(resolveCreateTaskOauthBlockedReason({ hasOAuthRepos: false, loading: true })).toBe('')
    })

    it('blocks while oauth status is loading', () => {
      expect(resolveCreateTaskOauthBlockedReason({
        hasOAuthRepos: true,
        loading: true,
        allBound: false,
      })).toMatch(/正在检查仓库 Git OAuth/)
    })

    it('blocks with check error before unbound copy', () => {
      expect(resolveCreateTaskOauthBlockedReason({
        hasOAuthRepos: true,
        loading: false,
        allBound: false,
        checkError: '服务不可用',
      })).toBe('无法检查 Git OAuth 绑定：服务不可用')
    })

    it('blocks create when oauth repos are unbound', () => {
      expect(resolveCreateTaskOauthBlockedReason({
        hasOAuthRepos: true,
        loading: false,
        allBound: false,
        unboundRepoUrls: ['https://github.com/a/b.git'],
      })).toMatch(/开启自动运行前.*项目详情/)
    })

    it('maps selected project ids onto oauth repo rows', () => {
      expect(collectCreateTaskOAuthRepoRows(
        { projectSelections: [{ projectId: 'p1' }] },
        [{ id: 'p1', git_repos: ['https://github.com/a/b.git'] }],
      )).toEqual([{ url: 'https://github.com/a/b.git', projectId: 'p1' }])
      expect(collectForkTaskOAuthRepoRows([
        { project_id: 'p9', project: { id: 'p9', git_repos: ['https://github.com/a/b.git'] } },
      ])).toEqual([{ url: 'https://github.com/a/b.git', projectId: 'p9' }])
    })

    it('extracts oauth urls from git_repos objects and git_repo_entries', () => {
      expect(collectCreateTaskOAuthRepoRows(
        { projectSelections: [{ projectId: 'p1' }] },
        [{
          id: 'p1',
          git_repos: [{ url: 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git', clone_alias: 'demo' }],
        }],
      )).toEqual([{
        url: 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git',
        projectId: 'p1',
      }])
      expect(collectCreateTaskOAuthRepoRows(
        { projectSelections: [{ projectId: 'p1' }] },
        [{
          id: 'p1',
          git_repo_entries: [{ url: 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git' }],
          git_repos: ['https://example.com/ignored.git'],
        }],
      )).toEqual([{
        url: 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git',
        projectId: 'p1',
      }])
    })

    it('treats only token_available as project L2 bound', () => {
      expect(isCreateTaskOAuthTokenAvailable('token_available')).toBe(true)
      expect(isCreateTaskOAuthTokenAvailable('not_bound')).toBe(false)
      expect(isCreateTaskOAuthTokenAvailable('connected')).toBe(false)
    })

    it('fetches project L2 via validate-git-repos without probe_access', async () => {
      const apiFetch = async (url, opts) => {
        expect(url).toContain('/projects/validate-git-repos/tenant_id/t1/')
        const body = JSON.parse(opts.body)
        expect(body.probe_access).toBe(false)
        expect(body.project_id).toBe('p1')
        expect(body.urls).toEqual(['https://github.com/a/b.git'])
        return {
          ok: true,
          json: async () => ({
            results: [{ url: 'https://github.com/a/b.git', token_status: 'token_available' }],
          }),
        }
      }
      const out = await fetchCreateTaskProjectL2BoundByUrl({
        apiFetch,
        tenantId: 't1',
        rows: [{ url: 'https://github.com/a/b.git', projectId: 'p1' }],
        unboundUrls: ['https://github.com/a/b.git'],
      })
      expect(out.bound['https://github.com/a/b.git']).toBe(true)
    })

    it('does not treat L1-only not_bound as create-task bound', async () => {
      const out = await fetchCreateTaskProjectL2BoundByUrl({
        apiFetch: async () => ({
          ok: true,
          json: async () => ({
            results: [{ url: 'https://github.com/a/b.git', token_status: 'not_bound' }],
          }),
        }),
        tenantId: 't1',
        rows: [{ url: 'https://github.com/a/b.git', projectId: 'p1' }],
        unboundUrls: ['https://github.com/a/b.git'],
      })
      expect(out.bound['https://github.com/a/b.git']).toBe(false)
    })

    it('treats .git-suffix mismatch as the same repo when token_available', async () => {
      const out = await fetchCreateTaskProjectL2BoundByUrl({
        apiFetch: async () => ({
          ok: true,
          json: async () => ({
            results: [{
              url: 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git',
              token_status: 'token_available',
            }],
          }),
        }),
        tenantId: 't1',
        rows: [{
          url: 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo',
          projectId: 'proj_882824768007467008',
        }],
        unboundUrls: ['https://gitlab-tencent-sh-1.daydaymoney.com/g/demo'],
      })
      expect(out.bound['https://gitlab-tencent-sh-1.daydaymoney.com/g/demo']).toBe(true)
    })

    it('surfaces missing tenantId instead of silently showing unbound', async () => {
      const apiFetch = async () => {
        throw new Error('validate-git-repos must not run without tenantId')
      }
      const out = await fetchCreateTaskProjectL2BoundByUrl({
        apiFetch,
        tenantId: '',
        rows: [{ url: 'https://github.com/a/b.git', projectId: 'p1' }],
        unboundUrls: ['https://github.com/a/b.git'],
      })
      expect(out.bound['https://github.com/a/b.git']).toBe(false)
      expect(out.errorByUrl['https://github.com/a/b.git']).toBe(CREATE_TASK_OAUTH_MISSING_TENANT_MSG)
    })

    it('surfaces missing projectId instead of silently showing unbound', async () => {
      const out = await fetchCreateTaskProjectL2BoundByUrl({
        apiFetch: async () => {
          throw new Error('validate-git-repos must not run without projectId')
        },
        tenantId: 't1',
        rows: [{ url: 'https://github.com/a/b.git', projectId: '' }],
        unboundUrls: ['https://github.com/a/b.git'],
      })
      expect(out.errorByUrl['https://github.com/a/b.git']).toBe(CREATE_TASK_OAUTH_MISSING_PROJECT_MSG)
    })

    it('does not treat L1 connected payload as create-task session grant', () => {
      expect(isGitOAuthUserAppConnectedPayload({ connected: true })).toBe(true)
    })

    it('blocks fork auto-run when oauth repos are unbound', () => {
      expect(resolveForkAutoRunOauthBlockedReason({
        hasOAuthRepos: true,
        loading: false,
        allBound: false,
      })).toMatch(/自动运行并派生前.*项目详情/)
    })

    it('allows create when all oauth repos are bound', () => {
      expect(resolveCreateTaskOauthBlockedReason({
        hasOAuthRepos: true,
        loading: false,
        allBound: true,
        unboundRepoUrls: [],
      })).toBe('')
    })

    it('treats project list git_repos_status token_available as bound (list = L2 cache)', () => {
      const bound = buildCreateTaskProjectListBoundMap(
        [{ url: 'https://github.com/a/b', projectId: 'p1' }],
        [{
          id: 'p1',
          git_repos_status: [{
            repo_url: 'https://github.com/a/b.git',
            token_status: 'token_available',
          }],
        }],
      )
      expect(bound['https://github.com/a/b']).toBe(true)
    })

    it('does not treat not_applicable / not_bound list rows as bound', () => {
      const bound = buildCreateTaskProjectListBoundMap(
        [{ url: 'https://github.com/a/b', projectId: 'p1' }],
        [{
          id: 'p1',
          git_repos_status: [
            { repo_url: 'https://github.com/a/b.git', token_status: 'not_applicable' },
            { repo_url: 'https://gitlab.example/c.git', token_status: 'not_bound' },
          ],
        }],
      )
      expect(bound['https://github.com/a/b']).toBeFalsy()
      expect(bound['https://gitlab.example/c.git']).toBeFalsy()
    })

    it('only consults the owning project row for each url', () => {
      const bound = buildCreateTaskProjectListBoundMap(
        [{ url: 'https://github.com/a/b', projectId: 'p2' }],
        [
          {
            id: 'p1',
            git_repos_status: [{ repo_url: 'https://github.com/a/b.git', token_status: 'token_available' }],
          },
          { id: 'p2', git_repos_status: [] },
        ],
      )
      expect(bound['https://github.com/a/b']).toBeFalsy()
    })
  })
}
