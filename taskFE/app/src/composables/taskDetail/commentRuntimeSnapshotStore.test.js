import { describe, expect, it } from 'vitest'
import { createCommentRuntimeSnapshotStore } from './commentRuntimeSnapshotStore.js'

describe('createCommentRuntimeSnapshotStore', () => {
  it('keeps two comment slots independent', () => {
    const store = createCommentRuntimeSnapshotStore()
    const a = store.refsFor('cmt_a')
    const b = store.refsFor('cmt_b')
    a.serverRuntimeStatus.value = 'Running'
    a.serverRuntimeStatusMessage.value = 'ok-a'
    a.serverRuntimeStatusResponse.value = { instance_id: 'i-a' }
    b.serverRuntimeStatus.value = 'Stopped'
    b.serverRuntimeStatusMessage.value = 'ok-b'
    b.serverRuntimeStatusResponse.value = { instance_id: 'i-b' }

    expect(store.get('cmt_a').status).toBe('Running')
    expect(store.get('cmt_a').message).toBe('ok-a')
    expect(store.get('cmt_a').response.instance_id).toBe('i-a')
    expect(store.get('cmt_b').status).toBe('Stopped')
    expect(store.get('cmt_b').message).toBe('ok-b')
    expect(store.get('cmt_b').response.instance_id).toBe('i-b')
  })

  it('tracks loading per comment without cross-talk', () => {
    const store = createCommentRuntimeSnapshotStore()
    store.refsFor('c1').isServerRuntimeStatusLoading.value = true
    expect(store.get('c1').loading).toBe(true)
    expect(store.get('c2').loading).toBe(false)
  })

  it('ignores empty comment id patches', () => {
    const store = createCommentRuntimeSnapshotStore()
    store.refsFor('').serverRuntimeStatus.value = 'Running'
    expect(store.state.value).toEqual({})
    expect(store.get('').status).toBe('')
  })
})
