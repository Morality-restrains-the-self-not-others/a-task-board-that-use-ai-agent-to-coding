// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] peopleInviteLinkPayload.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { buildLinkInviteBody, formatInviteUseLabel, memberNameRequiredForLink, parseOpenMaxUses } = await import('./peopleInviteLinkPayload.js')

  describe('peopleInviteLinkPayload', () => {
    it('单次不带 link_kind', () => {
      const body = buildLinkInviteBody({
        formData: { company_member_name: 'A', role: 'member', expiration_days: 7, workspace_id: 'ws1' },
        pendingGrants: [],
        selectedRoleNames: [],
        linkKind: 'single',
        maxUses: '',
      })
      expect(body.link_kind).toBeUndefined()
      expect(memberNameRequiredForLink('single')).toBe(true)
    })

    it('开放且未填人数 → max_uses=0', () => {
      const body = buildLinkInviteBody({
        formData: { company_member_name: '', role: 'member', expiration_days: 7, workspace_id: 'ws1' },
        pendingGrants: [],
        selectedRoleNames: [],
        linkKind: 'open',
        maxUses: '',
      })
      expect(body.link_kind).toBe('open')
      expect(body.max_uses).toBe(0)
      expect(memberNameRequiredForLink('open')).toBe(false)
    })

    it('开放且填人数 → max_uses=5', () => {
      const body = buildLinkInviteBody({
        formData: { company_member_name: '', role: 'member', expiration_days: 7, workspace_id: 'ws1' },
        pendingGrants: [],
        selectedRoleNames: [],
        linkKind: 'open',
        maxUses: '5',
      })
      expect(body.link_kind).toBe('open')
      expect(body.max_uses).toBe(5)
    })

    it('parseOpenMaxUses', () => {
      expect(parseOpenMaxUses('5')).toBe(5)
      expect(parseOpenMaxUses('')).toBe(0)
    })

    it('formatInviteUseLabel', () => {
      expect(formatInviteUseLabel({ invite_method: 'link' })).toBe('单次')
      expect(formatInviteUseLabel({ link_kind: 'open', use_count: 2, max_uses: 0 })).toBe('已加入 2 / 不限')
      expect(formatInviteUseLabel({ link_kind: 'open', use_count: 1, max_uses: 10 })).toBe('已加入 1 / 10')
    })
  })
}
