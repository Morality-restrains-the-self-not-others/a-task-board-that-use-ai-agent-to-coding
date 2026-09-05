// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskParentDeliverableField.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: CreateTaskParentDeliverableField } = await import('./CreateTaskParentDeliverableField.vue')

  const taskTypes = [
    { id: 'd1', name: '价值流', order: 0 },
    { id: 'd2', name: '活动', order: 1 },
  ]

  describe('CreateTaskParentDeliverableField', () => {
    it('上层交付物下拉仅列出上一层交付物任务标题', () => {
      const wrapper = mount(CreateTaskParentDeliverableField, {
        props: {
          modelValue: '',
          taskTypes,
          selectedCategoryId: 'd2',
          todos: [
            { id: '850256677331014872', title: '价值流A', deliverable_obj_id: 'd1' },
            { id: 'p2', title: '活动B', deliverable_obj_id: 'd2' },
          ],
        },
      })
      const optionTexts = wrapper.findAll('option').map((o) => o.text())
      expect(optionTexts.some((t) => t === '价值流A')).toBe(true)
      expect(optionTexts.some((t) => t.includes('活动B'))).toBe(false)
      wrapper.unmount()
    })

    it('上层交付物下拉展示任务编号', () => {
      const wrapper = mount(CreateTaskParentDeliverableField, {
        props: {
          modelValue: '',
          taskTypes,
          selectedCategoryId: 'd2',
          todos: [
            {
              id: '850256677331014872',
              title: '价值流A',
              deliverable_obj_id: 'd1',
              workspace_seq: 12,
            },
          ],
        },
      })
      const optionTexts = wrapper.findAll('option').map((o) => o.text().trim())
      expect(optionTexts).toContain('#12 价值流A')
      wrapper.unmount()
    })
  })
}
