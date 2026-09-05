// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] useProjectsBatchDelete.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

const apiFetchMock = vi.hoisted(() => vi.fn())

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => apiFetchMock(...args),
}))

  const { computed, ref } = await import('vue')
  const { useProjectsBatchDelete } = await import('./useProjectsBatchDelete.js')

function jsonResponse(body, ok = true) {
  return {
    ok,
    status: ok ? 200 : 500,
    json: async () => body,
  }
}

describe('useProjectsBatchDelete', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  it('toggleSelectAllFiltered 仅作用于 filteredProjects', () => {
    const projects = ref([
      { id: '1', name: 'A' },
      { id: '2', name: 'B' },
      { id: '3', name: 'C' },
    ])
    const filteredProjects = computed(() => projects.value.filter((p) => p.name !== 'C'))
    const batch = useProjectsBatchDelete({
      tenantId: ref('850256677331562496'),
      projects,
      filteredProjects,
      onDeleted: vi.fn(),
    })

    batch.toggleSelectAllFiltered(true)
    expect(batch.selectedCount.value).toBe(2)
    expect(batch.selectedProjectIds.value['3']).toBeUndefined()
  })

  it('executeBatchDelete 成功后退出选择模式', async () => {
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        deleted: [{ id: '1', name: 'A' }],
        errors: [],
      }),
    )

    const projects = ref([{ id: '1', name: 'A' }])
    const filteredProjects = computed(() => projects.value)
    const onDeleted = vi.fn()
    const batch = useProjectsBatchDelete({
      tenantId: ref('850256677331562496'),
      projects,
      filteredProjects,
      onDeleted,
    })

    batch.enterSelectionMode()
    batch.toggleProject('1', true)
    batch.showConfirmModal.value = true
    await batch.executeBatchDelete()

    expect(apiFetchMock).toHaveBeenCalledWith(
      '/api/projects/batch-delete/tenant_id/850256677331562496/',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ project_ids: ['1'] }),
      }),
    )
    expect(onDeleted).toHaveBeenCalled()
    expect(batch.selectionMode.value).toBe(false)
    expect(batch.showConfirmModal.value).toBe(false)
  })
})
}
