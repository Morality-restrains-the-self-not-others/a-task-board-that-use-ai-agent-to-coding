// @vitest-environment jsdom
/**
 * 项目文件树刷新时不得先清空已有树（避免布局高度塌陷导致页面抖动）。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailProjectFileTree.refresh-stability.unit.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: {
        tenant: '850256677331562496',
        workspace: '861623708318031872',
        taskId: 'task_13165596112687184871',
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

  function filesToEntries(files) {
    return (Array.isArray(files) ? files : []).map((f) => ({
      type: 'file',
      path: String(f || '').replace(/\/+$/, ''),
    }))
  }


  describe('TaskDetailProjectFileTree refresh layout stability', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('刷新时保留已有树节点，不因 loading 卸载导致高度塌陷', async () => {
      let resolveRefresh
      const firstFiles = ['repo-a/README.md', 'repo-a/src/main.js']
      const secondFiles = ['repo-a/README.md', 'repo-a/src/main.js', 'repo-b/package.json']

      hoistedMocks.apiFetchMock
        .mockResolvedValueOnce(jsonResponse({ entries: filesToEntries(firstFiles) }))
        .mockImplementationOnce(
          () =>
            new Promise((resolve) => {
              resolveRefresh = () => resolve(jsonResponse({ entries: filesToEntries(secondFiles) }))
            })
        )

      // 不设 containerEndpointRegistered，避免与 layerId watch 双触发消耗 mock
      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_1',
          expandByDefault: true,
          fileTreeRefreshNonce: 0,
        },
      })

      await flushPromises()
      expect(wrapper.find('[data-testid="project-file-tree-body"]').exists()).toBe(true)
      // 目录默认折叠，根级可见仓库名即可证明旧树仍挂载
      expect(wrapper.text()).toContain('repo-a')
      expect(wrapper.text()).not.toContain('加载文件列表中…')

      const refreshBtn = wrapper.find('button[title="刷新文件列表"]')
      expect(refreshBtn.exists()).toBe(true)
      await refreshBtn.trigger('click')
      await flushPromises()

      // 刷新进行中：旧树仍在 DOM，且不出现仅「加载中」替换整块内容
      expect(wrapper.find('[data-testid="project-file-tree-body"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('repo-a')
      expect(wrapper.text()).not.toContain('加载文件列表中…')
      expect(wrapper.find('[data-testid="project-file-tree-body"]').classes()).toContain('opacity-70')
      expect(typeof resolveRefresh).toBe('function')

      resolveRefresh()
      await flushPromises()

      expect(wrapper.text()).toContain('repo-b')
      expect(wrapper.find('[data-testid="project-file-tree-body"]').classes()).not.toContain('opacity-70')
      wrapper.unmount()
    })

    it('首次加载无数据时仍显示加载提示', async () => {
      let resolveFirst = null
      hoistedMocks.apiFetchMock.mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveFirst = () => resolve(jsonResponse({ entries: filesToEntries(['only/file.txt']) }))
          })
      )

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_1',
          expandByDefault: true,
        },
      })

      await flushPromises()
      expect(wrapper.text()).toContain('加载文件列表中…')
      expect(wrapper.find('[data-testid="project-file-tree-body"]').exists()).toBe(false)
      expect(typeof resolveFirst).toBe('function')

      resolveFirst()
      await flushPromises()
      expect(wrapper.find('[data-testid="project-file-tree-body"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('only')
      expect(wrapper.text()).not.toContain('加载文件列表中…')
      wrapper.unmount()
    })
  })
}
