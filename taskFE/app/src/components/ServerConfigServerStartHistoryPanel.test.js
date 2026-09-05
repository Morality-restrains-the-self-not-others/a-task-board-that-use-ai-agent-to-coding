// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ServerConfigServerStartHistoryPanel.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: ServerConfigServerStartHistoryPanel } = await import(
    './ServerConfigServerStartHistoryPanel.vue'
  )

  const EMPTY_TEXT = '暂无历史服务器启动记录'

  describe('ServerConfigServerStartHistoryPanel 空态单条（OPT-20260811-054）', () => {
    it('空成功（records=[]，message 为空）仅渲染一条空态文案', () => {
      const wrapper = mount(ServerConfigServerStartHistoryPanel, {
        props: { records: [], loading: false, message: '', messageTraceId: '' },
      })
      expect(wrapper.text().includes(EMPTY_TEXT)).toBe(true)
      const count = wrapper.findAll('*').filter((n) => n.text() === EMPTY_TEXT).length
      // 仅面板内空态 div 一条；message 为空时不渲染 message div
      expect(count).toBe(1)
      expect(wrapper.find('[data-testid="server-start-history-message"]').exists()).toBe(false)
    })

    it('空态文案不因 message 与空态双渲染而重复', () => {
      const wrapper = mount(ServerConfigServerStartHistoryPanel, {
        props: { records: [], loading: false, message: EMPTY_TEXT, messageTraceId: '' },
      })
      // 即便上游（旧逻辑）写入 message=空态文案，面板也只显示一条（v-else-if 含 !message）
      expect(wrapper.findAll('*').filter((n) => n.text() === EMPTY_TEXT).length).toBe(1)
      expect(wrapper.find('[data-testid="server-start-history-message"]').text()).toBe(EMPTY_TEXT)
    })

    it('有记录时渲染卡片而非空态', () => {
      const wrapper = mount(ServerConfigServerStartHistoryPanel, {
        props: {
          records: [{ id: 'r1', created_at: '2026-07-01T00:00:00Z', platform: '阿里云' }],
          loading: false,
          message: '共 1 条历史记录',
          messageTraceId: '',
        },
      })
      expect(wrapper.findAll('[data-testid="server-start-history-card"]').length).toBe(1)
      expect(wrapper.text().includes(EMPTY_TEXT)).toBe(false)
    })
  })
}
