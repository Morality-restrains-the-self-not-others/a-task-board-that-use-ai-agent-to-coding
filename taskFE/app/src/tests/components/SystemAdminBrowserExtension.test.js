// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminBrowserExtension.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { default: SystemAdminBrowserExtension } = await import('../../views/SystemAdminBrowserExtension.vue')

  const OIDC_API = '/api/system-admin/oidc-extension/'
  const ID1 = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
  const ID2 = 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'

  const okJSON = (data, extra = {}) => ({ ok: true, json: async () => data, _errorData: null, ...extra })
  const errJSON = (data, status = 400, traceId = '') => ({
    ok: false,
    status,
    json: async () => data,
    _errorData: data,
    ...(traceId ? { traceId } : {}),
  })

  beforeEach(() => {
    apiFetch.mockReset()
  })

  describe('SystemAdminBrowserExtension 页面加载', () => {
    it('加载 GET 现状并渲染插件 ID chips', async () => {
      apiFetch.mockResolvedValueOnce(okJSON({
        client_id: 'chrome-extension',
        name: 'chrome-extension',
        managed_by: 'admin',
        extension_ids: [ID1],
        raw_redirect_uris: [`chrome-extension://${ID1}/oauth-callback.html`],
      }))

      const wrapper = mount(SystemAdminBrowserExtension)
      await flushPromises()

      expect(apiFetch).toHaveBeenCalledWith(OIDC_API, expect.objectContaining({ method: 'GET' }))
      expect(wrapper.text()).toContain(ID1)
      expect(wrapper.text()).toContain('管理员')
    })

    it('加载失败展示错误与 data-traceId', async () => {
      apiFetch.mockResolvedValueOnce(errJSON({ error: 'oidc client not found' }, 404, 'trace-load-fail'))

      const wrapper = mount(SystemAdminBrowserExtension)
      await flushPromises()

      expect(wrapper.text()).toContain('oidc client not found')
      const errEl = wrapper.find('[data-traceId="trace-load-fail"]')
      expect(errEl.exists()).toBe(true)
    })
  })

  describe('SystemAdminBrowserExtension 编辑保存', () => {
    it('保存触发 PUT 到尾斜杠契约 URL，成功后刷新 chips', async () => {
      apiFetch
        .mockResolvedValueOnce(okJSON({ client_id: 'chrome-extension', managed_by: 'admin', extension_ids: [ID1], raw_redirect_uris: [] }))
        .mockResolvedValueOnce(okJSON({ client_id: 'chrome-extension', managed_by: 'admin', extension_ids: [ID1, ID2], raw_redirect_uris: [] }))

      const wrapper = mount(SystemAdminBrowserExtension)
      await flushPromises()

      await wrapper.find('textarea').setValue(`${ID1}\n${ID2}`)
      await wrapper.find('button').trigger('click')
      await flushPromises()

      const putCall = apiFetch.mock.calls.find((c) => c[1] && c[1].method === 'PUT')
      expect(putCall).toBeTruthy()
      expect(putCall[0]).toBe(OIDC_API) // 尾斜杠契约
      expect(JSON.parse(putCall[1].body)).toEqual({ extension_ids: [ID1, ID2] })
      expect(wrapper.text()).toContain('已保存')
      expect(wrapper.findAll('.font-mono').some((el) => el.text().includes(ID2))).toBe(true)
    })

    it('非法 ID 前端拦截，不发起 PUT', async () => {
      apiFetch.mockResolvedValueOnce(okJSON({ client_id: 'chrome-extension', managed_by: 'admin', extension_ids: [], raw_redirect_uris: [] }))

      const wrapper = mount(SystemAdminBrowserExtension)
      await flushPromises()

      await wrapper.find('textarea').setValue('ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ')
      await wrapper.find('button').trigger('click')
      await flushPromises()

      const putCall = apiFetch.mock.calls.find((c) => c[1] && c[1].method === 'PUT')
      expect(putCall).toBeUndefined()
      expect(wrapper.text()).toContain('非法插件 ID')
    })

    it('PUT 失败展示后端错误与 data-traceId', async () => {
      apiFetch
        .mockResolvedValueOnce(okJSON({ client_id: 'chrome-extension', managed_by: 'admin', extension_ids: [ID1], raw_redirect_uris: [] }))
        .mockResolvedValueOnce(errJSON({ error: 'extension_ids too many', trace_id: 'trace-put-fail' }, 400))

      const wrapper = mount(SystemAdminBrowserExtension)
      await flushPromises()

      await wrapper.find('textarea').setValue(ID1)
      await wrapper.find('button').trigger('click')
      await flushPromises()

      expect(wrapper.text()).toContain('extension_ids too many')
      const errEl = wrapper.find('[data-traceId="trace-put-fail"]')
      expect(errEl.exists()).toBe(true)
    })
  })
}
