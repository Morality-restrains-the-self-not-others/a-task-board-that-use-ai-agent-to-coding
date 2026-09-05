// @vitest-environment jsdom
/**
 * 项目文件树：提交日志 / 文件预览请求失败时错误节点须带 data-traceId。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailProjectFileTree.preview-traceId.unit.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: {
        tenant: 't1',
        workspace: 'w1',
        taskId: 'task_1',
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

  function jsonResponse(body, ok = true, status = 200, traceId = '') {
    return {
      ok,
      status,
      traceId,
      json: async () => body,
    }
  }

  function filesToEntries(files) {
    return (Array.isArray(files) ? files : []).map((f) => ({
      type: 'file',
      path: String(f || '').replace(/\/+$/, ''),
    }))
  }

  describe('TaskDetailProjectFileTree preview data-traceId', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('提交日志 401 时错误节点带 data-traceId', async () => {
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        const u = String(url || '')
        if (u.includes('container-layer-children/')) {
          return Promise.resolve(jsonResponse({ entries: filesToEntries(['ram-work/README.md']) }))
        }
        if (u.includes('container-layer-git-log/')) {
          return Promise.resolve(
            jsonResponse(
              { detail: 'Invalid or missing access token' },
              false,
              401,
              'tid-git-log-401',
            ),
          )
        }
        return Promise.resolve(jsonResponse({}))
      })

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_1',
          expandByDefault: true,
        },
      })
      await flushPromises()

      const dirBtn = wrapper.findAll('button').find((b) => b.text().trim() === 'ram-work')
      expect(dirBtn).toBeTruthy()
      await dirBtn.trigger('click')
      await flushPromises()

      const err = wrapper.find('[data-testid="layer-change-preview-error"]')
      expect(err.exists()).toBe(true)
      expect(err.text()).toBe('Invalid or missing access token')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('tid-git-log-401')
      wrapper.unmount()
    })
  })
}
