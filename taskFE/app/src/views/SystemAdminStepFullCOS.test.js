// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminStepFullCOS.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))

  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))

  const { default: Page } = await import('./SystemAdminStepFullCOS.vue')

  function okJson(body) {
    return {
      ok: true,
      json: async () => body,
      clone() { return this },
      headers: { get: () => null },
    }
  }

  describe('SystemAdminStepFullCOS', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue(okJson({
        backend: 'local',
        bucket: '',
        region: 'ap-guangzhou',
        pathRule: 'workspace_{workspaceId}/task_{taskId}/comment_{commentId}/layer_{layerId}/step_full.json',
        startupLogsPathRule: 'workspace_{workspaceId}/task_{taskId}/comment_{commentId}/startup_logs.json',
        keyPrefix: '',
        secret_configured: false,
      }))
    })

    it('loads COS settings without leaking secret keys', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      expect(apiFetch).toHaveBeenCalledWith('/api/system-admin/step-full-cos/')
      expect(wrapper.text()).not.toContain('AKIDxxx')
      expect(wrapper.get('[data-testid="step-full-cos-backend"]').element.value).toBe('local')
      expect(wrapper.get('[data-testid="step-full-cos-startup-logs-path-rule"]').element.value).toContain('startup_logs.json')
    })

    it('places secret status next to the page heading, not after secret inputs', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      const status = wrapper.get('[data-testid="step-full-cos-secret-status"]')
      expect(status.text()).toBe('当前密钥状态：未配置')
      const heading = wrapper.get('h2')
      expect(heading.element.parentElement.contains(status.element)).toBe(true)
      const html = wrapper.html()
      expect(html.indexOf('step-full-cos-secret-status')).toBeLessThan(html.indexOf('step-full-cos-secret-id'))
      expect(html.indexOf('step-full-cos-secret-status')).toBeLessThan(html.indexOf('step-full-cos-save'))
    })

    it('GET 403 读取失败时密钥状态展示未知而非未配置', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 403,
        clone() { return this },
        json: async () => ({ detail: '无权限' }),
        headers: { get: () => null },
      })
      const wrapper = mount(Page)
      await flushPromises()
      expect(apiFetch).toHaveBeenCalledWith('/api/system-admin/step-full-cos/')
      const status = wrapper.get('[data-testid="step-full-cos-secret-status"]')
      expect(status.text()).not.toContain('未配置')
      expect(status.text()).toContain('未知')
      wrapper.unmount()
    })

    it('shows configured secret status after load when secret_configured is true', async () => {
      apiFetch.mockResolvedValue(okJson({
        backend: 'cos',
        bucket: 'b1',
        region: 'ap-guangzhou',
        pathRule: 'workspace_{workspaceId}/task_{taskId}/comment_{commentId}/layer_{layerId}/step_full.json',
        startupLogsPathRule: 'workspace_{workspaceId}/task_{taskId}/comment_{commentId}/startup_logs.json',
        keyPrefix: '',
        secret_configured: true,
      }))
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.get('[data-testid="step-full-cos-secret-status"]').text()).toBe('当前密钥状态：已配置')
    })

    it('double click saves once with the same Idempotency-Key', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      apiFetch.mockResolvedValue(okJson({
        backend: 'cos',
        bucket: 'b1',
        region: 'ap-guangzhou',
        pathRule: 'workspace_{workspaceId}/task_{taskId}/comment_{commentId}/layer_{layerId}/step_full.json',
        secret_configured: true,
      }))
      const btn = wrapper.get('[data-testid="step-full-cos-save"]')
      await Promise.all([btn.trigger('click'), btn.trigger('click')])
      await flushPromises()
      const patches = apiFetch.mock.calls.filter((c) => c[1] && c[1].method === 'PATCH')
      expect(patches).toHaveLength(1)
      expect(patches[0][1].headers['Idempotency-Key']).toBeTruthy()
      expect(wrapper.get('[data-testid="step-full-cos-secret-status"]').text()).toBe('当前密钥状态：已配置')
    })
  })
}
