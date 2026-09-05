// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] OrderInvoiceSection.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: vi.fn(),
    extractErrorMessage: (d) => d?.error || '',
  }))
  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))

  const Comp = (await import('./OrderInvoiceSection.vue')).default

  describe('OrderInvoiceSection', () => {
    it('已支付且无申请时显示申请开票', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: { id: '9', status: 'paid', invoices: [], total_yuan_cents: 55, payment_method: 'wechat', payment_ref: 'wechat:wx123' },
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-apply-btn"]').exists()).toBe(true)
    })

    it('总价为 0 时禁止申请开票并显示说明', async () => {
      const wrapper = mount(Comp, {
        props: { tenantId: '1', order: { id: '9', status: 'paid', invoices: [], total_yuan_cents: 0 } },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-apply-btn"]').exists()).toBe(false)
      const hint = wrapper.find('[data-testid="order-invoice-zero-amount-label"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('订单金额为 0 元，无法申请开票')
    })

    it('total_yuan 为 0.00 时同样禁止开票', async () => {
      const wrapper = mount(Comp, {
        props: { tenantId: '1', order: { id: '9', status: 'paid', invoices: [], total_yuan: '0.00' } },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-apply-btn"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="order-invoice-zero-amount-label"]').exists()).toBe(true)
    })

    it('admin_grant 零额订单同样禁止开票', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: { id: '9', status: 'paid', invoices: [], total_yuan_cents: 0, payment_method: 'admin_grant' },
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-apply-btn"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="order-invoice-zero-amount-label"]').text()).toContain('无法申请开票')
    })

    it('1 分订单仍显示申请开票', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: { id: '9', status: 'paid', invoices: [], total_yuan_cents: 1, payment_method: 'wechat', payment_ref: 'wechat:wx123' },
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-apply-btn"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-invoice-zero-amount-label"]').exists()).toBe(false)
    })

    it('admin_grant 正额已支付不显示申请开票（仅微信渠道可开票）', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: { id: '9', status: 'paid', invoices: [], total_yuan_cents: 55, payment_method: 'admin_grant' },
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-apply-btn"]').exists()).toBe(false)
      const hint = wrapper.find('[data-testid="order-invoice-non-wechat-label"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('仅微信支付订单可申请开票')
    })

    it('paypal 已支付也不可开票（后端仅放行 wechat）', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: { id: '9', status: 'paid', invoices: [], total_yuan_cents: 55, payment_method: 'paypal', payment_ref: 'paypal:pp123' },
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-apply-btn"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="order-invoice-non-wechat-label"]').exists()).toBe(true)
    })

    it('pending 申请显示审批中', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: { id: '9', status: 'paid', invoices: [], invoice_application: { status: 'pending' } },
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-pending-label"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-invoice-apply-btn"]').exists()).toBe(false)
    })

    it('列出挂在订单下的发票', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: {
            id: '9',
            status: 'paid',
            invoices: [{ id: 'a', kind: 'blue', purpose: 'original', status: 'issued', amount_yuan: '0.55' }],
          },
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-list"]').text()).toContain('蓝字发票')
      expect(wrapper.find('[data-testid="order-invoice-list"]').text()).toContain('0.55')
    })

    it('冲红待确认时显示72小时提醒', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: {
            id: '9',
            status: 'refunding',
            invoices: [{
              id: 'r',
              kind: 'red',
              purpose: 'reverse',
              status: 'reverse_pending',
              amount_yuan: '0.55',
              reverse_confirm_hours: 72,
              reverse_confirm_deadline: '2026-08-26T00:00:00Z',
              reverse_confirm_expired: false,
            }],
            invoice_reverse_confirm: {
              required: true,
              hours: 72,
              deadline: '2026-08-26T00:00:00Z',
              expired: false,
              message: '请在微信卡包于72小时内确认冲红，逾期冲红将失效',
            },
          },
        },
      })
      await flushPromises()
      const hint = wrapper.find('[data-testid="order-invoice-reverse-confirm-hint"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('72小时')
      expect(hint.text()).toContain('逾期冲红将失效')
    })

    it('冲红确认超时显示失效提示', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: {
            id: '9',
            status: 'refunded',
            invoices: [{
              id: 'r',
              kind: 'red',
              purpose: 'reverse',
              status: 'reverse_expired',
              amount_yuan: '0.55',
              reverse_confirm_hours: 72,
              reverse_confirm_deadline: '2026-08-04T00:00:00Z',
              reverse_confirm_expired: true,
            }],
            invoice_reverse_confirm: {
              required: true,
              hours: 72,
              deadline: '2026-08-04T00:00:00Z',
              expired: true,
              message: '冲红确认已超过72小时，冲红可能已失效，请联系平台处理',
            },
          },
        },
      })
      await flushPromises()
      const hint = wrapper.find('[data-testid="order-invoice-reverse-confirm-hint"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('已超过72小时')
    })

    it('选择专票后展示完整单位字段，未填全时禁止提交', async () => {
      const { apiFetch } = await import('../utils/apiUtils')
      const wrapper = mount(Comp, {
        props: { tenantId: '1', order: { id: '9', status: 'paid', invoices: [], payment_method: 'wechat', payment_ref: 'wechat:wx123' } },
      })
      await flushPromises()
      await wrapper.get('[data-testid="order-invoice-apply-btn"]').trigger('click')
      apiFetch.mockClear()
      await wrapper.get('[data-testid="order-invoice-type-select"]').setValue('special')
      await flushPromises()
      expect(wrapper.find('[data-testid="order-invoice-address"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-invoice-telephone"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-invoice-bank-name"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-invoice-bank-account"]').exists()).toBe(true)
      const confirm = wrapper.get('[data-testid="order-invoice-apply-confirm"]')
      expect(confirm.attributes('disabled')).toBeDefined()
    })

    it('专票提交 body 包含 invoice_type 与完整开票信息', async () => {
      const { apiFetch } = await import('../utils/apiUtils')
      apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
      const wrapper = mount(Comp, {
        props: { tenantId: '1', order: { id: '9', status: 'paid', invoices: [], payment_method: 'wechat', payment_ref: 'wechat:wx123' } },
      })
      await flushPromises()
      await wrapper.get('[data-testid="order-invoice-apply-btn"]').trigger('click')
      apiFetch.mockClear()
      await wrapper.get('[data-testid="order-invoice-type-select"]').setValue('special')
      await flushPromises()
      await wrapper.find('input[placeholder="姓名或公司全称"]').setValue('测试科技有限公司')
      await wrapper.find('input[placeholder="统一社会信用代码"]').setValue('91310000MA1FL4XH6A')
      await wrapper.get('[data-testid="order-invoice-address"]').setValue('上海市浦东新区测试路1号')
      await wrapper.get('[data-testid="order-invoice-telephone"]').setValue('021-12345678')
      await wrapper.get('[data-testid="order-invoice-bank-name"]').setValue('招商银行上海分行')
      await wrapper.get('[data-testid="order-invoice-bank-account"]').setValue('6225880212345678')
      await flushPromises()
      await wrapper.get('[data-testid="order-invoice-apply-confirm"]').trigger('click')
      await flushPromises()
      const [url, opts] = apiFetch.mock.calls.find(([u]) => String(u).includes('/invoice-applications/'))
      expect(url).toBe('/api/tenant/1/billing/orders/9/invoice-applications/')
      const body = JSON.parse(opts.body)
      expect(body.invoice_type).toBe('special')
      expect(body.type).toBe('ORGANIZATION')
      expect(body.taxpayer_id).toBe('91310000MA1FL4XH6A')
      expect(body.address).toBe('上海市浦东新区测试路1号')
      expect(body.telephone).toBe('021-12345678')
      expect(body.bank_name).toBe('招商银行上海分行')
      expect(body.bank_account).toBe('6225880212345678')
    })

    it('普票提交 invoice_type 默认 general，不携带专票字段', async () => {
      const { apiFetch } = await import('../utils/apiUtils')
      apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
      const wrapper = mount(Comp, {
        props: { tenantId: '1', order: { id: '9', status: 'paid', invoices: [], payment_method: 'wechat', payment_ref: 'wechat:wx123' } },
      })
      await flushPromises()
      await wrapper.get('[data-testid="order-invoice-apply-btn"]').trigger('click')
      apiFetch.mockClear()
      await wrapper.find('input[placeholder="姓名或公司全称"]').setValue('张三')
      await flushPromises()
      await wrapper.get('[data-testid="order-invoice-apply-confirm"]').trigger('click')
      await flushPromises()
      const [url, opts] = apiFetch.mock.calls.find(([u]) => String(u).includes('/invoice-applications/'))
      const body = JSON.parse(opts.body)
      expect(body.invoice_type).toBe('general')
      expect(body.address).toBeUndefined()
      expect(body.bank_account).toBeUndefined()
    })

    it('发票列表显示类型徽标与发票文件下载链接', async () => {
      const wrapper = mount(Comp, {
        props: {
          tenantId: '1',
          order: {
            id: '9',
            status: 'paid',
            invoices: [{
              id: 'a',
              kind: 'blue',
              purpose: 'original',
              status: 'issued',
              amount_yuan: '0.55',
              invoice_type: 'special',
              invoice_file_url: '/api/tenant/1/billing/orders/9/invoices/a/file/',
            }],
          },
        },
      })
      await flushPromises()
      const list = wrapper.find('[data-testid="order-invoice-list"]')
      expect(list.text()).toContain('专票')
      expect(wrapper.find('[data-testid="order-invoice-file-link"]').attributes('href'))
        .toBe('/api/tenant/1/billing/orders/9/invoices/a/file/')
    })
  })
}
