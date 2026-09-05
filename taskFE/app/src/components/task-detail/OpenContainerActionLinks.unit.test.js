// @vitest-environment jsdom
/**
 * OPT-20260810-034：任务关联区「打开容器页面/打开容器开发页面」共享链接组件。
 * 验证保留 #open-container-page-btn / #open-container-vscode-btn id、URL 透传
 * 与不可达/未就绪时隐藏逻辑（原分散在两个父组件，这里锁定共享行为防漂移）。
 */
if (!process.env.VITEST) {
  console.log('[skip] OpenContainerActionLinks.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount } = await import('@vue/test-utils')
  const { default: OpenContainerActionLinks } = await import('./OpenContainerActionLinks.vue')

  const PAGE_URL = 'http://203.0.113.10:8765/ui/tok'
  const VSCODE_URL = 'http://203.0.113.10:18888/'

  function mountLinks(overrides = {}) {
    return mount(OpenContainerActionLinks, {
      props: {
        containerPageUrl: PAGE_URL,
        displayContainerVscodeUrl: VSCODE_URL,
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task1',
        pendingReveal: false,
        httpUnreachable: false,
        ...overrides,
      },
    })
  }

  describe('OpenContainerActionLinks', () => {
    it('渲染两个 id 且 href/target 正确', async () => {
      const wrapper = mountLinks()
      await nextTick()
      const pageBtn = wrapper.find('#open-container-page-btn')
      const vscodeBtn = wrapper.find('#open-container-vscode-btn')
      expect(pageBtn.exists()).toBe(true)
      expect(pageBtn.attributes('href')).toBe(PAGE_URL)
      expect(pageBtn.attributes('target')).toBe('_blank')
      expect(pageBtn.text()).toContain('打开容器页面')
      expect(vscodeBtn.exists()).toBe(true)
      expect(vscodeBtn.attributes('href')).toBe(VSCODE_URL)
      expect(vscodeBtn.text()).toContain('打开容器开发页面')
      wrapper.unmount()
    })

    it('无 vscode URL 时不渲染「打开容器开发页面」', async () => {
      const wrapper = mountLinks({ displayContainerVscodeUrl: '' })
      await nextTick()
      expect(wrapper.find('#open-container-page-btn').exists()).toBe(true)
      expect(wrapper.find('#open-container-vscode-btn').exists()).toBe(false)
      wrapper.unmount()
    })

    it('容器不可达或链接未就绪时整组隐藏', async () => {
      const unreachable = mountLinks({ httpUnreachable: true })
      await nextTick()
      expect(unreachable.find('#open-container-page-btn').exists()).toBe(false)
      expect(unreachable.find('#open-container-vscode-btn').exists()).toBe(false)
      unreachable.unmount()

      const pending = mountLinks({ pendingReveal: true })
      await nextTick()
      expect(pending.find('#open-container-page-btn').exists()).toBe(false)
      pending.unmount()
    })

    it('透传 commentId 给 ensure-client-ingress（评论级 CSC 路径）', async () => {
      const calls = []
      vi.doMock('../../utils/openContainerPage.js', () => ({
        openContainerPageWithIngressEnsure: (opts) => {
          calls.push(opts)
        },
      }))
      vi.resetModules()
      const { mount: freshMount } = await import('@vue/test-utils')
      const { default: FreshLinks } = await import('./OpenContainerActionLinks.vue')
      const wrapper = freshMount(FreshLinks, {
        props: {
          containerPageUrl: PAGE_URL,
          displayContainerVscodeUrl: VSCODE_URL,
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: 'task1',
          commentId: 'cmt_a',
          pendingReveal: false,
          httpUnreachable: false,
        },
      })
      await wrapper.find('#open-container-page-btn').trigger('click')
      await nextTick()
      expect(calls.length).toBe(1)
      expect(calls[0].commentId).toBe('cmt_a')
      expect(calls[0].taskId).toBe('task1')
      wrapper.unmount()
      vi.unmock('../../utils/openContainerPage.js')
    })
  })
}
