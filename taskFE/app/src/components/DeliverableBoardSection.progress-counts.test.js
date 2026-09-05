// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] DeliverableBoardSection.progress-counts.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const { default: DeliverableBoardSection } = await import('./DeliverableBoardSection.vue')

  const __dirname = dirname(fileURLToPath(import.meta.url))
  const sourceVue = readFileSync(join(__dirname, 'DeliverableBoardSection.vue'), 'utf8')

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

  describe('DeliverableBoardSection --fill CSS 契约（OPT-20260810-049）', () => {
    it('fill 分区 flex: 1 0 0% + min-height: 280px，空间不足不压缩', () => {
      const fillRule = sourceVue.match(/\.deliverable-board-section--fill\s*\{[^}]*\}/)?.[0] || ''
      expect(fillRule).toMatch(/flex:\s*1\s+0\s+0%/)
      expect(fillRule).toMatch(/min-height:\s*280px/)
      // 与旧版 flex:1 1 0% / min-height:0 相反：不压缩、保证最低 280px 贴底
      expect(fillRule).not.toMatch(/min-height:\s*0/)
    })

    it('fillRemaining 仅在展开时给 section 挂 --fill 类', () => {
      expect(sourceVue).toMatch(/deliverable-board-section--fill': fillRemaining && expanded/)
    })
  })
}
