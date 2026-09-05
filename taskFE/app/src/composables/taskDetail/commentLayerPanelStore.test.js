// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentLayerPanelStore.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const {
  createCommentLayerPanelStore,
  emptyCommentLayerPanelSlot,
} = await import('./commentLayerPanelStore.js')

describe('createCommentLayerPanelStore', () => {
  it('isolates snapshot and exec log between comments', () => {
    const store = createCommentLayerPanelStore()
    store.patch('cmt_a', {
      snapshot: { layers: [{ layer_id: 'layer-a' }], jobs: [] },
      execLogTopError: 'A 失败',
      containerEndpointRegistered: true,
    })
    store.patch('cmt_b', {
      snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [] },
      execLogTopError: '',
      containerEndpointRegistered: false,
    })
    expect(store.get('cmt_a').snapshot.layers[0].layer_id).toBe('layer-a')
    expect(store.get('cmt_b').snapshot.layers[0].layer_id).toBe('layer-b')
    expect(store.get('cmt_a').execLogTopError).toBe('A 失败')
    expect(store.get('cmt_b').execLogTopError).toBe('')
    expect(store.get('cmt_a').containerEndpointRegistered).toBe(true)
    expect(store.get('cmt_b').containerEndpointRegistered).toBe(false)
  })

  it('refsFor writes only the target comment', () => {
    const store = createCommentLayerPanelStore()
    store.refsFor('cmt_a').layerGraphCommandText.value = 'from A'
    store.refsFor('cmt_b').layerGraphCommandText.value = 'from B'
    expect(store.get('cmt_a').commandText).toBe('from A')
    expect(store.get('cmt_b').commandText).toBe('from B')
  })

  it('commentIdOwningJob finds the comment whose snapshot has the job', () => {
    const store = createCommentLayerPanelStore()
    store.patch('cmt_a', { snapshot: { layers: [], jobs: [{ id: 'job-a' }] } })
    store.patch('cmt_b', { snapshot: { layers: [], jobs: [{ id: 'job-b' }] } })
    expect(store.commentIdOwningJob('job-b')).toBe('cmt_b')
    expect(store.commentIdOwningJob('job-missing')).toBe('')
  })

  it('commentIdOwningLayer finds the comment whose snapshot has the layer', () => {
    const store = createCommentLayerPanelStore()
    store.patch('cmt_a', { snapshot: { layers: [{ layer_id: 'layer-a' }], jobs: [] } })
    store.patch('cmt_b', { snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [] } })
    expect(store.commentIdOwningLayer('layer-a')).toBe('cmt_a')
    expect(store.commentIdOwningLayer('layer-b')).toBe('cmt_b')
    expect(store.commentIdOwningLayer('layer-missing')).toBe('')
  })

  it('get on unknown id returns a fresh empty slot without storing it', () => {
    const store = createCommentLayerPanelStore()
    const slot = store.get('cmt_missing')
    expect(slot).toEqual(emptyCommentLayerPanelSlot())
    expect(store.state.value.cmt_missing).toBeUndefined()
  })
})
}
