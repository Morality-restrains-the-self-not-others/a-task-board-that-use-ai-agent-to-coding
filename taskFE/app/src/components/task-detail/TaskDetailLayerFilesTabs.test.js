// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailLayerFilesTabs.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailLayerFilesTabs } = await import('./TaskDetailLayerFilesTabs.vue')

  function mountTabs(props = {}) {
    return mount(TaskDetailLayerFilesTabs, {
      props: {
        layerId: 'layer-1',
        changesCount: 0,
        ...props,
      },
      slots: {
        tree: '<div data-testid="slot-tree">tree body</div>',
        changes: '<div data-testid="slot-changes">changes body</div>',
      },
    })
  }

  describe('TaskDetailLayerFilesTabs', () => {
    it('defaults to the file changes tab and keeps both slots mounted', () => {
      const wrapper = mountTabs({ changesCount: 3 })
      const tabs = wrapper.get('[data-testid="layer-files-tablist"]').findAll('[role="tab"]')
      expect(tabs).toHaveLength(2)
      expect(tabs[0].attributes('data-testid')).toBe('layer-files-tab-tree')
      expect(tabs[0].attributes('aria-selected')).toBe('false')
      expect(tabs[0].text()).toContain('项目文件树')
      expect(tabs[1].attributes('data-testid')).toBe('layer-files-tab-changes')
      expect(tabs[1].attributes('aria-selected')).toBe('true')
      expect(tabs[1].text()).toContain('文件变动')
      expect(tabs[1].text()).toContain('· 3')
      expect(wrapper.get('[data-testid="layer-files-panel-changes"]').element.style.display).not.toBe('none')
      expect(wrapper.get('[data-testid="layer-files-panel-changes"]').get('[data-testid="slot-changes"]').text()).toBe('changes body')
      expect(wrapper.get('[data-testid="layer-files-panel-tree"]').element.style.display).toBe('none')
      expect(wrapper.get('[data-testid="layer-files-panel-tree"]').get('[data-testid="slot-tree"]').text()).toBe('tree body')
    })

    it('switches to the file tree and back without unmounting the tree slot', async () => {
      const wrapper = mountTabs()
      expect(wrapper.get('[data-testid="layer-files-tab-changes"]').attributes('aria-selected')).toBe('true')
      await wrapper.get('[data-testid="layer-files-tab-tree"]').trigger('click')
      expect(wrapper.get('[data-testid="layer-files-tab-tree"]').attributes('aria-selected')).toBe('true')
      expect(wrapper.get('[data-testid="layer-files-panel-changes"]').element.style.display).toBe('none')
      expect(wrapper.get('[data-testid="layer-files-panel-tree"]').element.style.display).not.toBe('none')
      expect(wrapper.get('[data-testid="layer-files-panel-tree"]').get('[data-testid="slot-tree"]').text()).toBe('tree body')
      await wrapper.get('[data-testid="layer-files-tab-changes"]').trigger('click')
      expect(wrapper.get('[data-testid="layer-files-panel-changes"]').element.style.display).not.toBe('none')
      expect(wrapper.get('[data-testid="layer-files-panel-tree"]').element.style.display).toBe('none')
    })

    it('resets to the file changes tab when layerId changes', async () => {
      const wrapper = mountTabs()
      await wrapper.get('[data-testid="layer-files-tab-tree"]').trigger('click')
      expect(wrapper.get('[data-testid="layer-files-tab-tree"]').attributes('aria-selected')).toBe('true')
      await wrapper.setProps({ layerId: 'layer-2' })
      await nextTick()
      expect(wrapper.get('[data-testid="layer-files-tab-changes"]').attributes('aria-selected')).toBe('true')
      expect(wrapper.get('[data-testid="layer-files-panel-changes"]').element.style.display).not.toBe('none')
    })

    it('omits the count suffix when changesCount is 0', () => {
      const wrapper = mountTabs({ changesCount: 0 })
      expect(wrapper.get('[data-testid="layer-files-tab-changes"]').text().trim()).toBe('文件变动')
    })
  })
}
