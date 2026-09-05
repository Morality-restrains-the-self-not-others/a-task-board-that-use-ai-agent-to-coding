// @vitest-environment jsdom
/**
 * work-panel 弹窗打开任务详情时，路由仅有 tenant，无 workspace/taskId。
 * 文件树须能从 props 取得路由上下文并成功拉取，不得报「缺少路由上下文」。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailProjectFileTree.route-context.unit.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: {
        tenant: '850256677331562496',
        // work-panel：无 workspace / taskId
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

  function jsonResponse(body, ok = true, status = 200) {
    return {
      ok,
      status,
      json: async () => body,
    }
  }

  describe('TaskDetailProjectFileTree route context from props (work-panel modal)', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.routeMock.params = { tenant: '850256677331562496' }
    })

    it('路由缺 workspace/taskId 时，用 props 拉取文件树且不展示路由上下文错误', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValueOnce(
        jsonResponse({ entries: [{ type: 'file', path: 'repo-a/README.md' }] })
      )

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_1',
          expandByDefault: true,
          fileTreeRefreshNonce: 0,
          tenantId: '850256677331562496',
          workspaceId: '861623708318031872',
          taskId: 'task_13165596112687184871',
        },
      })

      await flushPromises()

      expect(wrapper.find('[data-testid="project-file-tree-error"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('缺少路由上下文，无法拉取文件树')
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledTimes(1)
      const calledPath = String(hoistedMocks.apiFetchMock.mock.calls[0][0] || '')
      expect(calledPath).toContain('/api/cloud/compute/container-layer-children/tenant_id/850256677331562496/workspace_id/861623708318031872')
      expect(calledPath).toContain('/task_id/task_13165596112687184871/')
      expect(wrapper.find('[data-testid="project-file-tree-body"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('repo-a')
    })

    it('路由与 props 均缺 workspace/taskId 时仍提示缺少路由上下文', async () => {
      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_1',
          expandByDefault: true,
          fileTreeRefreshNonce: 0,
        },
      })

      await flushPromises()

      const err = wrapper.find('[data-testid="project-file-tree-error"]')
      expect(err.exists()).toBe(true)
      expect(err.text()).toBe('缺少路由上下文，无法拉取文件树')
      expect(hoistedMocks.apiFetchMock).not.toHaveBeenCalled()
    })
  })
}
