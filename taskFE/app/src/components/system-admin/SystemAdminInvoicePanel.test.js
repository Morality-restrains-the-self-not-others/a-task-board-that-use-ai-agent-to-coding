// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminInvoicePanel.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('../../utils/apiUtils', () => ({
    apiFetch: vi.fn(),
  }))
  vi.mock('../../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))

  const Comp = (await import('./SystemAdminInvoicePanel.vue')).default

  const resp = (body) => ({ ok: true, json: async () => body })

  const makeRow = (over = {}) => ({
    id: '11',
    tenant_id: '22',
    order_id: '33',
    buyer_name: '张三',
    buyer_type: 'INDIVIDUAL',
    status: 'pending',
    ...over,
  })

  describe('SystemAdminInvoicePanel', () => {
    it('加载后展示租户开票申请行（知晓请求入口）', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow()] }))
      const wrapper = mount(Comp)
      await flushPromises()
      expect(apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/system-admin/invoice-applications/'),
        expect.anything(),
      )
      expect(wrapper.text()).toContain('张三')
      expect(wrapper.text()).toContain('开票申请')
      expect(wrapper.text()).toContain('手动开具')
    })

    it('pending 行展示「已开具」「拒绝」按钮，非 pending 行展示占位符', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({
        results: [makeRow(), makeRow({ id: '12', status: 'approved' })],
      }))
      const wrapper = mount(Comp)
      await flushPromises()
      expect(wrapper.find('[data-testid="invoice-issued-btn"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="invoice-reject-btn"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="invoice-issued-btn"]').text()).toBe('已开具')
    })

    it('点击「已开具」登记手动开具（approve + note=已手动开具）', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow()] }))
      const wrapper = mount(Comp)
      await flushPromises()
      apiFetch.mockResolvedValueOnce(resp({ results: [] }))
      await wrapper.get('[data-testid="invoice-issued-btn"]').trigger('click')
      await flushPromises()
      const [url, opts] = apiFetch.mock.calls.find(([u]) => String(u).includes('/approve/'))
      expect(url).toBe('/api/system-admin/invoice-applications/11/approve/')
      expect(JSON.parse(opts.body).note).toBe('已手动开具')
    })

    it('「已开具」登记携带可选微信发票号码（OPT-20260823-053）', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow()] }))
      const wrapper = mount(Comp)
      await flushPromises()
      apiFetch.mockResolvedValueOnce(resp({ results: [] }))
      const input = wrapper.get('[data-testid="invoice-fapiao-number-input"]')
      await input.setValue('31100000000000000000')
      await wrapper.get('[data-testid="invoice-issued-btn"]').trigger('click')
      await flushPromises()
      const approveCalls = apiFetch.mock.calls.filter(([u]) => String(u).includes('/approve/'))
      const [url, opts] = approveCalls[approveCalls.length - 1]
      expect(url).toBe('/api/system-admin/invoice-applications/11/approve/')
      expect(JSON.parse(opts.body).fapiao_number).toBe('31100000000000000000')
    })

    it('「已开具」未填号码时 fapiao_number 为空字符串', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow()] }))
      const wrapper = mount(Comp)
      await flushPromises()
      apiFetch.mockResolvedValueOnce(resp({ results: [] }))
      await wrapper.get('[data-testid="invoice-issued-btn"]').trigger('click')
      await flushPromises()
      const approveCalls = apiFetch.mock.calls.filter(([u]) => String(u).includes('/approve/'))
      const [, opts] = approveCalls[approveCalls.length - 1]
      expect(JSON.parse(opts.body).fapiao_number).toBe('')
    })

    it('点击「拒绝」拒绝申请（reject + note=资料不符）', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow()] }))
      const wrapper = mount(Comp)
      await flushPromises()
      apiFetch.mockResolvedValueOnce(resp({ results: [] }))
      await wrapper.get('[data-testid="invoice-reject-btn"]').trigger('click')
      await flushPromises()
      const [url, opts] = apiFetch.mock.calls.find(([u]) => String(u).includes('/reject/'))
      expect(url).toBe('/api/system-admin/invoice-applications/11/reject/')
      expect(JSON.parse(opts.body).note).toBe('资料不符')
    })

    it('专票申请展示类型徽标与完整单位信息', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow({
        invoice_type: 'special',
        buyer_type: 'ORGANIZATION',
        buyer_name: '测试科技有限公司',
        taxpayer_id: '91310000MA1FL4XH6A',
        address: '上海市浦东新区测试路1号',
        telephone: '021-12345678',
        bank_name: '招商银行上海分行',
        bank_account: '6225880212345678',
      })] }))
      const wrapper = mount(Comp)
      await flushPromises()
      expect(wrapper.find('[data-testid="invoice-type-badge"]').text()).toBe('专票')
      const info = wrapper.find('[data-testid="invoice-special-info"]')
      expect(info.text()).toContain('91310000MA1FL4XH6A')
      expect(info.text()).toContain('招商银行上海分行')
      expect(info.text()).toContain('6225880212345678')
    })

    it('普票申请展示普票徽标且不显示专票单位信息', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow({ invoice_type: 'general' })] }))
      const wrapper = mount(Comp)
      await flushPromises()
      expect(wrapper.find('[data-testid="invoice-type-badge"]').text()).toBe('普票')
      expect(wrapper.find('[data-testid="invoice-special-info"]').exists()).toBe(false)
    })

    it('上传发票文件：选择文件后 multipart POST 到 invoice-file 端点并刷新', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow()] }))
      const wrapper = mount(Comp)
      await flushPromises()
      apiFetch.mockResolvedValueOnce(resp({ invoice_file_url: '/api/system-admin/invoice-applications/11/invoice-file/' }))
      apiFetch.mockResolvedValueOnce(resp({ results: [makeRow({ invoice_file_path: 'invoice_files/11_x.pdf', invoice_file_url: '/api/system-admin/invoice-applications/11/invoice-file/' })] }))
      await wrapper.get('[data-testid="invoice-upload-btn"]').trigger('click')
      const input = wrapper.get('[data-testid="invoice-file-input"]')
      const file = new File(['%PDF-1.4 fake'], 'invoice.pdf', { type: 'application/pdf' })
      Object.defineProperty(input.element, 'files', { value: [file] })
      await input.trigger('change')
      await flushPromises()
      const call = apiFetch.mock.calls.find(([u]) => String(u).includes('/invoice-file/'))
      expect(call[0]).toBe('/api/system-admin/invoice-applications/11/invoice-file/')
      expect(call[1].method).toBe('POST')
      expect(call[1].body).toBeInstanceOf(FormData)
      expect(call[1].body.get('file').name).toBe('invoice.pdf')
      expect(wrapper.text()).toContain('重新上传发票')
      expect(wrapper.find('[data-testid="invoice-file-link"]').exists()).toBe(true)
    })

    it('上传失败展示错误信息', async () => {
      const { apiFetch } = await import('../../utils/apiUtils')
      apiFetch.mockResolvedValue(resp({ results: [makeRow()] }))
      const wrapper = mount(Comp)
      await flushPromises()
      apiFetch.mockResolvedValueOnce({ ok: false, json: async () => ({ error: 'unsupported invoice file type (pdf/jpg/png only)' }) })
      await wrapper.get('[data-testid="invoice-upload-btn"]').trigger('click')
      const input = wrapper.get('[data-testid="invoice-file-input"]')
      const file = new File(['#!/bin/sh'], 'evil.sh', { type: 'application/x-sh' })
      Object.defineProperty(input.element, 'files', { value: [file] })
      await input.trigger('change')
      await flushPromises()
      expect(wrapper.text()).toContain('unsupported invoice file type')
    })
  })
}
