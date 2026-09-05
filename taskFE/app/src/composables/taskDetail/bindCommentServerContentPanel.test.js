// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] bindCommentServerContentPanel.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { bindCommentServerContentPanel } = await import('./bindCommentServerContentPanel.js')

describe('bindCommentServerContentPanel', () => {
  it('binds fetchServerContent to the comment id and overlays snapshot fields', () => {
    const fetchServerContent = vi.fn()
    const panel = {
      snapshotForComment: (cid) => (cid === 'C9' ? { serverContentText: '<p>c9</p>', serverContentMessage: 'ok' } : {}),
      fetchServerContent,
    }
    const bound = bindCommentServerContentPanel(panel, 'C9')
    expect(bound.commentId).toBe('C9')
    expect(bound.serverContentText).toBe('<p>c9</p>')
    expect(bound.serverContentMessage).toBe('ok')
    bound.fetchServerContent()
    expect(fetchServerContent).toHaveBeenCalledWith('C9')
  })

  it('returns panel unchanged when comment id is missing', () => {
    const panel = { fetchServerContent: vi.fn() }
    expect(bindCommentServerContentPanel(panel, '')).toBe(panel)
    expect(bindCommentServerContentPanel(null, 'C1')).toBeNull()
  })
})
}
