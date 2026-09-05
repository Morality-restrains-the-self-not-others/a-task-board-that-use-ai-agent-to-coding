// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] createTaskQueuedAutoRun.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    shouldShowCreateTaskQueuedAutoRun,
    resolveQueuedAutoRunHint,
    appendQueuedAutoRunToCreatePayload,
  } = await import('./createTaskQueuedAutoRun.js')

  describe('shouldShowCreateTaskQueuedAutoRun', () => {
    it('T1 shows only when auto_run enabled and workspace schedule on', () => {
      expect(shouldShowCreateTaskQueuedAutoRun({
        canEnableAutoRun: true,
        autoRun: true,
        workspaceScheduleEnabled: true,
      })).toBe(true)
    })

    it('T2 hides when workspace schedule is off', () => {
      expect(shouldShowCreateTaskQueuedAutoRun({
        canEnableAutoRun: true,
        autoRun: true,
        workspaceScheduleEnabled: false,
      })).toBe(false)
    })

    it('hides when auto_run is off', () => {
      expect(shouldShowCreateTaskQueuedAutoRun({
        canEnableAutoRun: true,
        autoRun: false,
        workspaceScheduleEnabled: true,
      })).toBe(false)
    })
  })

  describe('appendQueuedAutoRunToCreatePayload', () => {
    it('T3 writes true when auto_run and queued_auto_run', () => {
      const payload = { title: 'x' }
      appendQueuedAutoRunToCreatePayload(payload, { auto_run: true, queued_auto_run: true })
      expect(payload.queued_auto_run).toBe(true)
    })

    it('T4 writes false when auto_run on but queue unchecked', () => {
      const payload = {}
      appendQueuedAutoRunToCreatePayload(payload, { auto_run: true, queued_auto_run: false })
      expect(payload.queued_auto_run).toBe(false)
    })

    it('omits field when auto_run is off', () => {
      const payload = {}
      appendQueuedAutoRunToCreatePayload(payload, { auto_run: false, queued_auto_run: true })
      expect(payload).not.toHaveProperty('queued_auto_run')
    })
  })

  describe('resolveQueuedAutoRunHint', () => {
    it('explains queue defers immediate start', () => {
      const hint = resolveQueuedAutoRunHint()
      expect(hint).toContain('自动调度')
      expect(hint).toContain('不立即启服')
    })
  })
}
