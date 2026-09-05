// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentExecutionGitIdentity.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    commentGitIdentityCompactLabel,
    commentGitIdentitySummaryItems,
    normalizeCommentRepoIdentities,
  } = await import('./commentExecutionGitIdentity.js')

  describe('normalizeCommentRepoIdentities', () => {
    it('returns empty for missing or invalid input', () => {
      expect(normalizeCommentRepoIdentities(null)).toEqual([])
      expect(normalizeCommentRepoIdentities(undefined)).toEqual([])
      expect(normalizeCommentRepoIdentities({})).toEqual([])
      expect(normalizeCommentRepoIdentities([{ repo_url: 'https://git.example/a.git' }])).toEqual([])
    })

    it('keeps unique git_identity_id in first-seen order', () => {
      expect(
        normalizeCommentRepoIdentities([
          { repo_url: 'https://git.example/a.git', git_identity_id: 'gi_1' },
          { repo_url: 'https://git.example/b.git', git_identity_id: 'gi_1' },
          { repo_url: 'https://git.example/c.git', git_identity_id: 'gi_2' },
          null,
        ]),
      ).toEqual([
        { repo_url: 'https://git.example/a.git', git_identity_id: 'gi_1' },
        { repo_url: 'https://git.example/c.git', git_identity_id: 'gi_2' },
      ])
    })
  })

  describe('commentGitIdentityCompactLabel', () => {
    it('prefers explicit label such as 默认身份', () => {
      expect(commentGitIdentityCompactLabel({ label: '默认身份', git_user_name: 'zhenghe' })).toBe(
        '默认身份',
      )
    })

    it('uses 默认身份 when is_default and label is empty', () => {
      expect(commentGitIdentityCompactLabel({ is_default: true, git_user_name: 'zhenghe' })).toBe(
        '默认身份',
      )
    })

    it('falls back to git user name then generic Git 身份', () => {
      expect(commentGitIdentityCompactLabel({ git_user_name: 'alice' })).toBe('alice')
      expect(commentGitIdentityCompactLabel(null)).toBe('Git 身份')
      expect(commentGitIdentityCompactLabel({ id: 'gi_x' })).toBe('Git 身份')
    })
  })

  describe('commentGitIdentitySummaryItems', () => {
    const options = [
      {
        id: 'gi_877397592462356480',
        label: '默认身份',
        git_user_name: 'zhenghe',
        git_user_email: 'zhenghe@example.com',
        is_default: true,
      },
      {
        id: 'gi_work',
        label: '工作身份',
        git_user_name: 'bob',
        git_user_email: 'bob@example.com',
      },
    ]

    it('returns empty when comment has no selected git identity', () => {
      expect(commentGitIdentitySummaryItems([], options)).toEqual([])
      expect(commentGitIdentitySummaryItems(undefined, options)).toEqual([])
    })

    it('resolves comment git_identity_id to compact label and full title', () => {
      const items = commentGitIdentitySummaryItems(
        [
          {
            repo_url: 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad',
            git_identity_id: 'gi_877397592462356480',
          },
        ],
        options,
      )
      expect(items).toEqual([
        {
          id: 'gi_877397592462356480',
          label: '默认身份',
          text: 'Git 身份 · 默认身份',
          title: 'zhenghe <zhenghe@example.com> · 默认身份',
        },
      ])
    })

    it('still emits a chip when identity catalog has not loaded', () => {
      const items = commentGitIdentitySummaryItems(
        [{ repo_url: 'https://git.example/a.git', git_identity_id: 'gi_missing' }],
        [],
      )
      expect(items).toEqual([
        { id: 'gi_missing', label: 'Git 身份', text: 'Git 身份', title: 'gi_missing' },
      ])
    })

    it('does not collapse two different identities into one chip', () => {
      const items = commentGitIdentitySummaryItems(
        [
          { repo_url: 'https://git.example/a.git', git_identity_id: 'gi_877397592462356480' },
          { repo_url: 'https://git.example/b.git', git_identity_id: 'gi_work' },
        ],
        options,
      )
      expect(items.map((row) => row.label)).toEqual(['默认身份', '工作身份'])
    })

    it('falls back to task-level identities when comment has none (auto_run first comment)', () => {
      const items = commentGitIdentitySummaryItems(
        [],
        options,
        [{ repo_url: 'https://git.example/a.git', git_identity_id: 'gi_work' }],
      )
      expect(items).toEqual([
        { id: 'gi_work', label: '工作身份', text: 'Git 身份 · 工作身份', title: 'bob <bob@example.com> · 工作身份' },
      ])
    })

    it('prefers comment identities over fallback when both present', () => {
      const items = commentGitIdentitySummaryItems(
        [{ repo_url: 'https://git.example/a.git', git_identity_id: 'gi_work' }],
        options,
        [{ repo_url: 'https://git.example/b.git', git_identity_id: 'gi_877397592462356480' }],
      )
      expect(items.map((row) => row.label)).toEqual(['工作身份'])
    })

    it('returns empty when both comment and fallback are empty', () => {
      expect(commentGitIdentitySummaryItems([], options, [])).toEqual([])
      expect(commentGitIdentitySummaryItems(undefined, options, undefined)).toEqual([])
    })
  })
}
