// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] Modal.ui.traceId.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: ModalUi } = await import('./Modal.ui.vue')

describe('Modal.ui data-traceId on message', () => {
  it('sets data-traceId on error message <p> when traceId present', () => {
    const wrapper = mount(ModalUi, {
      props: {
        show: true,
        title: '错误',
        message: '派生任务失败：自动运行需要镜像配置运行环境',
        type: 'alert',
        traceId: 'tid-modal-fork-1',
      },
    })
    const p = wrapper.find('div.space-y-4 p')
    expect(p.exists()).toBe(true)
    expect(p.attributes('data-traceid') || p.attributes('data-traceId')).toBe('tid-modal-fork-1')
    const root = wrapper.find('.bg-white.rounded-lg')
    expect(root.attributes('data-traceid') || root.attributes('data-traceId')).toBe('tid-modal-fork-1')
  })

  it('omits data-traceId on message <p> when traceId empty', () => {
    const wrapper = mount(ModalUi, {
      props: {
        show: true,
        title: '错误',
        message: '纯前端提示',
        type: 'alert',
        traceId: '',
      },
    })
    const p = wrapper.find('div.space-y-4 p')
    expect(p.attributes('data-traceid') || p.attributes('data-traceId')).toBeUndefined()
  })

  it('showCloseButton=false 时点击遮罩不关闭', async () => {
    const wrapper = mount(ModalUi, {
      props: {
        show: true,
        title: '请验证手机号',
        message: '须先完成手机号验证',
        type: 'alert',
        showCloseButton: false,
        confirmText: '去验证',
      },
    })
    expect(wrapper.find('h3').text()).toBe('请验证手机号')
    expect(wrapper.findAll('button').every((b) => b.text() !== '稍后再说')).toBe(true)
    await wrapper.find('.app-modal-overlay').trigger('click')
    expect(wrapper.emitted('close')).toBeUndefined()
  })
})
}
