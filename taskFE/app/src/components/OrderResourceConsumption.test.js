// @vitest-environment jsdom
/**
 * 订单资源消耗区块须展示本单购买的全部资源类型（任务帖 + GitLab），
 * 不得在仅购买 GitLab 时用「任务帖 已发放 0」占位。
 */
if (!process.env.VITEST) {
  console.log('[skip] OrderResourceConsumption.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const Comp = (await import('./OrderResourceConsumption.vue')).default

  describe('OrderResourceConsumption', () => {
    it('gitlab-only 订单展示磁盘已发放数量，不出现任务帖 0', () => {
      const wrapper = mount(Comp, {
        props: {
          alwaysShow: true,
          consumption: {
            gitlab_disk: {
              source_kind: 'purchase',
              granted: 10,
              consumed: 2,
              remaining: 8,
              region: 'tencent-sh-1',
              events: [],
            },
          },
        },
      })
      const text = wrapper.get('[data-testid="order-resource-consumption"]').text()
      expect(text).toContain('GitLab 磁盘')
      expect(text).toContain('已发放 10')
      expect(text).toContain('已消耗 2')
      expect(text).toContain('剩余 8')
      expect(text).toContain('tencent-sh-1')
      expect(text).not.toContain('任务帖')
      wrapper.unmount()
    })

    it('alwaysShow 且无购买资源时不伪造任务帖 0', () => {
      const wrapper = mount(Comp, {
        props: { alwaysShow: true, consumption: null },
      })
      const text = wrapper.get('[data-testid="order-resource-consumption"]').text()
      expect(text).toContain('资源消耗')
      expect(text).not.toMatch(/任务帖：已发放 0/)
      wrapper.unmount()
    })

    it('任务帖购买仍展示逐笔事件', () => {
      const wrapper = mount(Comp, {
        props: {
          consumption: {
            task_post: {
              source_kind: 'purchase',
              granted: 10,
              consumed: 3,
              remaining: 7,
              events: [
                { created_at: '2026-08-19T10:00:00Z', task_id: 'task-1', action: 'create', quantity: 1 },
              ],
            },
          },
        },
      })
      const text = wrapper.get('[data-testid="order-resource-consumption"]').text()
      expect(text).toContain('任务帖')
      expect(text).toContain('已消耗 3')
      expect(text).toContain('task-1')
      wrapper.unmount()
    })
  })
}
