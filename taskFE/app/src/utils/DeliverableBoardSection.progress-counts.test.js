// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] DeliverableBoardSection.progress-counts.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const { default: DeliverableBoardSection } = await import('../components/DeliverableBoardSection.vue')

  const threeColumns = [
    { id: 'a', name: '待处理', color: '#ff6b6b', count: 2 },
    { id: 'b', name: '进行中', color: '#4dabf7', count: 5 },
    { id: 'c', name: '已完成', color: '#51cf66', count: 1 },
  ]

  describe('DeliverableBoardSection column counts', () => {
    it('在分区标题旁展示每个进度列的数量', () => {
      const wrapper = mount(DeliverableBoardSection, {
        props: {
          title: 'runIt',
          titleClass: 'text-indigo-800',
          expanded: true,
          columnCounts: threeColumns,
        },
      })

      expect(wrapper.get('h3').text()).toBe('runIt')
      const items = wrapper.findAll('[data-alias="deliverable-section-column-count"]')
      expect(items).toHaveLength(3)
      expect(items.map((n) => n.text())).toEqual(['2', '5', '1'])
      expect(items.map((n) => n.attributes('title'))).toEqual(['待处理', '进行中', '已完成'])
      expect(wrapper.find('[data-alias="deliverable-section-count"]').exists()).toBe(true)
    })

    it('无 columnCounts 时不展示计数条', () => {
      const wrapper = mount(DeliverableBoardSection, {
        props: {
          title: '其他（未过滤）',
        },
      })
      expect(wrapper.find('[data-alias="deliverable-section-count"]').exists()).toBe(false)
      expect(wrapper.get('h3').text()).toBe('其他（未过滤）')
    })
  })
}
