// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] useFileTreeLazyChildren.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { ref } = await import('vue')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')
const { useFileTreeLazyChildren, entriesToPathStrings } = await import('./useFileTreeLazyChildren.js')

function makeCtx() {
  const layerId = ref('layer-1')
  const requestContext = ref({ tenantId: 't1', workspaceId: 'w1', taskId: 'task_1', commentId: 'cmt-1' })
  const flatFiles = ref([])
  const treeNodes = ref([])
  return { layerId, requestContext, flatFiles, treeNodes }
}

describe('useFileTreeLazyChildren released guard', () => {
  beforeEach(() => {
    vi.mocked(apiFetch).mockReset()
  })

  it('fetches children when not released', async () => {
    const { layerId, requestContext, flatFiles, treeNodes } = makeCtx()
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ entries: [{ path: 'a.txt', type: 'file' }] }),
    })
    const { fetchChildren } = useFileTreeLazyChildren(layerId, requestContext, flatFiles, treeNodes)
    const entries = await fetchChildren('sub')
    expect(entries).toHaveLength(1)
    expect(apiFetch).toHaveBeenCalledTimes(1)
    expect(String(apiFetch.mock.calls[0][0])).toContain('container-layer-children')
  })

  it('skips fetch when containerReleased is true（OPT-20260823-038）', async () => {
    const { layerId, requestContext, flatFiles, treeNodes } = makeCtx()
    const released = ref(true)
    const { fetchChildren } = useFileTreeLazyChildren(layerId, requestContext, flatFiles, treeNodes, released)
    const entries = await fetchChildren('sub')
    expect(entries).toEqual([])
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('skips fetch when released is a lazy function（OPT-20260823-038）', async () => {
    const { layerId, requestContext, flatFiles, treeNodes } = makeCtx()
    const released = ref(true)
    const { fetchChildren } = useFileTreeLazyChildren(layerId, requestContext, flatFiles, treeNodes, () => released.value)
    const entries = await fetchChildren('sub')
    expect(entries).toEqual([])
    expect(apiFetch).not.toHaveBeenCalled()
  })
})
}
