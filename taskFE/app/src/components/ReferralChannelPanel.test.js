// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ReferralChannelPanel.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  function jsonOk(body) {
    return Promise.resolve({ ok: true, status: 200, json: async () => body })
  }

  describe('ReferralChannelPanel', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      vi.stubGlobal('crypto', { randomUUID: () => 'ik-ch-1' })
    })

    it('states share codes work before qualification and commission is at order time', async () => {
      hoisted.apiFetchMock.mockImplementation(() => jsonOk({
        channels: [{ code: 'DefCode12ab', name: '默认', is_default: true, status: 'active' }],
      }))
      const { default: Panel } = await import('./ReferralChannelPanel.vue')
      const wrapper = mount(Panel, { props: { fallbackCode: 'DefCode12ab' } })
      await flushPromises()
      const intro = wrapper.get('[data-testid="referral-channel-intro"]')
      expect(intro.text()).toContain('下单时')
      expect(intro.text()).toContain('后续下单')
      expect(intro.text()).not.toContain('但被推荐人消费不分账')
    })

    it('creates a channel with Idempotency-Key', async () => {
      hoisted.apiFetchMock.mockImplementation((url, opts) => {
        if (opts && opts.method === 'POST') {
          return jsonOk({ code: 'NewCode12ab', name: '微信', is_default: false, status: 'active' })
        }
        return jsonOk({ channels: [{ code: 'DefCode12ab', name: '默认', is_default: true, status: 'active' }] })
      })
      const { default: Panel } = await import('./ReferralChannelPanel.vue')
      const wrapper = mount(Panel, { props: { fallbackCode: 'DefCode12ab' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-channel-name"]').setValue('微信')
      await wrapper.get('[data-testid="referral-channel-create"]').trigger('click')
      await flushPromises()
      const post = hoisted.apiFetchMock.mock.calls.find((c) => c[1] && c[1].method === 'POST')
      expect(post).toBeTruthy()
      expect(post[1].headers['Idempotency-Key']).toBe('ik-ch-1')
      expect(JSON.parse(post[1].body)).toEqual({ name: '微信' })
    })

    it('shows retry copy with data-traceId on gateway 502 HTML during create', async () => {
      hoisted.apiFetchMock.mockImplementation((url, opts) => {
        if (opts && opts.method === 'POST') {
          return Promise.resolve({
            ok: false,
            status: 502,
            traceId: 'tr-ch-502',
            headers: { get: () => null },
            _errorData: {
              _rawErrorText: '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>',
            },
            json: async () => { throw new Error('body is html') },
          })
        }
        return jsonOk({ channels: [{ code: 'DefCode12ab', name: '默认', is_default: true, status: 'active' }] })
      })
      const { default: Panel } = await import('./ReferralChannelPanel.vue')
      const wrapper = mount(Panel, { props: { fallbackCode: 'DefCode12ab' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-channel-name"]').setValue('微信')
      await wrapper.get('[data-testid="referral-channel-create"]').trigger('click')
      await flushPromises()
      const err = wrapper.find('p.text-red-500')
      expect(err.exists()).toBe(true)
      expect(err.text()).toContain('服务暂时不可用，请稍后重试')
      const attrs = err.attributes()
      expect(attrs['data-trace-id'] || attrs['data-traceid']).toBe('tr-ch-502')
    })

    it('deletes a channel after confirm and refreshes the list', async () => {
      vi.stubGlobal('confirm', vi.fn(() => true))
      const channels = [
        { code: 'DefCode12ab', name: '默认', is_default: true, status: 'active' },
        { code: 'WxCode34cd', name: '微信', is_default: false, status: 'active' },
      ]
      hoisted.apiFetchMock.mockImplementation((url, opts) => {
        if (opts && opts.method === 'DELETE') {
          return jsonOk({ status: 'ok' })
        }
        return jsonOk({ channels })
      })
      const { default: Panel } = await import('./ReferralChannelPanel.vue')
      const wrapper = mount(Panel, { props: { fallbackCode: 'DefCode12ab' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-channel-delete-WxCode34cd"]').trigger('click')
      await flushPromises()
      const del = hoisted.apiFetchMock.mock.calls.find((c) => c[1] && c[1].method === 'DELETE')
      expect(del).toBeTruthy()
      expect(del[0]).toBe('/api/referral/channels/code/WxCode34cd/')
      expect(del[1].headers['Idempotency-Key']).toBe('ik-ch-1')
      expect(vi.mocked(window.confirm)).toHaveBeenCalledTimes(1)
    })

    it('does not delete when confirm is dismissed', async () => {
      vi.stubGlobal('confirm', vi.fn(() => false))
      const channels = [
        { code: 'DefCode12ab', name: '默认', is_default: true, status: 'active' },
        { code: 'WxCode34cd', name: '微信', is_default: false, status: 'active' },
      ]
      hoisted.apiFetchMock.mockImplementation(() => jsonOk({ channels }))
      const { default: Panel } = await import('./ReferralChannelPanel.vue')
      const wrapper = mount(Panel, { props: { fallbackCode: 'DefCode12ab' } })
      await flushPromises()
      await wrapper.get('[data-testid="referral-channel-delete-WxCode34cd"]').trigger('click')
      await flushPromises()
      const del = hoisted.apiFetchMock.mock.calls.find((c) => c[1] && c[1].method === 'DELETE')
      expect(del).toBeFalsy()
    })

    it('default channel has no delete button but disabled channels do', async () => {
      const channels = [
        { code: 'DefCode12ab', name: '默认', is_default: true, status: 'active' },
        { code: 'OldCode56ef', name: '旧渠道', is_default: false, status: 'disabled' },
      ]
      hoisted.apiFetchMock.mockImplementation(() => jsonOk({ channels }))
      const { default: Panel } = await import('./ReferralChannelPanel.vue')
      const wrapper = mount(Panel, { props: { fallbackCode: 'DefCode12ab' } })
      await flushPromises()
      expect(wrapper.find('[data-testid="referral-channel-delete-DefCode12ab"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="referral-channel-delete-OldCode56ef"]').exists()).toBe(true)
    })
  })
}
