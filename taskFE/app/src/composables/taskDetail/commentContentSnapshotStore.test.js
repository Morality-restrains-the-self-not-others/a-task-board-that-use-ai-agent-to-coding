// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentContentSnapshotStore.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { createCommentContentSnapshotStore } = await import('./commentContentSnapshotStore.js')

describe('createCommentContentSnapshotStore', () => {
  it('isolates two comments so concurrent fetches cannot clobber each other', () => {
    const store = createCommentContentSnapshotStore()
    const a = store.refsFor('C1')
    const b = store.refsFor('C2')
    a.serverContentText.value = 'html-a'
    a.serverContentMessage.value = 'ok-a'
    b.serverContentText.value = 'html-b'
    b.serverContentMessage.value = 'ok-b'
    expect(store.get('C1').text).toBe('html-a')
    expect(store.get('C1').message).toBe('ok-a')
    expect(store.get('C2').text).toBe('html-b')
    expect(store.get('C2').message).toBe('ok-b')
  })

  it('returns empty slot for missing comment id', () => {
    const store = createCommentContentSnapshotStore()
    expect(store.get('')).toEqual({
      loading: false,
      message: '',
      messageTraceId: '',
      targetUrl: '',
      text: '',
    })
  })
})
}
