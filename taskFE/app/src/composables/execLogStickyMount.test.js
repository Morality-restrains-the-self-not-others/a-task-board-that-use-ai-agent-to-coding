// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] execLogStickyMount.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { nextTick } = await import('vue')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailExecLiveOutput } = await import('../components/task-detail/TaskDetailExecLiveOutput.vue')
const { default: TaskDetailExecCloneLogSection } = await import('../components/task-detail/TaskDetailExecCloneLogSection.vue')

function tallText(lines) {
  return Array.from({ length: lines }, (_, i) => `line-${i} ${'x'.repeat(40)}`).join('\n')
}

describe('执行日志虚拟列表 + sticky（组件挂载）', () => {
  it('LiveOutput：超长文本只渲染可见行，滚动容器身份稳定', async () => {
    const initial = tallText(200)
    const wrapper = mount(TaskDetailExecLiveOutput, {
      props: { liveOutputDisplay: initial },
      attachTo: document.body,
    })
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))

    const viewport = wrapper.get('[data-testid="layer-live-output-scroll"]').element
    const body = wrapper.get('[data-testid="virtual-log-body"]').element
    Object.defineProperty(viewport, 'clientHeight', { configurable: true, get: () => 128 })
    // 触发一次 scroll 重算窗口
    viewport.dispatchEvent(new Event('scroll'))
    await new Promise((r) => requestAnimationFrame(r))

    expect(body.textContent.length).toBeLessThan(initial.length)
    const topPad = wrapper.get('[data-testid="virtual-log-top-pad"]').element
    expect(topPad.style.height === '' || topPad.style.height.endsWith('px')).toBe(true)
    const vp1 = viewport

    await wrapper.setProps({ liveOutputDisplay: tallText(220) })
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    const vp2 = wrapper.get('[data-testid="layer-live-output-scroll"]').element
    expect(vp2).toBe(vp1)
    wrapper.unmount()
  })

  it('LiveOutput：内容从有到空再到有时滚动容器 DOM 节点保持同一引用', async () => {
    const wrapper = mount(TaskDetailExecLiveOutput, {
      props: { liveOutputDisplay: tallText(10) },
      attachTo: document.body,
    })
    await nextTick()
    const pre1 = wrapper.get('[data-testid="layer-live-output-scroll"]').element
    await wrapper.setProps({ liveOutputDisplay: '' })
    await nextTick()
    const pre2 = wrapper.get('[data-testid="layer-live-output-scroll"]').element
    await wrapper.setProps({ liveOutputDisplay: tallText(12) })
    await nextTick()
    const pre3 = wrapper.get('[data-testid="layer-live-output-scroll"]').element
    expect(pre2).toBe(pre1)
    expect(pre3).toBe(pre1)
    wrapper.unmount()
  })

  it('LiveOutput：非贴底时追加后 sticky 恢复 scrollTop', async () => {
    const wrapper = mount(TaskDetailExecLiveOutput, {
      props: { liveOutputDisplay: tallText(80) },
      attachTo: document.body,
    })
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    const viewport = wrapper.get('[data-testid="layer-live-output-scroll"]').element
    Object.defineProperty(viewport, 'clientHeight', { configurable: true, get: () => 128 })
    Object.defineProperty(viewport, 'scrollHeight', {
      configurable: true,
      get: () => 80 * 18,
    })
    viewport.scrollTop = 200
    viewport.dispatchEvent(new Event('scroll'))

    await wrapper.setProps({ liveOutputDisplay: tallText(90) })
    viewport.scrollTop = 0
    await new Promise((r) => requestAnimationFrame(r))
    expect(viewport.scrollTop).toBe(200)
    wrapper.unmount()
  })

  it('CloneLog：贴底时追加后跟随到底', async () => {
    const wrapper = mount(TaskDetailExecCloneLogSection, {
      props: {
        layerId: 'L1',
        cloneLogText: tallText(40),
        cloneLogFetchError: '',
      },
      attachTo: document.body,
    })
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    const viewport = wrapper.get('[data-testid="layer-clone-log-scroll"]').element
    Object.defineProperty(viewport, 'clientHeight', { configurable: true, get: () => 160 })
    let scrollHeight = 800
    Object.defineProperty(viewport, 'scrollHeight', { configurable: true, get: () => scrollHeight })
    viewport.scrollTop = 800 - 160
    viewport.dispatchEvent(new Event('scroll'))

    await wrapper.setProps({ cloneLogText: tallText(80) })
    scrollHeight = 1600
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))

    expect(viewport.scrollTop).toBe(1600)
    wrapper.unmount()
  })
})

}
