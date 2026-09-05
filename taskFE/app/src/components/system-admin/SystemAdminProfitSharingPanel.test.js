// @vitest-environment jsdom
/**
 * 待分账面板：渲染队列、空态、订单深链。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminProfitSharingPanel.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  const Page = (await import('./SystemAdminProfitSharingPanel.vue')).default

  function jsonOk(body) {
    return {
      ok: true,
      status: 200,
      headers: { get: () => 'application/json' },
      json: async () => body,
      clone: function () {
        return this
      },
    }
  }

  function jsonErr(body, extra = {}) {
    return {
      ok: false,
      status: extra.status || 400,
      headers: {
        get: (name) => {
          if (String(name).toLowerCase() === 'x-trace-id') return extra.traceId || null
          return extra.contentType || 'application/json'
        },
      },
      json: async () => body,
      clone: function () {
        return this
      },
    }
  }

  describe('SystemAdminProfitSharingPanel', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
    })

    it('渲染待分账行与订单深链', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonOk({
        items: [
          {
            id: '1',
            order_id: '877596007691485184',
            order_number: 'ORD877596007691485184',
            tenant_id: '877397588196749312',
            receiver_user_id: 'referrer-1',
            app_id: 'wxLOGIN',
            openid: 'oSTAFF-openid',
            total_yuan: '10.00',
            amount_yuan: '0.50',
            status: 'pending',
            settle_after: '2026-08-30T00:00:00Z',
            fail_reason: '',
          },
        ],
        total: 1,
        limit: 20,
        offset: 0,
      }))
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.text()).toContain('ORD877596007691485184')
      expect(wrapper.text()).toContain('待分账')
      expect(wrapper.text()).toContain('0.50')
      expect(wrapper.get('[data-testid="profit-sharing-app-id"]').text()).toBe('wxLOGIN')
      expect(wrapper.get('[data-testid="profit-sharing-openid"]').text()).toBe('oSTAFF-openid')
      const href = wrapper.get('[data-testid="profit-sharing-order-link"]').attributes('href')
      expect(href).toBe(
        '/system-admin/order-records/?tenant_id=877397588196749312&order_id=877596007691485184',
      )
      const called = String(hoisted.apiFetch.mock.calls[0][0])
      expect(called).toContain('/api/system-admin/profit-sharing/')
      expect(called).toContain('status=open')
    })

    it('失败原因 qualification_revoked 展示中文', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonOk({
        items: [
          {
            id: '2',
            order_id: '877596007691485184',
            order_number: 'ORD877596007691485184',
            tenant_id: '877397588196749312',
            receiver_user_id: 'referrer-1',
            total_yuan: '10.00',
            amount_yuan: '0.50',
            status: 'failed',
            settle_after: '2026-08-30T00:00:00Z',
            fail_reason: 'qualification_revoked',
          },
        ],
        total: 1,
        limit: 20,
        offset: 0,
      }))
      const wrapper = mount(Page)
      await flushPromises()
      const cell = wrapper.get('[data-testid="profit-sharing-fail-reason"]')
      expect(cell.text()).toBe('推荐资格已撤销')
      expect(cell.attributes('title')).toBe('推荐资格已撤销')
      expect(wrapper.text()).not.toContain('qualification_revoked')
      expect(cell.element.getAttribute('data-traceId')).toBeNull()
    })

    it('失败原因列携带 fail_trace_id 作为 data-traceId', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonOk({
        items: [{
          id: '11',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          receiver_user_id: 'referrer-1',
          total_yuan: '10.00',
          amount_yuan: '0.50',
          status: 'failed',
          settle_after: '2026-08-30T00:00:00Z',
          fail_reason: 'create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST","message":"appid与openid不匹配"}',
          fail_trace_id: 'ps-appid-openid-mismatch-01',
        }],
        total: 1,
        limit: 20,
        offset: 0,
      }))
      const wrapper = mount(Page)
      await flushPromises()
      const cell = wrapper.get('[data-testid="profit-sharing-fail-reason"]')
      expect(cell.element.getAttribute('data-traceId')).toBe('ps-appid-openid-mismatch-01')
    })

    it('长失败原因在单元格内完整换行展示且不 truncate', async () => {
      const failReason = 'create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST","message":"分账金额超出最大分账比例，最大可分账金额需等比例扣除退款与补差回退等逆向交易金额"}'
      hoisted.apiFetch.mockResolvedValue(jsonOk({
        items: [{
          id: '11',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          receiver_user_id: 'referrer-1',
          total_yuan: '10.00',
          amount_yuan: '0.50',
          status: 'failed',
          settle_after: '2026-08-30T00:00:00Z',
          fail_reason: failReason,
        }],
        total: 1,
        limit: 20,
        offset: 0,
      }))
      const wrapper = mount(Page)
      await flushPromises()
      const cell = wrapper.get('[data-testid="profit-sharing-fail-reason"]')
      expect(cell.classes()).not.toContain('truncate')
      expect(cell.classes().join(' ')).not.toMatch(/max-w-\[8rem\]/)
      expect(cell.classes()).toContain('whitespace-normal')
      expect(cell.classes()).toContain('break-all')
      expect(cell.text()).toBe(failReason)
      expect(cell.attributes('title')).toBe(failReason)
    })

    it('空列表展示文案', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonOk({ items: [], total: 0, limit: 20, offset: 0 }))
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.get('[data-testid="profit-sharing-empty"]').text()).toContain('暂无待分账订单')
    })

    it('展示微信订单号并允许待分账行发起分账', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonOk({
          items: [{
            id: '1',
            order_id: '877596007691485184',
            order_number: 'ORD877596007691485184',
            tenant_id: '877397588196749312',
            receiver_user_id: 'referrer-1',
            total_yuan: '10.00',
            amount_yuan: '0.50',
            status: 'pending',
            wechat_transaction_id: '4200003333',
            wechat_profit_sharing_id: '',
          }],
          total: 1,
          limit: 20,
          offset: 0,
        }))
        .mockResolvedValueOnce(jsonOk({ status: 'ok', state: 'shared' }))
        .mockResolvedValueOnce(jsonOk({ items: [], total: 0, limit: 20, offset: 0 }))
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.get('[data-testid="profit-sharing-wechat-txn"]').text()).toBe('4200003333')
      const btn = wrapper.get('[data-testid="profit-sharing-share"]')
      await btn.trigger('click')
      await flushPromises()
      expect(btn.element.disabled).toBe(false)
      expect(btn.attributes('aria-expanded')).toBe('true')
      const form = wrapper.get('[data-testid="profit-sharing-share-form"]')
      const table = wrapper.get('table.min-w-full')
      expect(form.element.compareDocumentPosition(table.element)
        & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
      expect(hoisted.apiFetch.mock.calls.filter((c) => String(c[0]).includes('/share/')).length).toBe(0)
      await wrapper.get('[data-testid="profit-sharing-share-reason"]').setValue('客服复核后补发起分账')
      await wrapper.get('[data-testid="profit-sharing-share-confirm"]').trigger('click')
      await flushPromises()
      const post = hoisted.apiFetch.mock.calls.find((c) => String(c[0]).includes('/share/'))
      expect(post[1].headers['Idempotency-Key']).toBeTruthy()
      expect(JSON.parse(post[1].body).reason).toBe('客服复核后补发起分账')
    })

    it('分账失败空账号 PARAM_ERROR 展示绑定微信文案', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonOk({
          items: [{
            id: '1',
            order_id: '877596007691485184',
            order_number: 'ORD877596007691485184',
            tenant_id: '877397588196749312',
            receiver_user_id: 'referrer-1',
            total_yuan: '10.00',
            amount_yuan: '0.50',
            status: 'pending',
          }],
          total: 1,
          limit: 20,
          offset: 0,
        }))
        .mockResolvedValueOnce(jsonErr(
          {
            error: 'create profit sharing failed: HTTP 400, {"code":"PARAM_ERROR","message":"输入源“/body/receivers/0/account”映射到值字段“分账接收方帐号”字符串规则校验失败，字符数 0，小于最小值 1"}',
            trace_id: 'admin-ps-empty-openid',
          },
          { status: 400, traceId: 'admin-ps-empty-openid' },
        ))
      const wrapper = mount(Page)
      await flushPromises()
      await wrapper.get('[data-testid="profit-sharing-share"]').trigger('click')
      await wrapper.get('[data-testid="profit-sharing-share-reason"]').setValue('客服复核后补发起分账')
      await wrapper.get('[data-testid="profit-sharing-share-confirm"]').trigger('click')
      await flushPromises()
      const err = wrapper.get('[data-testid="profit-sharing-error"]')
      expect(err.text()).toBe('推荐人未绑定微信收款账号')
      expect(err.text()).not.toContain('PARAM_ERROR')
      expect(err.element.getAttribute('data-traceId')).toBe('admin-ps-empty-openid')
      expect(wrapper.get('[data-testid="profit-sharing-fail-reason"]').element.getAttribute('data-traceId')).toBe('admin-ps-empty-openid')
    })

    it('展示商户单号且不单独列出商户分账单号列', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonOk({
        items: [{
          id: '1',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          receiver_user_id: 'referrer-1',
          total_yuan: '10.00',
          amount_yuan: '0.50',
          status: 'failed',
          out_trade_no: 'WX877596007691485184',
          out_profit_sharing_no: 'PS877596007691485185',
        }],
        total: 1,
        limit: 20,
        offset: 0,
      }))
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.text()).toContain('商户单号')
      expect(wrapper.findAll('th').map((th) => th.text())).not.toContain('商户分账单号')
      expect(wrapper.get('[data-testid="profit-sharing-out-trade-no"]').text()).toBe('WX877596007691485184')
      expect(wrapper.get('[data-testid="profit-sharing-wechat-order"]').text()).toBe('PS877596007691485185')
    })

    it('商户单号与分账单号均为空时展示破折号', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonOk({
        items: [{
          id: '1',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          receiver_user_id: 'referrer-1',
          total_yuan: '10.00',
          amount_yuan: '0.50',
          status: 'pending',
          out_trade_no: '',
        }],
        total: 1,
        limit: 20,
        offset: 0,
      }))
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.get('[data-testid="profit-sharing-out-trade-no"]').text()).toBe('—')
      expect(wrapper.get('[data-testid="profit-sharing-wechat-order"]').text()).toBe('—')
    })

    it('AppID 与 OpenID 为空时展示破折号', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonOk({
        items: [{
          id: '1',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          receiver_user_id: 'referrer-1',
          total_yuan: '10.00',
          amount_yuan: '0.50',
          status: 'pending',
          app_id: '',
          openid: '',
        }],
        total: 1,
        limit: 20,
        offset: 0,
      }))
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.get('[data-testid="profit-sharing-app-id"]').text()).toBe('—')
      expect(wrapper.get('[data-testid="profit-sharing-openid"]').text()).toBe('—')
    })
  })
}
