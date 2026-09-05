// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.translate-reexport.test.js requires vitest runtime')
} else {
const { describe, it, expect } = await import('vitest')
const { fetchTranslatedTaskTitleSegment } = await import('./taskDetailFetchFns.js')

describe('taskDetailFetchFns translate re-export', () => {
  it('re-exports fetchTranslatedTaskTitleSegment after the line-limit extract', () => {
    expect(typeof fetchTranslatedTaskTitleSegment).toBe('function')
  })
})
}
