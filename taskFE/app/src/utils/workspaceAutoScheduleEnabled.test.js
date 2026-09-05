// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] workspaceAutoScheduleEnabled.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    isWorkspaceAutoScheduleEnabled,
    fetchWorkspaceAutoScheduleEnabled,
  } = await import('./workspaceAutoScheduleEnabled.js')

  describe('isWorkspaceAutoScheduleEnabled', () => {
    it('true only when schedule_rhythm.enabled is boolean true', () => {
      expect(isWorkspaceAutoScheduleEnabled({ schedule_rhythm: { enabled: true } })).toBe(true)
      expect(isWorkspaceAutoScheduleEnabled({ schedule_rhythm: { enabled: false } })).toBe(false)
      expect(isWorkspaceAutoScheduleEnabled({ schedule_rhythm: {} })).toBe(false)
      expect(isWorkspaceAutoScheduleEnabled({})).toBe(false)
      expect(isWorkspaceAutoScheduleEnabled(null)).toBe(false)
      expect(isWorkspaceAutoScheduleEnabled({ schedule_rhythm: { enabled: 'true' } })).toBe(false)
    })
  })

  describe('fetchWorkspaceAutoScheduleEnabled', () => {
    it('returns enabled from GET snapshot', async () => {
      const apiFetch = async () => ({
        ok: true,
        json: async () => ({ schedule_rhythm: { enabled: true } }),
      })
      const got = await fetchWorkspaceAutoScheduleEnabled({
        tenantId: 't1',
        workspaceId: 'ws1',
        apiFetch,
      })
      expect(got.enabled).toBe(true)
    })

    it('attaches traceId on HTTP error', async () => {
      const apiFetch = async () => ({
        ok: false,
        status: 500,
        headers: { get: (k) => (k === 'X-Trace-Id' ? 'tid-err' : '') },
        json: async () => ({ error: 'boom' }),
      })
      await expect(fetchWorkspaceAutoScheduleEnabled({
        tenantId: 't1',
        workspaceId: 'ws1',
        apiFetch,
      })).rejects.toMatchObject({ message: 'boom', traceId: 'tid-err' })
    })
  })
}
