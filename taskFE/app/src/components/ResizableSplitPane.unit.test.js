// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ResizableSplitPane.unit.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: ResizableSplitPane } = await import('./ResizableSplitPane.vue')

describe('ResizableSplitPane', () => {
  it('renders left/right slots and a draggable gutter', () => {
    const wrapper = mount(ResizableSplitPane, {
      props: { storageKey: '' },
      slots: {
        left: '<div data-testid="slot-left">L</div>',
        right: '<div data-testid="slot-right">R</div>',
      },
    })
    expect(wrapper.find('[data-testid="slot-left"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="slot-right"]').exists()).toBe(true)
    const root = wrapper.find('[data-testid="resizable-split-pane"]')
    expect(root.classes()).toContain('flex-row')
    expect(root.classes()).not.toContain('flex-col')
    const gutter = wrapper.find('[data-testid="resizable-split-pane-gutter"]')
    expect(gutter.exists()).toBe(true)
    expect(gutter.attributes('role')).toBe('separator')
    expect(gutter.classes()).toContain('cursor-col-resize')
    expect(gutter.classes()).toContain('flex')
    expect(gutter.classes()).not.toContain('hidden')
  })

  it('adjusts width with keyboard arrows', async () => {
    const wrapper = mount(ResizableSplitPane, {
      props: { storageKey: '', defaultWidth: 256 },
      slots: { left: '<div />', right: '<div />' },
      attachTo: document.body,
    })
    const gutter = wrapper.find('[data-testid="resizable-split-pane-gutter"]')
    Object.defineProperty(wrapper.element, 'getBoundingClientRect', {
      value: () => ({ left: 0, width: 800, top: 0, height: 100, right: 800, bottom: 100 }),
    })
    await gutter.trigger('keydown', { key: 'ArrowRight' })
    const left = wrapper.find('[data-testid="resizable-split-pane-left"]')
    expect(left.attributes('style')).toContain('--split-left-width: 272px')
    wrapper.unmount()
  })
})

}
