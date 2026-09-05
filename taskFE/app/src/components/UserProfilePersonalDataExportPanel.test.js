// @vitest-environment jsdom
// PIPL「导出权」：个人资料页个人信息导出面板。
// none → 生成按钮；ready/partial → 下载 + 重新生成；expired → 重新生成。
// request/ 返回 201 {status, export_id, generated_at, expires_at, sections_unavailable}；
// download/ 返回 JSON blob（attachment）。
if (!process.env.VITEST) {
  console.log('[skip] UserProfilePersonalDataExportPanel.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  vi.mock('../utils/apiUtils', () => ({ apiFetch: apiFetchMock }))

  const { default: UserProfilePersonalDataExportPanel } = await import('./UserProfilePersonalDataExportPanel.vue')

  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  const statusPayload = (status) => ({
    ok: true,
    status: 200,
    json: async () => ({
      status,
      export_id: '12345',
      generated_at: '2026-08-24T08:00:00Z',
      expires_at: '2026-08-31T08:00:00Z',
      retention_days: 7,
    }),
  })

  const mountPanel = () => mount(UserProfilePersonalDataExportPanel)

  describe('UserProfilePersonalDataExportPanel 渲染', () => {
    it('无导出：显示生成按钮', async () => {
      apiFetchMock.mockResolvedValue(statusPayload('none'))
      const wrapper = mountPanel()
      await flushPromises()
      expect(wrapper.text()).toContain('个人信息导出')
      expect(wrapper.text()).toContain('生成导出文件')
      expect(wrapper.find('button').text()).toContain('生成导出文件')
    })

    it('已有导出：显示下载与重新生成按钮', async () => {
      apiFetchMock.mockResolvedValue(statusPayload('ready'))
      const wrapper = mountPanel()
      await flushPromises()
      expect(wrapper.text()).toContain('下载 JSON')
      expect(wrapper.text()).toContain('重新生成')
      expect(wrapper.text()).toContain('有效期至')
    })

    it('partial：提示部分数据源不可用', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          status: 'partial',
          export_id: '12345',
          generated_at: '2026-08-24T08:00:00Z',
          expires_at: '2026-08-31T08:00:00Z',
          retention_days: 7,
        }),
      })
      const wrapper = mountPanel()
      await flushPromises()
      expect(wrapper.text()).toContain('部分数据源暂不可用')
      expect(wrapper.text()).toContain('下载 JSON')
    })

    it('expired：仅显示重新生成', async () => {
      apiFetchMock.mockResolvedValue(statusPayload('expired'))
      const wrapper = mountPanel()
      await flushPromises()
      expect(wrapper.text()).toContain('已过期')
      expect(wrapper.text()).toContain('重新生成导出文件')
      expect(wrapper.text()).not.toContain('下载 JSON')
    })
  })

  describe('UserProfilePersonalDataExportPanel 交互', () => {
    it('生成导出：POST request/ 并切换为可下载状态', async () => {
      apiFetchMock.mockResolvedValueOnce(statusPayload('none'))
      apiFetchMock.mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({
          status: 'ready',
          export_id: '12345',
          generated_at: '2026-08-24T08:00:00Z',
          expires_at: '2026-08-31T08:00:00Z',
          retention_days: 7,
        }),
      })
      const wrapper = mountPanel()
      await flushPromises()
      await wrapper.find('button').trigger('click')
      await flushPromises()
      expect(apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/me/personal-data-export/request/',
        expect.objectContaining({ method: 'POST' }),
      )
      expect(wrapper.text()).toContain('下载 JSON')
    })

    it('下载导出：GET download/ 触发浏览器下载', async () => {
      apiFetchMock.mockResolvedValueOnce(statusPayload('ready'))
      apiFetchMock.mockResolvedValueOnce({
        ok: true,
        status: 200,
        blob: async () => new Blob(['{"format_version":1}'], { type: 'application/json' }),
      })
      const createObjectURL = vi.fn(() => 'blob:mock-url')
      const revokeObjectURL = vi.fn()
      URL.createObjectURL = createObjectURL
      URL.revokeObjectURL = revokeObjectURL
      const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
      const wrapper = mountPanel()
      await flushPromises()
      await wrapper.findAll('button').find((b) => b.text().includes('下载 JSON')).trigger('click')
      await flushPromises()
      expect(apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/me/personal-data-export/download/',
        expect.anything(),
      )
      expect(createObjectURL).toHaveBeenCalled()
      expect(clickSpy).toHaveBeenCalled()
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock-url')
      clickSpy.mockRestore()
    })

    it('下载过期：410 后切换为 expired 状态', async () => {
      apiFetchMock.mockResolvedValueOnce(statusPayload('ready'))
      apiFetchMock.mockResolvedValueOnce({
        ok: false,
        status: 410,
        json: async () => ({ error: 'export_expired', detail: '导出已过期，请重新发起导出请求' }),
      })
      const wrapper = mountPanel()
      await flushPromises()
      await wrapper.findAll('button').find((b) => b.text().includes('下载 JSON')).trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('已过期')
      expect(wrapper.text()).toContain('重新生成导出文件')
    })

    it('生成失败：展示错误信息并 emit error', async () => {
      apiFetchMock.mockResolvedValueOnce(statusPayload('none'))
      apiFetchMock.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: 'internal', detail: '服务内部错误' }),
      })
      const wrapper = mountPanel()
      await flushPromises()
      await wrapper.find('button').trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('服务内部错误')
      expect(wrapper.emitted('error')).toBeTruthy()
    })
  })
}
