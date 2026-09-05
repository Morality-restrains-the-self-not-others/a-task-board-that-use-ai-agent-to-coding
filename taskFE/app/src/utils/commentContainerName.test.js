// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentContainerName.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const {
  buildCommentContainerName,
  normalizeCommentContainerName,
} = await import('./commentContainerName.js')

describe('buildCommentContainerName', () => {
  it('prefixes bare task ids once', () => {
    expect(buildCommentContainerName('task1', 'cNew')).toBe('task_task1_cNew')
    expect(buildCommentContainerName('13908509172117356865', 'cmt_42')).toBe(
      'task_13908509172117356865_cmt_42',
    )
  })

  it('does not double-prefix task_ ids', () => {
    expect(
      buildCommentContainerName('task_15666874162351520866', 'cmt_15666877397783348868'),
    ).toBe('task_15666874162351520866_cmt_15666877397783348868')
  })

  it('returns empty when either side missing', () => {
    expect(buildCommentContainerName('', 'c1')).toBe('')
    expect(buildCommentContainerName('task_1', '')).toBe('')
  })
})

describe('normalizeCommentContainerName', () => {
  it('rewrites legacy double task_ prefix', () => {
    expect(
      normalizeCommentContainerName(
        'task_15666874162351520866',
        'cmt_9',
        'task_task_15666874162351520866_cmt_9',
      ),
    ).toBe('task_15666874162351520866_cmt_9')
  })

  it('keeps canonical and non-legacy names', () => {
    expect(
      normalizeCommentContainerName('task_1', 'cmt_9', 'task_1_cmt_9'),
    ).toBe('task_1_cmt_9')
    expect(
      normalizeCommentContainerName('task1', 'cNew', 'task_task1_cNew'),
    ).toBe('task_task1_cNew')
  })

  it('derives when stored empty', () => {
    expect(normalizeCommentContainerName('task_1', 'cmt_9', '')).toBe('task_1_cmt_9')
  })
})
}
