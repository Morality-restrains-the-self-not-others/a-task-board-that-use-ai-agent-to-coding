// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useCreateTaskCollaborators.defaultCurrentUser.unit.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { ref, nextTick } = await import('vue')
  const { flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    getStoredUserIdMock: vi.fn(() => 'user-100'),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoisted.apiFetchMock,
  }))

  vi.mock('../utils/sessionUserIdUtils.js', () => ({
    getStoredUserId: () => hoisted.getStoredUserIdMock(),
  }))

  vi.mock('../utils/cookieUtils', () => ({
    getCookie: () => '',
  }))

  vi.mock('../utils/workPanelApiUtils.js', () => ({
    normalizeListPayload: (data) => (Array.isArray(data) ? data : data?.results || []),
    parseJsonSafe: vi.fn(async (response) => {
      try {
        return await response.json()
      } catch {
        return null
      }
    }),
    warnNetworkFailure: vi.fn(),
    warnOptionalApiFailure: vi.fn(),
  }))

  const { useCreateTaskCollaborators } = await import('./useCreateTaskCollaborators.js')

  const collaborators = [
    { id: 'member-1', user: 'user-100', member_name: '当前用户昵称' },
    { id: 'member-2', user: 'user-200', member_name: '同事甲' },
  ]

  const mountComposable = (taskOverrides = {}) => {
    const editingTask = ref({
      owner: '',
      operator: '',
      assignees: [],
      ...taskOverrides,
    })
    const show = ref(true)
    const api = useCreateTaskCollaborators({
      editingTask: () => editingTask.value,
      tenantId: () => 'tenant-1',
      currentWorkspace: () => ({ id: 'ws-1' }),
      show: () => show.value,
    })
    return { editingTask, show, api }
  }

  describe('useCreateTaskCollaborators default current user', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoisted.getStoredUserIdMock.mockReturnValue('user-100')
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => collaborators,
      })
    })

    afterEach(() => {
      vi.clearAllMocks()
    })

    it('创建任务时负责人与操作员默认指向当前登录成员', async () => {
      const { editingTask, api } = mountComposable()
      await flushPromises()
      await nextTick()

      expect(api.localCollaborators.value).toHaveLength(2)
      expect(editingTask.value.owner).toBe('member-1')
      expect(editingTask.value.operator).toBe('member-1')
      expect(api.ownerDisplayLabel.value).toBe('当前用户昵称')
      expect(api.operatorDisplayLabel.value).toBe('当前用户昵称')
    })

    it('优先使用 localStorage currentUserId（getStoredUserId），不依赖可读 cookie', async () => {
      hoisted.getStoredUserIdMock.mockReturnValue('user-200')
      const { editingTask } = mountComposable()
      await flushPromises()
      await nextTick()

      expect(editingTask.value.owner).toBe('member-2')
      expect(editingTask.value.operator).toBe('member-2')
    })

    it('编辑已有任务时不覆盖已有负责人/操作员，也不强行填充空操作员', async () => {
      const { editingTask } = mountComposable({
        id: 'task-existing',
        owner: 'member-2',
        operator: '',
      })
      await flushPromises()
      await nextTick()

      expect(editingTask.value.owner).toBe('member-2')
      expect(editingTask.value.operator).toBe('')
    })

    it('创建草稿预填 userId 时对齐为成员主键', async () => {
      const { editingTask } = mountComposable({
        owner: 'user-100',
        operator: 'user-100',
      })
      await flushPromises()
      await nextTick()

      expect(editingTask.value.owner).toBe('member-1')
      expect(editingTask.value.operator).toBe('member-1')
    })
  })
}
