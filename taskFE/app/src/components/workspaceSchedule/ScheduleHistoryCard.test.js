// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ScheduleHistoryCard.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: ScheduleHistoryCard } = await import('./ScheduleHistoryCard.vue')

  describe('ScheduleHistoryCard', () => {
    const formatTime = (iso) => iso ? 'fmt:' + iso : ''

    it('空列表展示空态', () => {
      const w = mount(ScheduleHistoryCard, {
        props: { items: [], formatTime },
      })
      expect(w.get('[data-testid="schedule-history-empty"]').text()).toContain('尚无调度记录')
    })

    it('渲染时间、类型、文案与任务链接', () => {
      const w = mount(ScheduleHistoryCard, {
        props: {
          formatTime,
          items: [{
            id: 'qsh_1',
            event_type: 'member_enqueued',
            message: '加入自动调度队列：顶层任务',
            created_at: '2026-08-28T00:00:00Z',
            task_id: 'ta',
            task_title: '顶层任务',
            href: '/tenant/t1/workspace/ws9/task-detail/ta/',
          }],
        },
      })
      expect(w.get('[data-testid="schedule-history-time"]').text()).toContain('fmt:')
      expect(w.get('[data-testid="schedule-history-type"]').text()).toBe('加入队列')
      expect(w.get('[data-testid="schedule-history-message"]').text()).toContain('顶层任务')
      const a = w.get('[data-testid="schedule-history-task-ta"]')
      expect(a.element.tagName).toBe('A')
      expect(a.attributes('href')).toBe('/tenant/t1/workspace/ws9/task-detail/ta/')
    })

    it('hasMore 时加载更多发出 load-more', async () => {
      const w = mount(ScheduleHistoryCard, {
        props: {
          formatTime,
          hasMore: true,
          items: [{ id: 'qsh_1', event_type: 'rhythm_saved', message: '已保存', created_at: '2026-08-28T00:00:00Z' }],
        },
      })
      await w.get('[data-testid="schedule-history-load-more"]').trigger('click')
      expect(w.emitted('load-more')).toHaveLength(1)
    })

    it('hasMore=false 时不渲染加载更多按钮', () => {
      const w = mount(ScheduleHistoryCard, {
        props: {
          formatTime,
          hasMore: false,
          items: [{ id: 'qsh_1', event_type: 'rhythm_saved', message: '已保存', created_at: '2026-08-28T00:00:00Z' }],
        },
      })
      expect(w.find('[data-testid="schedule-history-load-more"]').exists()).toBe(false)
    })

    it('错误节点挂 data-traceId', () => {
      const w = mount(ScheduleHistoryCard, {
        props: { formatTime, items: [], loadError: 'boom', loadErrorTraceId: 'tr-hist' },
      })
      const el = w.get('[data-testid="schedule-history-error"]').element
      let tid
      for (const attr of el.attributes) {
        if (attr.name.toLowerCase() === 'data-traceid') tid = attr.value
      }
      expect(tid).toBe('tr-hist')
      expect(w.get('[data-testid="schedule-history-error"]').text()).toBe('boom')
    })
  })
}
