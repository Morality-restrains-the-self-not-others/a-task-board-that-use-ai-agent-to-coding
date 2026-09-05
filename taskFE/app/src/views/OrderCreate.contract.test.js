// @vitest-environment jsdom
// 创建订单成功后跳转独立详情页；失败不跳转且错误带 data-traceId。
if (!process.env.VITEST) {
  console.log('[skip] OrderCreate.contract.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    useRoute: () => ({ params: { tenant: '873472655125147648' }, query: {}, fullPath: '/', path: '/' }),
    push: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    extractErrorMessage: (_d, _r) => 'mock-err',
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => '' }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.useRoute(),
    useRouter: () => ({ push: mocks.push }),
  }))

  const { default: OrderCreate } = await import('./OrderCreate.vue')

  const ORDER = {
    id: '1',
    order_number: 'ORD-TEST-001',
    status: 'pending',
    total_yuan: '1.00',
    items: [{ id: 'i1', resource_type: 'task_post', quantity: 1, unit_price_yuan: '1.00', subtotal_yuan: '1.00' }],
  }

  function mockApi(overrides = {}) {
    mocks.apiFetch.mockImplementation((url, options = {}) => {
      const method = String(options.method || 'GET').toUpperCase()
      const key = `${method} ${url}`
      const hit = overrides[key]
      const body = hit !== undefined ? hit : null
      return Promise.resolve({
        ok: body !== null && !(body && body.__fail),
        status: body && body.__fail ? 400 : (body !== null ? 200 : 404),
        json: () => Promise.resolve(body && body.__fail ? body : (body ?? {})),
        traceId: body && body.trace_id ? body.trace_id : '',
      })
    })
  }

  describe('OrderCreate 创建后跳转详情页', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.push.mockReset()
      mockApi({
        'GET /api/tenant/873472655125147648/billing/order-pricing/': { pricing: { task_post: { price_yuan: 1, required_tier: 'normal' } } },
        'GET /api/tenant/873472655125147648/billing/membership/': { membership: { tier: 'normal', cumulative_consumption_yuan: 0 } },
        'POST /api/tenant/873472655125147648/billing/orders/': ORDER,
      })
    })

    it('创建订单成功后 router.push 到 billing_order_detail', async () => {
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await wrapper.find('input[type="number"]').setValue(1)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await flushPromises()
      expect(mocks.push).toHaveBeenCalledWith({
        name: 'billing_order_detail',
        params: { tenant: '873472655125147648', orderId: '1' },
      })
      expect(wrapper.text()).not.toContain('确认支付')
      wrapper.unmount()
    })

    it('购买 GitLab 资源时 POST body 必须带 region', async () => {
      mockApi({
        'GET /api/tenant/873472655125147648/billing/order-pricing/': {
          pricing: {
            task_post: { price_yuan: 1, required_tier: 'normal' },
            gitlab_disk: { price_yuan: 1, required_tier: 'vip1', min_quantity: 10 },
          },
        },
        'GET /api/tenant/873472655125147648/billing/membership/': { membership: { tier: 'vip1', cumulative_consumption_yuan: 99 } },
        'GET /api/billing/gitlab-regions/tenant_id/873472655125147648/': { regions: [{ slug: 'tencent-sh-1', name: '腾讯上海一区' }] },
        'POST /api/tenant/873472655125147648/billing/orders/': ORDER,
      })
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      const region = wrapper.find('[data-testid="order-gitlab-region"]')
      expect(region.exists()).toBe(true)
      await region.setValue('tencent-sh-1')
      await wrapper.find('[data-testid="order-gitlab-disk-gb"]').setValue(10)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post[1].body).toContain('"region":"tencent-sh-1"')
      wrapper.unmount()
    })

    function mockVipGitlabPricing() {
      mockApi({
        'GET /api/tenant/873472655125147648/billing/order-pricing/': {
          pricing: {
            task_post: { price_yuan: 1, required_tier: 'normal' },
            gitlab_disk: { price_yuan: 4, required_tier: 'vip1', min_quantity: 10 },
            gitlab_traffic: { price_yuan: 1, required_tier: 'vip1' },
          },
        },
        'GET /api/tenant/873472655125147648/billing/membership/': { membership: { tier: 'vip1', cumulative_consumption_yuan: 99 } },
        'GET /api/billing/gitlab-regions/tenant_id/873472655125147648/': {
          regions: [
            { slug: 'tencent-sh-1', name: '腾讯上海一区', is_active: true },
            { slug: 'tencent-shanghai-5', name: '腾讯上海五区', is_active: true },
          ],
        },
        'POST /api/tenant/873472655125147648/billing/orders/': ORDER,
      })
    }

    it('GitLab 流量位于磁盘卡片内且只有一个区域下拉', async () => {
      mockVipGitlabPricing()
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      const card = wrapper.find('[data-testid="order-gitlab-resources"]')
      expect(card.exists()).toBe(true)
      expect(card.text()).toContain('GitLab 磁盘')
      expect(card.text()).toContain('GitLab 流量')
      expect(wrapper.findAll('[data-testid="order-gitlab-region"]')).toHaveLength(1)
      expect(wrapper.find('[data-testid="order-gitlab-traffic-region"]').exists()).toBe(false)
      expect(card.find('[data-testid="order-gitlab-region"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="order-gitlab-disk-min-hint"]').text()).toContain('起购 10 GB')
      wrapper.unmount()
    })

    it('磁盘数量不足起购时不 POST', async () => {
      mockVipGitlabPricing()
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await wrapper.find('[data-testid="order-gitlab-region"]').setValue('tencent-sh-1')
      await wrapper.find('[data-testid="order-gitlab-disk-gb"]').setValue(5)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await flushPromises()
      const posts = mocks.apiFetch.mock.calls.filter((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(posts).toHaveLength(0)
      expect(wrapper.text()).toContain('GitLab 磁盘起购 10 GB')
      wrapper.unmount()
    })

    it('同时购买磁盘与流量时两行共用同一 region', async () => {
      mockVipGitlabPricing()
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await wrapper.find('[data-testid="order-gitlab-region"]').setValue('tencent-sh-1')
      await wrapper.find('[data-testid="order-gitlab-disk-gb"]').setValue(10)
      await wrapper.find('[data-testid="order-gitlab-traffic-gb"]').setValue(2)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      const body = JSON.parse(post[1].body)
      const disk = body.items.find((i) => i.resource_type === 'gitlab_disk')
      const traffic = body.items.find((i) => i.resource_type === 'gitlab_traffic')
      expect(disk.region).toBe('tencent-sh-1')
      expect(disk.quantity).toBe(10)
      expect(traffic.region).toBe('tencent-sh-1')
      expect(traffic.quantity).toBe(2)
      wrapper.unmount()
    })

    it('阿里云区域出现在人工开通 optgroup，选中后 POST hangzhou 并显示提示', async () => {
      mockApi({
        'GET /api/tenant/873472655125147648/billing/order-pricing/': {
          pricing: {
            task_post: { price_yuan: 1, required_tier: 'normal' },
            gitlab_disk: { price_yuan: 4, required_tier: 'vip1', min_quantity: 10 },
          },
        },
        'GET /api/tenant/873472655125147648/billing/membership/': { membership: { tier: 'vip1', cumulative_consumption_yuan: 99 } },
        'GET /api/billing/gitlab-regions/tenant_id/873472655125147648/': {
          regions: [
            { slug: 'tencent-sh-1', name: '腾讯上海一区', is_active: true, cloud_provider: 'tencent', infra_status: 'ready' },
            { slug: 'aliyun-cn-hangzhou', name: '阿里云杭州（华东1）', is_active: true, cloud_provider: 'aliyun', infra_status: 'pending_node' },
          ],
        },
        'POST /api/tenant/873472655125147648/billing/orders/': ORDER,
      })
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      const groups = wrapper.findAll('optgroup')
      expect(groups.some((g) => g.attributes('label') === '阿里云（人工开通节点）')).toBe(true)
      expect(wrapper.find('[data-testid="order-gitlab-pending-node-hint"]').exists()).toBe(false)
      await wrapper.find('[data-testid="order-gitlab-region"]').setValue('aliyun-cn-hangzhou')
      await flushPromises()
      expect(wrapper.find('[data-testid="order-gitlab-pending-node-hint"]').exists()).toBe(true)
      await wrapper.find('[data-testid="order-gitlab-disk-gb"]').setValue(10)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post[1].body).toContain('"region":"aliyun-cn-hangzhou"')
      wrapper.unmount()
    })

    it('仅购买流量时仍使用共用区域下拉', async () => {
      mockVipGitlabPricing()
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await wrapper.find('[data-testid="order-gitlab-region"]').setValue('tencent-shanghai-5')
      await wrapper.find('[data-testid="order-gitlab-traffic-gb"]').setValue(3)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      const body = JSON.parse(post[1].body)
      expect(body.items).toEqual([
        { resource_type: 'gitlab_traffic', quantity: 3, region: 'tencent-shanghai-5' },
      ])
      wrapper.unmount()
    })

    it('未选区域时填写流量不 POST 并提示选择 GitLab 区域', async () => {
      mockVipGitlabPricing()
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await wrapper.find('[data-testid="order-gitlab-traffic-gb"]').setValue(1)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await flushPromises()
      const posts = mocks.apiFetch.mock.calls.filter((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(posts).toHaveLength(0)
      expect(wrapper.text()).toContain('请先选择 GitLab 区域')
      wrapper.unmount()
    })

    it('选 GitLab 磁盘时显示留言框，POST 含 buyer_note', async () => {
      mockApi({
        'GET /api/tenant/873472655125147648/billing/order-pricing/': {
          pricing: {
            task_post: { price_yuan: 1, required_tier: 'normal' },
            gitlab_disk: { price_yuan: 1, required_tier: 'vip1', min_quantity: 10 },
          },
        },
        'GET /api/tenant/873472655125147648/billing/membership/': { membership: { tier: 'vip1', cumulative_consumption_yuan: 99 } },
        'GET /api/billing/gitlab-regions/tenant_id/873472655125147648/': { regions: [{ slug: 'tencent-sh-1', name: '腾讯上海一区' }] },
        'POST /api/tenant/873472655125147648/billing/orders/': ORDER,
      })
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-buyer-note"]').exists()).toBe(false)
      await wrapper.find('[data-testid="order-gitlab-region"]').setValue('tencent-sh-1')
      await wrapper.find('[data-testid="order-gitlab-disk-gb"]').setValue(10)
      await flushPromises()
      const note = wrapper.find('[data-testid="order-buyer-note"]')
      expect(note.exists()).toBe(true)
      await note.setValue('请开通 team-foo')
      await wrapper.findAll('button').find((b) => b.text().includes('创建订单')).trigger('click')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post[1].body).toContain('"buyer_note":"请开通 team-foo"')
      wrapper.unmount()
    })

    it('仅任务帖时不显示施工留言框', async () => {
      mockApi({
        'GET /api/tenant/873472655125147648/billing/order-pricing/': { pricing: { task_post: { price_yuan: 1, required_tier: 'normal' } } },
        'GET /api/tenant/873472655125147648/billing/membership/': { membership: { tier: 'normal', cumulative_consumption_yuan: 0 } },
        'POST /api/tenant/873472655125147648/billing/orders/': ORDER,
      })
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await wrapper.find('input[type="number"]').setValue(1)
      await flushPromises()
      expect(wrapper.find('[data-testid="order-buyer-note"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('连点创建订单只发一次 POST 且带 Idempotency-Key', async () => {
      let releasePost
      const postGate = new Promise((resolve) => {
        releasePost = resolve
      })
      mocks.apiFetch.mockImplementation((url, options = {}) => {
        const method = String(options.method || 'GET').toUpperCase()
        if (method === 'POST') {
          return postGate.then(() => ({
            ok: true,
            status: 200,
            json: () => Promise.resolve(ORDER),
            traceId: '',
          }))
        }
        if (String(url).includes('order-pricing')) {
          return Promise.resolve({
            ok: true,
            status: 200,
            json: () => Promise.resolve({ pricing: { task_post: { price_yuan: 1, required_tier: 'normal' } } }),
            traceId: '',
          })
        }
        if (String(url).includes('membership')) {
          return Promise.resolve({
            ok: true,
            status: 200,
            json: () => Promise.resolve({ membership: { tier: 'normal', cumulative_consumption_yuan: 0 } }),
            traceId: '',
          })
        }
        return Promise.resolve({ ok: false, status: 404, json: () => Promise.resolve({}), traceId: '' })
      })
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await wrapper.find('input[type="number"]').setValue(1)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await createBtn.trigger('click')
      releasePost()
      await flushPromises()
      const posts = mocks.apiFetch.mock.calls.filter((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(posts).toHaveLength(1)
      expect(posts[0][1].headers['Idempotency-Key']).toBeTruthy()
      wrapper.unmount()
    })

    it('创建失败不跳转，错误带 data-traceId', async () => {
      mockApi({
        'GET /api/tenant/873472655125147648/billing/order-pricing/': { pricing: { task_post: { price_yuan: 1, required_tier: 'normal' } } },
        'GET /api/tenant/873472655125147648/billing/membership/': { membership: { tier: 'normal', cumulative_consumption_yuan: 0 } },
        'POST /api/tenant/873472655125147648/billing/orders/': { __fail: true, error: '余额不足', trace_id: 'trace-create-fail' },
      })
      const wrapper = mount(OrderCreate, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      await wrapper.find('input[type="number"]').setValue(1)
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建订单'))
      await createBtn.trigger('click')
      await flushPromises()
      expect(mocks.push).not.toHaveBeenCalled()
      const errEl = wrapper.find('[data-traceId]')
      expect(errEl.exists()).toBe(true)
      expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe('trace-create-fail')
      wrapper.unmount()
    })
  })
}
