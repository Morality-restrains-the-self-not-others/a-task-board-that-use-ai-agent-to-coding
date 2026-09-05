// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ReferralWechatProfitSharingTab.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  const Tab = (await import('./ReferralWechatProfitSharingTab.vue')).default

  function jsonResp(body, extra = {}) {
    return {
      ok: extra.ok !== false,
      status: extra.status || 200,
      headers: {
        get: (name) => {
          if (String(name).toLowerCase() === 'x-trace-id') return extra.traceId || null
          return null
        },
      },
      json: async () => body,
      clone: function () {
        return this
      },
    }
  }

  describe('ReferralWechatProfitSharingTab', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
    })

    it('GET 带 referrer_user_id 与 status=all', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({ items: [], total: 0, receiver_registration_status: '' }))
      mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      const called = String(hoisted.apiFetch.mock.calls[0][0])
      expect(called).toContain('/api/system-admin/profit-sharing/')
      expect(called).toContain('referrer_user_id=ref-1')
      expect(called).toContain('status=all')
    })

    it('空列表展示暂无微信分账记录', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({ items: [], total: 0 }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-empty"]').text()).toContain('暂无微信分账记录')
    })

    it('有记录展示被推荐人、AppID、OpenID 与订单号', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        receiver_registration_status: 'registered',
        items: [{
          id: '11',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          referred_user_id: 'u-buyer',
          app_id: 'wxLOGIN',
          openid: 'oBUYER-openid',
          amount_yuan: '0.50',
          status: 'pending',
          settle_after: '2026-08-30T00:00:00Z',
          fail_reason: '',
        }],
        total: 1,
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      expect(wrapper.text()).toContain('u-buyer')
      expect(wrapper.text()).toContain('ORD877596007691485184')
      expect(wrapper.text()).toContain('0.50')
      expect(wrapper.text()).toContain('待分账')
      expect(wrapper.get('[data-testid="referral-ps-receiver-status"]').text()).toContain('已登记')
      expect(wrapper.get('[data-testid="referral-ps-app-id"]').text()).toBe('wxLOGIN')
      expect(wrapper.get('[data-testid="referral-ps-openid"]').text()).toBe('oBUYER-openid')
      const href = wrapper.get('[data-testid="referral-ps-order-link"]').attributes('href')
      expect(href).toContain('/system-admin/order-records/')
    })

    it('AppID 与 OpenID 为空时展示破折号', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          referred_user_id: 'u-buyer',
          app_id: '',
          openid: '',
          amount_yuan: '0.50',
          status: 'pending',
        }],
        total: 1,
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-app-id"]').text()).toBe('—')
      expect(wrapper.get('[data-testid="referral-ps-openid"]').text()).toBe('—')
    })

    it('失败原因 qualification_revoked 展示中文', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          referred_user_id: 'u-buyer',
          amount_yuan: '0.50',
          status: 'failed',
          settle_after: '2026-08-30T00:00:00Z',
          fail_reason: 'qualification_revoked',
        }],
        total: 1,
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      const cell = wrapper.get('[data-testid="referral-ps-fail-reason"]')
      expect(cell.text()).toBe('推荐资格已撤销')
      expect(cell.attributes('title')).toBe('推荐资格已撤销')
      expect(wrapper.text()).not.toContain('qualification_revoked')
      expect(cell.element.getAttribute('data-traceId')).toBeNull()
    })

    it('失败原因列携带 fail_trace_id 作为 data-traceId', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          referred_user_id: 'u-buyer',
          amount_yuan: '0.50',
          status: 'failed',
          settle_after: '2026-08-30T00:00:00Z',
          fail_reason: 'create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST","message":"appid与openid不匹配"}',
          fail_trace_id: 'ps-appid-openid-mismatch-01',
        }],
        total: 1,
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      const cell = wrapper.get('[data-testid="referral-ps-fail-reason"]')
      expect(cell.element.getAttribute('data-traceId')).toBe('ps-appid-openid-mismatch-01')
      expect(cell.attributes('title')).toContain('appid与openid不匹配')
    })

    it('长失败原因在单元格内完整换行展示且不 truncate', async () => {
      const failReason = 'create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST","message":"分账金额超出最大分账比例，最大可分账金额需等比例扣除退款与补差回退等逆向交易金额"}'
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          referred_user_id: 'u-buyer',
          amount_yuan: '0.50',
          status: 'failed',
          settle_after: '2026-08-30T00:00:00Z',
          fail_reason: failReason,
        }],
        total: 1,
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      const cell = wrapper.get('[data-testid="referral-ps-fail-reason"]')
      expect(cell.classes()).not.toContain('truncate')
      expect(cell.classes().join(' ')).not.toMatch(/max-w-\[8rem\]/)
      expect(cell.classes()).toContain('whitespace-normal')
      expect(cell.classes()).toContain('break-all')
      expect(cell.text()).toBe(failReason)
      expect(cell.attributes('title')).toBe(failReason)
    })

    it('失败原因无 fail_trace_id 时不挂 data-traceId', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '1',
          order_number: 'ORD1',
          tenant_id: '2',
          referred_user_id: 'u1',
          amount_yuan: '0.50',
          status: 'failed',
          settle_after: '',
          fail_reason: 'create profit sharing failed: HTTP 400',
          fail_trace_id: '',
        }],
        total: 1,
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      const cell = wrapper.get('[data-testid="referral-ps-fail-reason"]')
      expect(cell.element.getAttribute('data-traceId')).toBeNull()
    })

    it('首屏直接展示后端回填的尚未提交微信状态（无需先同步）', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '877596007691485184',
          order_number: 'ORD877596007691485184',
          tenant_id: '877397588196749312',
          referred_user_id: 'u-buyer',
          amount_yuan: '0.50',
          status: 'pending',
          settle_after: '2026-08-30T00:00:00Z',
          fail_reason: '',
          wechat_state: '尚未提交微信',
        }],
        total: 1,
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      const cell = wrapper.get('[data-testid="referral-ps-wechat-state"]')
      expect(cell.text()).toBe('未向微信发起分账')
      expect(cell.attributes('title')).toContain('分账单')
      expect(cell.attributes('title')).toContain('绑定')
      expect(cell.classes()).not.toContain('text-red-600')
      expect(wrapper.get('[data-testid="referral-ps-wechat-state-col"]').text()).toBe('微信分账单状态')
      // 首屏未点击同步，不应出现同步请求
      expect(hoisted.apiFetch.mock.calls.every(([, opts]) => !opts || opts.method !== 'POST')).toBe(true)
    })

    it('同步成功更新微信状态', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
            settle_after: '',
            fail_reason: '',
          }],
        }))
        .mockResolvedValueOnce(jsonResp({
          items: [{ id: '11', wechat_state: 'FINISHED', wechat_error: '' }],
        }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-sync"]').trigger('click')
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-wechat-state"]').text()).toBe('微信已分账完成')
      expect(wrapper.text()).not.toContain('FINISHED')
      const post = hoisted.apiFetch.mock.calls[1]
      expect(String(post[0])).toContain('/api/system-admin/profit-sharing/refresh-wechat/')
      expect(post[1].method).toBe('POST')
      const body = JSON.parse(post[1].body)
      expect(body.referrer_user_id).toBe('ref-1')
      expect(body.ids).toEqual(['11'])
    })

    it('同步后未提交微信展示尚未提交微信而非404文案', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
            settle_after: '',
            fail_reason: '',
          }],
        }))
        .mockResolvedValueOnce(jsonResp({
          items: [{ id: '11', wechat_state: '尚未提交微信', wechat_error: '' }],
        }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-sync"]').trigger('click')
      await flushPromises()
      const cell = wrapper.get('[data-testid="referral-ps-wechat-state"]')
      expect(cell.text()).toBe('未向微信发起分账')
      expect(cell.attributes('title')).toContain('不是用户是否绑定微信')
      expect(cell.classes()).not.toContain('text-red-600')
      expect(wrapper.text()).not.toContain('尚未提交微信')
      expect(wrapper.text()).not.toContain('微信侧未找到分账单')
    })

    it('同步失败展示 data-traceId', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
          }],
        }))
        .mockResolvedValueOnce(jsonResp(
          { error: '同步失败', trace_id: 'web-ps-sync-fail-01' },
          { ok: false, status: 502, traceId: 'web-ps-sync-fail-01' },
        ))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-sync"]').trigger('click')
      await flushPromises()
      const err = wrapper.get('[data-testid="referral-ps-error"]')
      expect(err.element.getAttribute('data-traceId')).toBe('web-ps-sync-fail-01')
    })

    it('微信状态列 wechat_error 非空时挂 wechat_error_trace_id（OPT-20260826-008）', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
            settle_after: '',
            fail_reason: '',
          }],
        }))
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            wechat_state: '',
            wechat_error: '查询微信分账状态失败，请稍后重试',
            wechat_error_trace_id: 'web-ps-wechat-state-01',
          }],
        }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-sync"]').trigger('click')
      await flushPromises()
      const cell = wrapper.get('[data-testid="referral-ps-wechat-state"]')
      expect(cell.text()).toContain('查询微信分账状态失败')
      expect(cell.element.getAttribute('data-traceId')).toBe('web-ps-wechat-state-01')
      expect(cell.classes()).toContain('text-red-600')
    })

    it('微信状态列无 wechat_error 时不挂 data-traceId（OPT-20260826-008）', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
            settle_after: '',
            fail_reason: '',
          }],
        }))
        .mockResolvedValueOnce(jsonResp({
          items: [{ id: '11', wechat_state: 'PROCESSING', wechat_error: '' }],
        }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-sync"]').trigger('click')
      await flushPromises()
      const cell = wrapper.get('[data-testid="referral-ps-wechat-state"]')
      expect(cell.element.getAttribute('data-traceId')).toBeNull()
      expect(cell.classes()).not.toContain('text-red-600')
    })

    it('同步进行中连点第二次不发请求', async () => {
      let resolvePost
      hoisted.apiFetch.mockImplementation((url) => {
        if (String(url).includes('refresh-wechat')) {
          return new Promise((resolve) => {
            resolvePost = resolve
          })
        }
        return Promise.resolve(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
          }],
        }))
      })
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      const btn = wrapper.get('[data-testid="referral-ps-sync"]')
      const p1 = btn.trigger('click')
      await Promise.resolve()
      const p2 = btn.trigger('click')
      await Promise.resolve()
      const postCalls = hoisted.apiFetch.mock.calls.filter((c) => String(c[0]).includes('refresh-wechat'))
      expect(postCalls.length).toBe(1)
      resolvePost(jsonResp({ items: [{ id: '11', wechat_state: 'FINISHED' }] }))
      await p1
      await p2
      await flushPromises()
    })

    it('展示微信订单号与微信分账单号', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '1',
          order_number: 'ORD1',
          tenant_id: '2',
          referred_user_id: 'u1',
          amount_yuan: '1.00',
          status: 'pending',
          wechat_transaction_id: '4200001111',
          wechat_profit_sharing_id: '3000002222',
        }],
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-wechat-txn"]').text()).toBe('4200001111')
      expect(wrapper.get('[data-testid="referral-ps-wechat-order"]').text()).toBe('3000002222')
    })

    it('展示商户单号且不单独列出商户分账单号列', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '1',
          order_number: 'ORD1',
          tenant_id: '2',
          referred_user_id: 'u1',
          amount_yuan: '1.00',
          status: 'failed',
          out_trade_no: 'WX880041686850371585',
          out_profit_sharing_no: 'PS880041686850371586',
        }],
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      expect(wrapper.text()).toContain('商户单号')
      expect(wrapper.findAll('th').map((th) => th.text())).not.toContain('商户分账单号')
      expect(wrapper.get('[data-testid="referral-ps-out-trade-no"]').text()).toBe('WX880041686850371585')
      expect(wrapper.get('[data-testid="referral-ps-wechat-order"]').text()).toBe('PS880041686850371586')
    })

    it('商户单号与分账单号均为空时展示破折号', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '1',
          order_number: 'ORD1',
          tenant_id: '2',
          referred_user_id: 'u1',
          amount_yuan: '1.00',
          status: 'pending',
          out_trade_no: '',
          out_profit_sharing_no: '',
        }],
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-out-trade-no"]').text()).toBe('—')
      expect(wrapper.get('[data-testid="referral-ps-wechat-order"]').text()).toBe('—')
    })

    it('点分账后按钮保持可点、不发 POST，缘由表单出现在表格上方', async () => {
      const scrollSpy = vi.fn()
      HTMLElement.prototype.scrollIntoView = scrollSpy
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '1',
          order_number: 'ORD1',
          tenant_id: '2',
          referred_user_id: 'u1',
          amount_yuan: '1.00',
          status: 'pending',
        }],
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      const btn = wrapper.get('[data-testid="referral-ps-share"]')
      await btn.trigger('click')
      await flushPromises()
      expect(btn.element.disabled).toBe(false)
      expect(btn.attributes('aria-expanded')).toBe('true')
      const form = wrapper.get('[data-testid="referral-ps-share-form"]')
      const table = wrapper.get('table.min-w-full')
      expect(form.element.compareDocumentPosition(table.element)
        & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
      expect(scrollSpy).toHaveBeenCalled()
      const sharePosts = hoisted.apiFetch.mock.calls.filter((c) => String(c[0]).includes('/share/'))
      expect(sharePosts.length).toBe(0)
    })

    it('待分账行可打开缘由表单并用同一 Idempotency-Key 提交', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
          }],
        }))
        .mockResolvedValueOnce(jsonResp({ status: 'ok', state: 'shared' }))
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'processing',
          }],
        }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-share"]').trigger('click')
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-share-form"]').exists()).toBe(true)
      await wrapper.get('[data-testid="referral-ps-share-reason"]').setValue('客服复核后补发起分账')
      await wrapper.get('[data-testid="referral-ps-share-confirm"]').trigger('click')
      await flushPromises()
      const post = hoisted.apiFetch.mock.calls.find((c) => String(c[0]).includes('/share/'))
      expect(post).toBeTruthy()
      expect(post[1].method).toBe('POST')
      expect(post[1].headers['Idempotency-Key']).toBeTruthy()
      expect(JSON.parse(post[1].body).reason).toBe('客服复核后补发起分账')
    })

    it('缘由不足 8 字不发请求', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '1',
          order_number: 'ORD1',
          tenant_id: '2',
          referred_user_id: 'u1',
          amount_yuan: '1.00',
          status: 'failed',
        }],
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-share"]').trigger('click')
      await wrapper.get('[data-testid="referral-ps-share-reason"]').setValue('太短')
      await wrapper.get('[data-testid="referral-ps-share-confirm"]').trigger('click')
      await flushPromises()
      const posts = hoisted.apiFetch.mock.calls.filter((c) => String(c[0]).includes('/share/'))
      expect(posts.length).toBe(0)
      expect(wrapper.get('[data-testid="referral-ps-error"]').text()).toContain('8')
    })

    it('已完成行不展示分账按钮', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [{
          id: '11',
          order_id: '1',
          order_number: 'ORD1',
          tenant_id: '2',
          referred_user_id: 'u1',
          amount_yuan: '1.00',
          status: 'finished',
        }],
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      expect(wrapper.find('[data-testid="referral-ps-share"]').exists()).toBe(false)
    })

    it('未绑定接收方时展示绑定微信提示', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResp({
        items: [],
        receiver_registration_status: 'pending_openid',
      }))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-pending-openid"]').text()).toContain('尚未绑定微信')
    })

    it('分账失败 PARAM_ERROR 空账号展示绑定微信文案而非 HTTP dump', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
          }],
        }))
        .mockResolvedValueOnce(jsonResp(
          {
            error: 'create profit sharing failed: HTTP 400, {"code":"PARAM_ERROR","detail":{"location":"body","value":0},"message":"输入源“/body/receivers/0/account”映射到值字段“分账接收方帐号”字符串规则校验失败，字符数 0，小于最小值 1"}',
            trace_id: 'web-ps-share-empty-openid',
          },
          { ok: false, status: 400, traceId: 'web-ps-share-empty-openid' },
        ))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-share"]').trigger('click')
      await wrapper.get('[data-testid="referral-ps-share-reason"]').setValue('客服复核后补发起分账')
      await wrapper.get('[data-testid="referral-ps-share-confirm"]').trigger('click')
      await flushPromises()
      const err = wrapper.get('[data-testid="referral-ps-error"]')
      expect(err.text()).toBe('推荐人未绑定微信收款账号')
      expect(err.text()).not.toContain('PARAM_ERROR')
      expect(err.element.getAttribute('data-traceId')).toBe('web-ps-share-empty-openid')
    })

    it('分账失败展示 data-traceId', async () => {
      hoisted.apiFetch
        .mockResolvedValueOnce(jsonResp({
          items: [{
            id: '11',
            order_id: '1',
            order_number: 'ORD1',
            tenant_id: '2',
            referred_user_id: 'u1',
            amount_yuan: '1.00',
            status: 'pending',
          }],
        }))
        .mockResolvedValueOnce(jsonResp(
          { error: '分账失败', trace_id: 'web-ps-share-fail-01' },
          { ok: false, status: 502, traceId: 'web-ps-share-fail-01' },
        ))
      const wrapper = mount(Tab, { props: { userId: 'ref-1' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-ps-share"]').trigger('click')
      await wrapper.get('[data-testid="referral-ps-share-reason"]').setValue('客服复核后补发起分账')
      await wrapper.get('[data-testid="referral-ps-share-confirm"]').trigger('click')
      await flushPromises()
      const err = wrapper.get('[data-testid="referral-ps-error"]')
      expect(err.element.getAttribute('data-traceId')).toBe('web-ps-share-fail-01')
      expect(wrapper.get('[data-testid="referral-ps-fail-reason"]').element.getAttribute('data-traceId')).toBe('web-ps-share-fail-01')
    })
  })
}
