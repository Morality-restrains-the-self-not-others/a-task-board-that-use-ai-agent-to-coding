/**
 * @vitest-environment jsdom
 * 定价页不再渲染「会员等级体系」描述区块。
 */
if (!process.env.VITEST) {
  console.log('[skip] Pricing.membership-section.unit.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
  }))

  const { default: Pricing } = await import('./Pricing.vue')

  const PRICING = {
    task_post: {
      name: '创建任务帖',
      price_yuan: '0.55',
      required_tier: 'normal',
      required_tier_label: '普通会员',
      min_consumption_cents: 0,
      description: '每个任务帖含 12 个月存续期',
    },
    gitlab_disk: {
      name: 'GitLab 磁盘',
      price_yuan: '1.00',
      required_tier: 'vip1',
      required_tier_label: 'VIP1',
      description: 'GitLab 仓库磁盘空间，起购 10 GB',
      min_quantity: 10,
    },
    gitlab_traffic: {
      name: 'GitLab 流量费',
      price_yuan: '0.10',
      required_tier: 'vip1',
      required_tier_label: 'VIP1',
      description: 'GitLab 外网流量（同区域内网不计费）',
      requires_label: 'GitLab 磁盘',
    },
    tiers: {
      normal: {
        name: '普通会员',
        description: '注册用户默认等级，可购买任务帖',
        can_purchase_labels: ['任务'],
      },
      vip1: {
        name: 'VIP1',
        description: '可购买全部资源：任务帖、GitLab 磁盘、GitLab 流量费',
        upgrade_threshold_yuan: '100.00',
        can_purchase_labels: ['任务', 'GitLab 磁盘', 'GitLab 流量费'],
      },
    },
  }

  function mockPricingOk() {
    mocks.apiFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ status: 'success', pricing: PRICING }),
      traceId: '',
    })
  }

  describe('Pricing 去掉会员等级体系描述', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mockPricingOk()
    })

    it('不渲染会员等级体系标题、介绍与两张会员卡描述', async () => {
      const wrapper = mount(Pricing)
      await flushPromises()
      const text = wrapper.text()
      expect(text).not.toContain('会员等级体系')
      expect(text).not.toContain('平台采用两级会员制度')
      expect(text).not.toContain('可购买资源：')
      expect(text).not.toContain('升级条件')
      expect(text).not.toContain('默认等级')
      expect(text).not.toContain('注册用户默认等级')
      expect(text).not.toContain('商品购买门槛')
      expect(text).not.toContain('以下各商品价格、所需会员等级及购买限制')
      expect(text).not.toContain('与购买门槛')
      expect(text).toContain('计费标准')
    })

    it('仍展示三项资源与购买说明', async () => {
      const wrapper = mount(Pricing)
      await flushPromises()
      const text = wrapper.text()
      expect(text).toContain('资源收费标准')
      expect(wrapper.get('[data-alias="view-pricing-page"]').exists()).toBe(true)
      expect(wrapper.findAll('h3').map((n) => n.text())).toEqual(
        expect.arrayContaining(['任务', 'GitLab 磁盘', 'GitLab 流量费']),
      )
      expect(text).toContain('购买与扣费说明')
      expect(text).toContain('0.55')
      expect(text).toContain('最低购买数量')
      expect(text).toContain('10 GB')
      expect(text).not.toContain('累计限购')
    })

    it('价格保留声明位于购买与扣费说明之后', async () => {
      const wrapper = mount(Pricing)
      await flushPromises()
      const html = wrapper.html()
      const noticeIdx = html.indexOf('data-testid="pricing-change-notice"')
      const notesIdx = html.indexOf('购买与扣费说明')
      const productsIdx = html.indexOf('元/帖/12个月')
      expect(noticeIdx).toBeGreaterThan(-1)
      expect(notesIdx).toBeGreaterThan(-1)
      expect(noticeIdx).toBeGreaterThan(notesIdx)
      expect(noticeIdx).toBeGreaterThan(productsIdx)
      expect(wrapper.get('[data-testid="pricing-change-notice"]').text()).toContain(
        '平台保留修改价格的权利',
      )
    })

    it('标题相对原位置上移 40px（-mt-10）', async () => {
      const wrapper = mount(Pricing)
      expect(wrapper.get('h1').classes()).toContain('-mt-10')
    })
  })
}
