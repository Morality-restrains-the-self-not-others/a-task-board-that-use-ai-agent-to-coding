// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.deliverableCategory.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { fetchDeliverableCategoryOptions } = await import('./taskDetailFetchFns.js')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')

describe('fetchDeliverableCategoryOptions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads current_deliverable_objs into options', async () => {
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        current_deliverable_objs: [
          { id: 1, name: '价值流', order: 1 },
          { id: '2', name: '业务流程', order: 2 },
        ],
      }),
    })
    const deliverableCategoryOptions = { value: [] }
    const isDeliverableCategoriesLoading = { value: false }
    const deliverableCategoryError = { value: 'stale' }
    await fetchDeliverableCategoryOptions({
      effectiveTenantId: { value: 't1' },
      effectiveWorkspaceId: { value: 'w1' },
      deliverableCategoryOptions,
      isDeliverableCategoriesLoading,
      deliverableCategoryError,
    })
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/projects/manage-deliverable-system/tenant_id/t1?workspace_id=w1',
      expect.objectContaining({ credentials: 'include' }),
    )
    expect(deliverableCategoryOptions.value).toEqual([
      { id: '1', name: '价值流', order: 1, color: undefined },
      { id: '2', name: '业务流程', order: 2, color: undefined },
    ])
    expect(deliverableCategoryError.value).toBe('')
    expect(isDeliverableCategoriesLoading.value).toBe(false)
  })

  it('skips fetch for default workspace', async () => {
    const deliverableCategoryOptions = { value: [{ id: 'x' }] }
    await fetchDeliverableCategoryOptions({
      effectiveTenantId: { value: 't1' },
      effectiveWorkspaceId: { value: 'default' },
      deliverableCategoryOptions,
      isDeliverableCategoriesLoading: { value: false },
      deliverableCategoryError: { value: '' },
    })
    expect(apiFetch).not.toHaveBeenCalled()
    expect(deliverableCategoryOptions.value).toEqual([])
  })
})
}
