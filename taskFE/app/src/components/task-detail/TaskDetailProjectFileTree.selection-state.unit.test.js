// @vitest-environment jsdom
/**
 * 项目文件树：选中 path / 预览类型按 layerId 缓存，切回层后恢复预览。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailProjectFileTree.selection-state.unit.test.js requires vitest runtime')
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

  /** 将扁平 path 列表转为 children API 的 entries（测例用完整列表模拟旧 files 行为） */
  function filesToEntries(files) {
    return (files || []).map((f) => {
      const path = String(f || '').replace(/\/+$/, '')
      return { type: 'file', path }
    })
  }

  function parseChildrenUrl(url) {
    const u = String(url || '')
    if (!u.includes('container-layer-children/')) return null
    const layer = decodeURIComponent((u.match(/layer_id=([^&]*)/) || [])[1] || '')
    const dir = decodeURIComponent((u.match(/[?&]dir=([^&]*)/) || [])[1] || '')
    return { layer, dir }
  }

  describe('TaskDetailProjectFileTree selection restore across layers', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('选中文件后切换 layer 再切回，应恢复选中 path 并重新拉取文件预览', async () => {
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        const u = String(url || '')
        const children = parseChildrenUrl(u)
        if (children) {
          if (children.layer === 'layer_a') {
            return Promise.resolve(jsonResponse({
              entries: filesToEntries(['repo-a/README.md', 'repo-a/src/main.js']),
            }))
          }
          return Promise.resolve(jsonResponse({
            entries: filesToEntries(['other/x.txt']),
          }))
        }
        if (u.includes('container-layer-file-content/')) {
          if (u.includes('path=repo-a%2FREADME.md') || u.includes('path=repo-a/README.md')) {
            return Promise.resolve(
              jsonResponse({ path: 'repo-a/README.md', content: 'hello-from-a', truncated: false })
            )
          }
          if (u.includes('path=other%2Fx.txt') || u.includes('path=other/x.txt')) {
            return Promise.resolve(
              jsonResponse({ path: 'other/x.txt', content: 'hello-from-b', truncated: false })
            )
          }
          return Promise.resolve(jsonResponse({ path: 'unknown', content: '', truncated: false }))
        }
        return Promise.resolve(jsonResponse({}))
      })

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_a',
          expandByDefault: true,
        },
      })
      await flushPromises()

      // 展开 repo-a 后点击 README.md
      await wrapper.find('[data-testid="project-file-tree-toggle-dir"]').trigger('click')
      await flushPromises()
      const fileBtn = wrapper.findAll('button').find((b) => b.text().trim() === 'README.md')
      expect(fileBtn).toBeTruthy()
      await fileBtn.trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('hello-from-a')
      expect(wrapper.text()).toContain('repo-a/README.md')

      await wrapper.setProps({ layerId: 'layer_b' })
      await flushPromises()
      expect(wrapper.text()).toContain('other')
      expect(wrapper.text()).not.toContain('hello-from-a')

      await wrapper.setProps({ layerId: 'layer_a' })
      await flushPromises()
      expect(wrapper.text()).toContain('hello-from-a')
      expect(wrapper.text()).toContain('repo-a/README.md')

      const contentFetches = hoistedMocks.apiFetchMock.mock.calls.filter(([url]) =>
        String(url).includes('container-layer-file-content/')
      )
      // 初次选中 + 切回恢复，至少两次内容拉取
      expect(contentFetches.length).toBeGreaterThanOrEqual(2)
      wrapper.unmount()
    })

    it('选中目录（git 预览）切层再切回，应恢复 git 预览', async () => {
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        const u = String(url || '')
        const children = parseChildrenUrl(u)
        if (children) {
          if (children.layer === 'layer_a') {
            return Promise.resolve(jsonResponse({
              entries: filesToEntries(['repo-a/README.md']),
            }))
          }
          return Promise.resolve(jsonResponse({
            entries: filesToEntries(['other/x.txt']),
          }))
        }
        if (u.includes('container-layer-git-log/')) {
          return Promise.resolve(
            jsonResponse({
              text: 'abc 2026-01-01 init',
              commits: [{ hash: 'abc' }],
              is_repo_root: true,
              current_branch: 'main',
            })
          )
        }
        return Promise.resolve(jsonResponse({}))
      })

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_a',
          expandByDefault: true,
        },
      })
      await flushPromises()

      const dirBtn = wrapper.findAll('button').find((b) => b.text().trim() === 'repo-a')
      expect(dirBtn).toBeTruthy()
      await dirBtn.trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('提交日志')
      expect(wrapper.text()).toContain('abc 2026-01-01 init')
      expect(wrapper.find('[data-testid="git-log-current-branch"]').text()).toContain('main')

      await wrapper.setProps({ layerId: 'layer_b' })
      await flushPromises()
      expect(wrapper.text()).not.toContain('abc 2026-01-01 init')

      await wrapper.setProps({ layerId: 'layer_a' })
      await flushPromises()
      expect(wrapper.text()).toContain('提交日志')
      expect(wrapper.text()).toContain('abc 2026-01-01 init')
      wrapper.unmount()
    })

    it('切回层时缓存 path 已不在文件列表中：静默清缓存且不请求预览、不展示错误', async () => {
      let layerAFilesCall = 0
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        const u = String(url || '')
        const children = parseChildrenUrl(u)
        if (children) {
          if (children.layer === 'layer_a') {
            // 仅统计根列表请求（无 dir），避免展开懒加载干扰计数
            if (!children.dir) {
              layerAFilesCall += 1
              if (layerAFilesCall === 1) {
                return Promise.resolve(jsonResponse({
                  entries: filesToEntries(['repo-a/README.md']),
                }))
              }
              return Promise.resolve(jsonResponse({
                entries: filesToEntries(['repo-a/only-new.txt']),
              }))
            }
            return Promise.resolve(jsonResponse({ entries: [] }))
          }
          return Promise.resolve(jsonResponse({
            entries: filesToEntries(['other/x.txt']),
          }))
        }
        if (u.includes('container-layer-file-content/')) {
          return Promise.resolve(
            jsonResponse({ path: 'repo-a/README.md', content: 'stale-content', truncated: false })
          )
        }
        return Promise.resolve(jsonResponse({}))
      })

      const wrapper = mount(TaskDetailProjectFileTree, {
        props: {
          layerId: 'layer_a',
          expandByDefault: true,
        },
      })
      await flushPromises()

      await wrapper.find('[data-testid="project-file-tree-toggle-dir"]').trigger('click')
      await flushPromises()
      const fileBtn = wrapper.findAll('button').find((b) => b.text().trim() === 'README.md')
      await fileBtn.trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('stale-content')

      await wrapper.setProps({ layerId: 'layer_b' })
      await flushPromises()

      const contentCallsBeforeReturn = hoistedMocks.apiFetchMock.mock.calls.filter(([url]) =>
        String(url).includes('container-layer-file-content/')
      ).length

      await wrapper.setProps({ layerId: 'layer_a' })
      await flushPromises()

      // 不应再请求已消失文件的内容预览
      const contentCallsAfterReturn = hoistedMocks.apiFetchMock.mock.calls.filter(([url]) =>
        String(url).includes('container-layer-file-content/')
      ).length
      expect(contentCallsAfterReturn).toBe(contentCallsBeforeReturn)

      expect(wrapper.text()).not.toContain('stale-content')
      expect(wrapper.find('[data-testid="project-file-tree-error"]').exists()).toBe(false)
      // 预览区保持空态提示，而非错误文案
      expect(wrapper.text()).toContain('点击左侧文件可在此查看内容')
      expect(wrapper.text()).toContain('only-new.txt')
      wrapper.unmount()
    })
  })
}
