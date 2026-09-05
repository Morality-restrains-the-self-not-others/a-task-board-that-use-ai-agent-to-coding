// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] App.toast.traceId.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, afterEach } = await import('vitest')
  const { mount } = await import('@vue/test-utils')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ name: 'dashboard' }),
  }))

  const { default: App } = await import('./App.vue')
  const toastService = (await import('./utils/toastService.js')).default

  const mountApp = () =>
    mount(App, {
      global: {
        stubs: {
          ModalUi: { template: '<div />' },
          PrivacyReconsentGate: { template: '<div />' },
          'router-view': { template: '<div />' },
        },
      },
    })

  // 注意：DOM 中属性名小写化为 data-traceid，getAttribute 不区分大小写，
  // 与浏览器/页面元素工具读取行为一致（test-utils attributes() 为精确匹配）。
  const traceIdOf = (wrapper) => wrapper.element.getAttribute('data-traceId')

  describe('App 全局 Toast data-traceId', () => {
    afterEach(() => {
      toastService.hide()
      vi.useRealTimers()
    })

    it('error toast 带 traceId 时，根容器与消息 <p> 均渲染 data-traceId', async () => {
      toastService.show('授权未完成（exchange_rejected）', 'error', 3000, {
        traceId: 'trc-123456',
      })
      const wrapper = mountApp()
      const root = wrapper.find('.fixed.top-4.right-4')
      const msg = wrapper.find('p.text-sm.font-medium')
      expect(root.exists()).toBe(true)
      expect(msg.exists()).toBe(true)
      expect(traceIdOf(root)).toBe('trc-123456')
      expect(traceIdOf(msg)).toBe('trc-123456')
      expect(msg.text()).toBe('授权未完成（exchange_rejected）')
    })

    it('error toast 无 traceId 时，两者均不带 data-traceId 属性', () => {
      toastService.error('网络错误')
      const wrapper = mountApp()
      const root = wrapper.find('.fixed.top-4.right-4')
      const msg = wrapper.find('p.text-sm.font-medium')
      expect(traceIdOf(root)).toBeNull()
      expect(traceIdOf(msg)).toBeNull()
    })

    it('success toast 即使带 traceId 也不渲染（仅 error 类型标记）', () => {
      toastService.success('保存成功', { traceId: 'trc-999' })
      const wrapper = mountApp()
      const msg = wrapper.find('p.text-sm.font-medium')
      expect(traceIdOf(msg)).toBeNull()
    })
  })
}
