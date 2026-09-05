// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] createTaskGitIdentityGate.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    shouldRequireAutoRunGitIdentityGate,
    collectCreateTaskRepoUrls,
    collectForkTaskRepoUrls,
    resolveForkAutoRunGitIdentityBlockedReason,
    validateCreateTaskGitIdentities,
    buildCreateTaskRepoIdentitiesPayload,
    stampRepoIdentityOauthGitsite,
    isOAuthCapableGitsite,
    formatGitIdentityOptionLabel,
  } = await import('./createTaskGitIdentityGate.js')

  describe('createTaskGitIdentityGate', () => {
    it('requires git identity gate only when auto_run is enabled', () => {
      expect(shouldRequireAutoRunGitIdentityGate(true)).toBe(true)
      expect(shouldRequireAutoRunGitIdentityGate(false)).toBe(false)
      expect(shouldRequireAutoRunGitIdentityGate(undefined)).toBe(false)
    })

    it('collects unique repo urls from selected projects', () => {
      const urls = collectCreateTaskRepoUrls(
        {
          projectSelections: [
            { projectId: '100' },
            { projectId: '100' },
            { projectId: '200' },
          ],
        },
        [
          { id: '100', git_repos: ['https://git.example/a.git', 'https://git.example/b.git'] },
          { id: '200', git_repos: ['https://git.example/a.git', 'https://git.example/c.git'] },
        ],
      )
      expect(urls).toEqual([
        'https://git.example/a.git',
        'https://git.example/b.git',
        'https://git.example/c.git',
      ])
    })

    it('collects repo urls from git_repos objects', () => {
      expect(collectCreateTaskRepoUrls(
        { projectSelections: [{ projectId: '100' }] },
        [{ id: '100', git_repos: [{ url: 'https://gitlab.example/g/a.git', clone_alias: 'a' }] }],
      )).toEqual(['https://gitlab.example/g/a.git'])
    })

    it('does not block when auto_run is off even if identities missing', () => {
      expect(validateCreateTaskGitIdentities(
        false,
        [],
        ['https://git.example/a.git'],
      )).toBe('')
    })

    it('does not block auto_run when there are no repo urls', () => {
      expect(validateCreateTaskGitIdentities(true, [], [])).toBe('')
    })

    it('blocks auto_run when a linked repo has no git identity', () => {
      const err = validateCreateTaskGitIdentities(
        true,
        [{ repo_url: 'https://git.example/a.git', git_identity_id: 'gid-1' }],
        ['https://git.example/a.git', 'https://git.example/b.git'],
      )
      expect(err).toContain('Git 提交身份')
      expect(err).toContain('https://git.example/b.git')
    })

    it('passes auto_run when every required repo has git_identity_id', () => {
      expect(validateCreateTaskGitIdentities(
        true,
        [
          { repo_url: 'https://git.example/a.git', git_identity_id: 'gid-1' },
          { repo_url: 'https://git.example/b.git', git_identity_id: 'gid-2' },
        ],
        ['https://git.example/a.git', 'https://git.example/b.git'],
      )).toBe('')
    })

    it('omits repo_identities payload when auto_run is off', () => {
      expect(buildCreateTaskRepoIdentitiesPayload(
        false,
        [{ repo_url: 'https://git.example/a.git', git_identity_id: 'gid-1' }],
      )).toBeUndefined()
    })

    it('builds payload only for selected identities when auto_run is on', () => {
      expect(buildCreateTaskRepoIdentitiesPayload(
        true,
        [
          { repo_url: 'https://git.example/a.git', git_identity_id: 'gid-1' },
          { repo_url: 'https://git.example/b.git', git_identity_id: '' },
        ],
      )).toEqual([{
        repo_url: 'https://git.example/a.git',
        git_identity_id: 'gid-1',
      }])
    })

    it('OPT-20260902-028: does not stamp oauth_gitsite for generic git hosts', () => {
      expect(stampRepoIdentityOauthGitsite([
        { repo_url: 'https://git.example/a.git', git_identity_id: 'gid-1' },
      ])).toEqual([{
        repo_url: 'https://git.example/a.git',
        git_identity_id: 'gid-1',
      }])
      expect(buildCreateTaskRepoIdentitiesPayload(true, [
        { repo_url: 'https://git.example/a.git', git_identity_id: 'gid-1' },
      ])).toEqual([{
        repo_url: 'https://git.example/a.git',
        git_identity_id: 'gid-1',
      }])
    })

    it('OPT-20260902-028: isOAuthCapableGitsite mirrors backend github/gitlab rule', () => {
      expect(isOAuthCapableGitsite('github.com')).toBe(true)
      expect(isOAuthCapableGitsite('sub.github.com')).toBe(true)
      expect(isOAuthCapableGitsite('gitlab.daydaymoney.com')).toBe(true)
      expect(isOAuthCapableGitsite('git.example')).toBe(false)
      expect(isOAuthCapableGitsite('example.com')).toBe(false)
      expect(isOAuthCapableGitsite('')).toBe(false)
    })

    it('stamps GitLab host as oauth_gitsite on fork auto-run payload', () => {
      const url = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git'
      expect(stampRepoIdentityOauthGitsite([
        { repo_url: url, git_identity_id: 'gid-gl' },
      ])).toEqual([{
        repo_url: url,
        git_identity_id: 'gid-gl',
        oauth_gitsite: 'gitlab-tencent-sh-1.daydaymoney.com',
      }])
      expect(buildCreateTaskRepoIdentitiesPayload(true, [
        { repo_url: url, git_identity_id: 'gid-gl' },
      ])).toEqual([{
        repo_url: url,
        git_identity_id: 'gid-gl',
        oauth_gitsite: 'gitlab-tencent-sh-1.daydaymoney.com',
      }])
    })

    it('formats identity labels as name <email>', () => {
      expect(formatGitIdentityOptionLabel({
        git_user_name: 'Ann',
        git_user_email: 'ann@example.com',
      })).toBe('Ann <ann@example.com>')
    })

    it('collects fork repo urls from task project details', () => {
      expect(collectForkTaskRepoUrls([
        { project: { git_repos: ['https://gitlab.example/a.git', 'https://gitlab.example/a.git'] } },
        { project: { git_repos: ['https://gitlab.example/b.git'] } },
      ])).toEqual([
        'https://gitlab.example/a.git',
        'https://gitlab.example/b.git',
      ])
    })

    it('blocks fork auto-run while git identities are loading', () => {
      expect(resolveForkAutoRunGitIdentityBlockedReason({
        loading: true,
        requiredRepoUrls: ['https://gitlab.example/a.git'],
        selections: [],
      })).toContain('正在加载')
    })

    it('blocks fork auto-run when a linked repo has no current-user identity', () => {
      const err = resolveForkAutoRunGitIdentityBlockedReason({
        requiredRepoUrls: ['https://gitlab.example/a.git'],
        selections: [],
      })
      expect(err).toContain('Git 提交身份')
      expect(err).toContain('https://gitlab.example/a.git')
    })

    it('allows fork auto-run when every linked repo has git_identity_id', () => {
      expect(resolveForkAutoRunGitIdentityBlockedReason({
        requiredRepoUrls: ['https://gitlab.example/a.git'],
        selections: [{
          repo_url: 'https://gitlab.example/a.git',
          git_identity_id: 'gid-current',
        }],
      })).toBe('')
    })
  })
}
