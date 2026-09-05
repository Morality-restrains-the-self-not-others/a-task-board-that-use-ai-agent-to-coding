// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskKindOptions.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    DEFAULT_TASK_KIND_OPTIONS,
    normalizeTaskKindOptions,
    taskKindOptionsFromResponse,
  } = await import('./taskKindOptions.js')

  describe('taskKindOptions', () => {
    it('默认值为 bug-fix / feature', () => {
      expect(DEFAULT_TASK_KIND_OPTIONS).toEqual(['bug-fix', 'feature'])
      expect(normalizeTaskKindOptions([])).toEqual(['bug-fix', 'feature'])
    })

    it('去重并保留首次大小写', () => {
      expect(normalizeTaskKindOptions([' Feature ', 'FEATURE', 'bug-fix', ''])).toEqual([
        'Feature',
        'bug-fix',
      ])
    })

    it('从 API 响应解析 options', () => {
      expect(taskKindOptionsFromResponse({ options: ['chore'] })).toEqual(['chore'])
      expect(taskKindOptionsFromResponse(null)).toEqual(['bug-fix', 'feature'])
    })
  })
}
