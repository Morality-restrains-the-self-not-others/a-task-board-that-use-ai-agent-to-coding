// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailProjectFileTree.layer-pending.unit.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: {
        tenant: '875588283562749952',
        workspace: 'ws_-2740859684112864748',
        taskId: 'task_876722807445155840',
      },
    },
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => ({ push: vi.fn() }),
  }))

  const { default: TaskDetailProjectFileTree } = await import('./TaskDetailProjectFileTree.vue')
  const { LAYER_DIR_PENDING_HINT } = await import('../../utils/containerComputeWaiting.js')

  describe('TaskDetailProjectFileTree layer pending', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('shows waiting hint instead of raw layer not found on 404', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: false,
        status: 404,
        traceId: '87d18b17-707a-4ab7-aa36-6d796f0f0a26',
        json: async () => ({ detail: 'layer not found' }),
      })

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: '20260816_074513_e0a004',
          expandByDefault: true,
          tenantId: '875588283562749952',
          workspaceId: 'ws_-2740859684112864748',
          taskId: 'task_876722807445155840',
          commentId: 'cmt_876722826143363072',
        },
      })
      await flushPromises()

      expect(wrapper.find('[data-testid="project-file-tree-error"]').exists()).toBe(false)
      const waiting = wrapper.get('[data-testid="project-file-tree-waiting"]')
      expect(waiting.text()).toBe(LAYER_DIR_PENDING_HINT)
      expect(waiting.text()).not.toBe('layer not found')
      expect(waiting.attributes('data-traceid') || waiting.attributes('data-traceId')).toBe(
        '87d18b17-707a-4ab7-aa36-6d796f0f0a26',
      )
    })
  })
}
