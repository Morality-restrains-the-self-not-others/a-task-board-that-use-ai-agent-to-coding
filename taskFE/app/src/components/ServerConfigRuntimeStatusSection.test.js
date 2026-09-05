// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ServerConfigRuntimeStatusSection.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: ServerConfigRuntimeStatusSection } = await import(
    './ServerConfigRuntimeStatusSection.vue'
  )

  describe('ServerConfigRuntimeStatusSection data-traceId', () => {
    it('binds data-traceId on status message when traceId present', () => {
      const wrapper = mount(ServerConfigRuntimeStatusSection, {
        props: {
          serverRuntimeStatusDisplayText: '未知',
          serverRuntimeStatusMessage: '未找到云平台授权信息',
          serverRuntimeStatusTraceId: 'trace-runtime-auth-1',
          fetchServerRuntimeStatus: () => {},
        },
      })
      const el = wrapper.get('[data-testid="server-runtime-status-message"]')
      expect(el.text()).toContain('未找到云平台授权信息')
      expect(el.attributes('data-traceid') || el.attributes('data-traceId')).toBe(
        'trace-runtime-auth-1',
      )
    })

    it('shows failed-start hint instead of idle missing-CSC copy', () => {
      const wrapper = mount(ServerConfigRuntimeStatusSection, {
        props: {
          serverRuntimeStatusDisplayText: '未知',
          serverRuntimeStatusMessage: '启动失败且未创建评论级服务器配置。请用启动 TraceId 排查后重试。',
          serverRuntimeStatusTraceId: 'start-trace-miss-csc',
          fetchServerRuntimeStatus: () => {},
        },
      })
      const el = wrapper.get('[data-testid="server-runtime-status-message"]')
      expect(el.text()).toContain('启动失败')
      expect(el.text()).not.toBe('未找到服务器配置记录')
      expect(el.attributes('data-traceid') || el.attributes('data-traceId')).toBe(
        'start-trace-miss-csc',
      )
    })

    it('omits data-traceId when message has no request trace', () => {
      const wrapper = mount(ServerConfigRuntimeStatusSection, {
        props: {
          serverRuntimeStatusDisplayText: '未创建',
          serverRuntimeStatusMessage: '该任务尚未创建云实例',
          serverRuntimeStatusTraceId: '',
          fetchServerRuntimeStatus: () => {},
        },
      })
      const el = wrapper.get('[data-testid="server-runtime-status-message"]')
      expect(el.attributes('data-traceid') || el.attributes('data-traceId') || '').toBe('')
    })

    it('does not query cloud on mount; only the refresh button does', async () => {
      let called = 0
      const wrapper = mount(ServerConfigRuntimeStatusSection, {
        props: {
          commentId: 'cmt_card_1',
          serverRuntimeStatusDisplayText: '运行中',
          fetchServerRuntimeStatus: () => {
            called += 1
          },
        },
      })
      expect(called).toBe(0)
      await wrapper.get('button').trigger('click')
      expect(called).toBe(1)
    })

    it('renders internet bandwidth rows in the instance details grid', () => {
      const wrapper = mount(ServerConfigRuntimeStatusSection, {
        props: {
          serverRuntimeStatusDisplayText: '运行中',
          serverRuntimeStatusDetails: [
            { label: '平台', value: 'aliyun' },
            { label: '带宽计费模式', value: '按流量计费' },
            { label: '公网出带宽', value: '5 Mbps' },
            { label: '公网入带宽', value: '5 Mbps' },
          ],
          fetchServerRuntimeStatus: () => {},
        },
      })
      const text = wrapper.text()
      expect(text).toContain('带宽计费模式：按流量计费')
      expect(text).toContain('公网出带宽：5 Mbps')
      expect(text).toContain('公网入带宽：5 Mbps')
    })
  })
}
