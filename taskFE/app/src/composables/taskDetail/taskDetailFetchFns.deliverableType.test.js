// @vitest-environment node
/**
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.deliverableType.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const {
    fetchDeliverableTypeOptions,
    onDeliverableTypeChange,
  } = await import('./taskDetailFetchFns.js')

  describe('fetchDeliverableTypeOptions', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('成功时写入 current_deliverable_objs', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          current_deliverable_objs: [
            { id: 'd1', name: '价值流', color: '#111', order: 0 },
            { id: 'd2', name: '活动', color: '#222', order: 1 },
          ],
        }),
      })
      const deliverableTypeOptions = ref([])
      const isDeliverableTypesLoading = ref(false)
      const deliverableTypeError = ref('stale')
      await fetchDeliverableTypeOptions({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        deliverableTypeOptions,
        isDeliverableTypesLoading,
        deliverableTypeError,
      })
      expect(apiFetch).toHaveBeenCalledWith(
        '/api/projects/manage-deliverable-system/tenant_id/t1?workspace_id=w1',
        expect.objectContaining({ credentials: 'include' }),
      )
      expect(deliverableTypeOptions.value).toEqual([
        { id: 'd1', name: '价值流' },
        { id: 'd2', name: '活动' },
      ])
      expect(deliverableTypeError.value).toBe('')
      expect(isDeliverableTypesLoading.value).toBe(false)
    })

    it('失败时清空选项并设置错误', async () => {
      apiFetch.mockResolvedValue({ ok: false, status: 500 })
      const deliverableTypeOptions = ref([{ id: 'x', name: 'old' }])
      const isDeliverableTypesLoading = ref(false)
      const deliverableTypeError = ref('')
      await fetchDeliverableTypeOptions({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        deliverableTypeOptions,
        isDeliverableTypesLoading,
        deliverableTypeError,
      })
      expect(deliverableTypeOptions.value).toEqual([])
      expect(deliverableTypeError.value).toBe('获取交付物类别失败')
    })
  })

  describe('onDeliverableTypeChange', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('PATCH 成功后更新 localTask.deliverable_obj_id', async () => {
      apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
      const localTask = ref({ id: 'task-1', deliverable_obj_id: 'd1' })
      const isUpdatingDeliverableType = ref(false)
      const deliverableTypeError = ref('')
      await onDeliverableTypeChange(
        { target: { value: 'd2' } },
        {
          effectiveTenantId: ref('t1'),
          effectiveWorkspaceId: ref('w1'),
          effectiveTaskId: ref('task-1'),
          deliverableTypeOptions: ref([{ id: 'd2', name: '活动' }]),
          isUpdatingDeliverableType,
          deliverableTypeError,
          localTask,
        },
      )
      expect(apiFetch).toHaveBeenCalledWith(
        '/api/tasks/todos/tenant_id/t1/workspace_id/w1/task-1/',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ deliverable_obj_id: 'd2' }),
        }),
      )
      expect(localTask.value.deliverable_obj_id).toBe('d2')
      expect(isUpdatingDeliverableType.value).toBe(false)
    })

    it('清空类别时 PATCH null', async () => {
      apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
      const localTask = ref({ id: 'task-1', deliverable_obj_id: 'd1' })
      await onDeliverableTypeChange(
        { target: { value: '' } },
        {
          effectiveTenantId: ref('t1'),
          effectiveWorkspaceId: ref('w1'),
          effectiveTaskId: ref('task-1'),
          deliverableTypeOptions: ref([]),
          isUpdatingDeliverableType: ref(false),
          deliverableTypeError: ref(''),
          localTask,
        },
      )
      expect(apiFetch).toHaveBeenCalledWith(
        '/api/tasks/todos/tenant_id/t1/workspace_id/w1/task-1/',
        expect.objectContaining({
          body: JSON.stringify({ deliverable_obj_id: null }),
        }),
      )
      expect(localTask.value.deliverable_obj_id).toBeNull()
    })

    it('失败时设置错误且不改 localTask', async () => {
      apiFetch.mockResolvedValue({ ok: false, status: 400 })
      const localTask = ref({ id: 'task-1', deliverable_obj_id: 'd1' })
      const deliverableTypeError = ref('')
      await onDeliverableTypeChange(
        { target: { value: 'd2' } },
        {
          effectiveTenantId: ref('t1'),
          effectiveWorkspaceId: ref('w1'),
          effectiveTaskId: ref('task-1'),
          deliverableTypeOptions: ref([{ id: 'd2', name: '活动' }]),
          isUpdatingDeliverableType: ref(false),
          deliverableTypeError,
          localTask,
        },
      )
      expect(localTask.value.deliverable_obj_id).toBe('d1')
      expect(deliverableTypeError.value).toBe('更新交付物类别失败')
    })
  })
}
