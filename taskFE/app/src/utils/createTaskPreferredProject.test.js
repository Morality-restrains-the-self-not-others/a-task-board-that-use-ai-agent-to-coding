// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] createTaskPreferredProject.test.js requires vitest runtime')
} else {
  const { afterEach, describe, expect, it } = await import('vitest')
  const {
    createTaskPreferredProjectStorageKey,
    pickCreateTaskDefaultProjectId,
    readCreateTaskPreferredProject,
    rememberCreateTaskPreferredProject,
  } = await import('./createTaskPreferredProject.js')

  describe('createTaskPreferredProject', () => {
    afterEach(() => {
      localStorage.clear()
    })

    it('stores and reads preferred project per tenant', () => {
      rememberCreateTaskPreferredProject('882297276515512320', 'proj_882824768007467008')
      expect(readCreateTaskPreferredProject('882297276515512320')).toBe('proj_882824768007467008')
      expect(readCreateTaskPreferredProject('other-tenant')).toBe('')
      expect(createTaskPreferredProjectStorageKey('882297276515512320')).toContain('882297276515512320')
    })

    it('prefers last-viewed project over list order when it is in the workspace', () => {
      const projects = [
        { id: 'proj_aaa_first_by_name' },
        { id: 'proj_882824768007467008' },
      ]
      expect(pickCreateTaskDefaultProjectId(
        projects,
        new Set(),
        'proj_882824768007467008',
      )).toBe('proj_882824768007467008')
    })

    it('falls back to first unused project when preferred is missing from the workspace', () => {
      expect(pickCreateTaskDefaultProjectId(
        [{ id: 'proj_aaa' }, { id: 'proj_bbb' }],
        new Set(),
        'proj_not_in_workspace',
      )).toBe('proj_aaa')
    })

    it('skips used ids when picking fallback', () => {
      expect(pickCreateTaskDefaultProjectId(
        [{ id: 'p1' }, { id: 'p2' }],
        new Set(['p1']),
        '',
      )).toBe('p2')
    })
  })
}
