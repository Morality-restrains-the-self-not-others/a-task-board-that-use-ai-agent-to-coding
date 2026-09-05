// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] DeliverableBreadcrumb.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: DeliverableBreadcrumb } = await import('./DeliverableBreadcrumb.vue')

  const categories = [
    { id: 'c1', name: '价值流', color: '#111', order: 1 },
  ]

  describe('DeliverableBreadcrumb content select', () => {
    it('下拉 option 展示 #序号 与标题', () => {
      const wrapper = mount(DeliverableBreadcrumb, {
        props: {
          categories,
          todos: [
            {
              id: 't-a',
              title: '写一个 hello world程序',
              deliverable_obj_id: 'c1',
              workspace_seq: 3,
            },
            {
              id: 't-b',
              title: '写一个 hello world程序',
              deliverable_obj_id: 'c1',
              workspace_seq: 8,
            },
          ],
        },
      })
      const select = wrapper.get('[data-alias="deliverable-content-select"]')
      const optionTexts = select.findAll('option').map((o) => o.text().trim())
      expect(optionTexts).toContain('#3 写一个 hello world程序')
      expect(optionTexts).toContain('#8 写一个 hello world程序')
      const values = select.findAll('option').map((o) => o.attributes('value'))
      expect(values).toContain('t-a')
      expect(values).toContain('t-b')
      wrapper.unmount()
    })

    it('关闭态选中项可见 #序号', () => {
      const wrapper = mount(DeliverableBreadcrumb, {
        props: {
          categories,
          todos: [
            {
              id: 't-a',
              title: '写一个 hello world程序',
              deliverable_obj_id: 'c1',
              workspace_seq: 3,
            },
          ],
          path: [
            { type: 'category', id: 'c1' },
            { type: 'task', id: 't-a' },
          ],
        },
      })
      const select = wrapper.get('[data-alias="deliverable-content-select"]')
      const selectedIndex = select.element.selectedIndex
      const selectedText = select.element.options[selectedIndex]?.text.trim() || ''
      expect(select.element.value).toBe('t-a')
      expect(selectedText).toContain('#3')
      expect(selectedText).toContain('写一个 hello world程序')
      wrapper.unmount()
    })

    it('无序号时 option 仍为标题', () => {
      const wrapper = mount(DeliverableBreadcrumb, {
        props: {
          categories,
          todos: [
            { id: 't-x', title: '仅标题任务', deliverable_obj_id: 'c1' },
          ],
        },
      })
      const optionTexts = wrapper
        .get('[data-alias="deliverable-content-select"]')
        .findAll('option')
        .map((o) => o.text().trim())
      expect(optionTexts).toContain('仅标题任务')
      expect(optionTexts.some((t) => t.startsWith('#'))).toBe(false)
      wrapper.unmount()
    })
  })
}
