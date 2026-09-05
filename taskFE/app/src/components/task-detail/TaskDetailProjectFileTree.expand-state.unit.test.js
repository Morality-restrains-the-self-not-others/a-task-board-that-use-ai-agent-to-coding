// @vitest-environment jsdom
/**
 * 项目文件树：目录展开状态由父组件托管，按 layerId 缓存，跨层切换后可恢复。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailProjectFileTree.expand-state.unit.test.js requires vitest runtime')
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


  describe('TaskDetailProjectFileTree expand state across layers', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('展开目录后切换 layer 再切回，应恢复展开状态', async () => {
      const layerAFiles = ['repo-a/src/main.js', 'repo-a/README.md']
      const layerBFiles = ['other/x.txt']

      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        const u = String(url || '')
        if (u.includes('layer_id=layer_a')) {
          return Promise.resolve(jsonResponse({ entries: filesToEntries(layerAFiles) }))
        }
        if (u.includes('layer_id=layer_b')) {
          return Promise.resolve(jsonResponse({ entries: filesToEntries(layerBFiles) }))
        }
        return Promise.resolve(jsonResponse({ entries: filesToEntries([]) }))
      })

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_a',
          expandByDefault: true,
        },
      })

      await flushPromises()
      expect(wrapper.text()).toContain('repo-a')
      expect(wrapper.text()).not.toContain('main.js')

      const toggle = wrapper.find('[data-testid="project-file-tree-toggle-dir"]')
      expect(toggle.exists()).toBe(true)
      await toggle.trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('src')

      const toggles = wrapper.findAll('[data-testid="project-file-tree-toggle-dir"]')
      const srcToggle = toggles.find((btn) => {
        const name = btn.element.parentElement?.querySelector('.font-mono')?.textContent?.trim()
        return name === 'src'
      })
      expect(srcToggle).toBeTruthy()
      await srcToggle.trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('main.js')

      await wrapper.setProps({ layerId: 'layer_b' })
      await flushPromises()
      expect(wrapper.text()).toContain('other')
      expect(wrapper.text()).not.toContain('repo-a')

      await wrapper.setProps({ layerId: 'layer_a' })
      await flushPromises()
      // 跨层切回后应恢复 repo-a 与 src 的展开，直接可见 main.js
      expect(wrapper.text()).toContain('repo-a')
      expect(wrapper.text()).toContain('src')
      expect(wrapper.text()).toContain('main.js')
      wrapper.unmount()
    })

    it('同层刷新不丢失已展开目录', async () => {
      let call = 0
      hoistedMocks.apiFetchMock.mockImplementation(() => {
        call += 1
        return Promise.resolve(
          jsonResponse({ entries: filesToEntries(call === 1
              ? ['repo-a/src/a.js']
              : ['repo-a/src/a.js', 'repo-a/src/b.js'],) })
        )
      })

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_a',
          expandByDefault: true,
        },
      })
      await flushPromises()

      const rootToggle = wrapper.find('[data-testid="project-file-tree-toggle-dir"]')
      await rootToggle.trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('src')
      expect(rootToggle.attributes('aria-expanded')).toBe('true')

      await wrapper.find('button[title="刷新文件列表"]').trigger('click')
      await flushPromises()
      // 刷新后仍保持 repo-a 展开（可见子目录 src）；新文件进入树后父级展开状态不丢
      expect(wrapper.text()).toContain('src')
      const afterRefreshToggle = wrapper.find('[data-testid="project-file-tree-toggle-dir"]')
      expect(afterRefreshToggle.attributes('aria-expanded')).toBe('true')
      wrapper.unmount()
    })

    it('刷新后已失效的展开目录应被剪枝，有效展开仍保留', async () => {
      let call = 0
      hoistedMocks.apiFetchMock.mockImplementation(() => {
        call += 1
        return Promise.resolve(
          jsonResponse({ entries: filesToEntries(call === 1
              ? ['repo-a/src/a.js', 'repo-b/keep.txt']
              : ['repo-b/keep.txt'],) })
        )
      })

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_a',
          expandByDefault: true,
        },
      })
      await flushPromises()

      const toggles = () => wrapper.findAll('[data-testid="project-file-tree-toggle-dir"]')
      const toggleByName = (name) =>
        toggles().find((btn) => {
          const label = btn.element.parentElement?.querySelector('.font-mono')?.textContent?.trim()
          return label === name
        })

      await toggleByName('repo-a').trigger('click')
      await flushPromises()
      await toggleByName('src').trigger('click')
      await flushPromises()
      await toggleByName('repo-b').trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('a.js')
      expect(wrapper.text()).toContain('keep.txt')

      await wrapper.find('button[title="刷新文件列表"]').trigger('click')
      await flushPromises()

      // repo-a / src 已不存在于列表，展开态应剪掉；repo-b 仍展开可见 keep.txt
      expect(wrapper.text()).not.toContain('repo-a')
      expect(wrapper.text()).not.toContain('a.js')
      expect(wrapper.text()).toContain('repo-b')
      expect(wrapper.text()).toContain('keep.txt')
      expect(toggleByName('repo-b').attributes('aria-expanded')).toBe('true')
      wrapper.unmount()
    })
  })
}
